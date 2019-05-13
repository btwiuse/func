package reconciler

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/func/func/resource/hash"
	"github.com/func/func/resource/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// job is a reconciliation job for a single namespace-project.
type job struct {
	sem      chan int
	ns       string
	graph    *graph.Graph
	existing *existingResources
	project  config.Project
	state    StateStorage
	source   SourceStorage
	logger   *zap.Logger
	backoff  func() backoff.BackOff

	mu      sync.Mutex
	process map[*graph.Resource]chan error
}

func (j *job) CreateUpdate(ctx context.Context) error {
	j.logger.Info("Creating/updating resources")

	var leaves []*graph.Resource
	for _, r := range j.graph.Resources() {
		if len(r.Dependents()) == 0 {
			leaves = append(leaves, r)
		}
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, l := range leaves {
		l := l
		g.Go(func() error {
			return <-j.processResource(ctx, l)
		})
	}

	return g.Wait()
}

// acquireSem acquires a semaphore to limit concurrency. The function blocks
// until allowed to execute and periodically logs updates. Returns with an
// error in case the context is cancelled.
func (j *job) acquireSem(ctx context.Context, logger *zap.Logger) (func(), error) {
	done := func() {
		<-j.sem
	}

	wait := 100 * time.Millisecond
	for {
		select {
		case j.sem <- 1:
			// Got semaphore
			return done, nil
		case <-ctx.Done():
			// Context cancelled
			return nil, ctx.Err()
		case <-time.After(wait):
			// Log update
			if wait < 5*time.Second {
				wait *= 2
			}
			logger.Debug("Waiting for semaphore")
		}
	}
}

func (j *job) waitForDeps(ctx context.Context, res *graph.Resource, logger *zap.Logger) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, dep := range res.Dependencies() {
		dep := dep
		g.Go(func() error {
			if err := j.waitForDep(ctx, dep, logger); err != nil {
				return errors.Wrap(err, "wait for dep")
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (j *job) waitForDep(ctx context.Context, dep *graph.Dependency, logger *zap.Logger) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, p := range dep.Parents() {
		p := p
		log := logger.With(
			zap.String("type", p.Config.Def.Type()),
			zap.String("name", p.Config.Name),
		)
		g.Go(func() error {
			log.Debug("Waiting on dependency")
			err := <-j.processResource(ctx, p)
			log.Debug("Dependency done", zap.Error(err))
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return errors.WithStack(err)
	}

	// All dependencies resolved, evaluate and set value.
	if err := evalDependency(dep); err != nil {
		return errors.Wrap(err, "eval")
	}

	return nil
}
func (j *job) processResource(ctx context.Context, res *graph.Resource) <-chan error {
	logger := j.logger.With(
		zap.String("type", res.Config.Def.Type()),
		zap.String("name", res.Config.Name),
	)

	if err := j.waitForDeps(ctx, res, logger); err != nil {
		errc := make(chan error, 1)
		errc <- errors.Wrap(err, "process deps")
		return errc
	}

	// Acquire Semaphore to limit concurrency.
	// This must be done after dependencies are resolved, otherwise
	// dependencies may not get a semaphore to run.
	done, err := j.acquireSem(ctx, logger)
	if err != nil {
		errc := make(chan error, 1)
		errc <- errors.Wrap(err, "acquire semaphore")
		return errc
	}
	defer done()

	j.mu.Lock()
	errc := j.process[res]
	if errc != nil {
		j.mu.Unlock()
		return errc
	}
	errc = make(chan error, 1)
	j.process[res] = errc
	j.mu.Unlock()

	defer func() {
		close(errc)
	}()

	hash := hash.Compute(res.Config.Def)

	logger.With(zap.String("config_hash", hash)).Info("Processing")

	srcs := res.Sources()
	sourceList := make([]resource.SourceCode, len(srcs))
	for i, src := range res.Sources() {
		sourceList[i] = &source{
			info:    src.Config,
			storage: j.source,
		}
		res.Config.Sources = append(res.Config.Sources, src.Config.Key)
		logger.Debug("Set source code", zap.String("sha", src.Config.Key))
	}

	ex := j.existing.Find(res.Config.Def.Type(), res.Config.Name)
	updateConfig := false
	updateSource := false
	if ex != nil {
		logger.Debug("Existing version of resource exists")
		j.existing.Keep(ex)

		// Copy outputs from previous value
		prevVal := reflect.Indirect(reflect.ValueOf(ex.res.Def))
		nextVal := reflect.Indirect(reflect.ValueOf(res.Config.Def))
		outputs := schema.Outputs(prevVal.Type())
		for _, output := range outputs {
			prev := prevVal.Field(output.Index)
			next := nextVal.Field(output.Index)
			next.Set(prev)
		}

		if ex.hash == hash {
			// Resource config did not change.
			logger.Debug("Configuration did not change")
		} else {
			// Resource config did change.
			logger.Debug("Update configuration")
			updateConfig = true
		}

		opts := []cmp.Option{
			cmpopts.SortSlices(func(a, b string) bool { return a < b }),
			cmpopts.EquateEmpty(),
		}
		updateSource = !cmp.Equal(ex.res.Sources, res.Config.Sources, opts...)

		logger.Debug("Compare source",
			zap.Bool("update", updateSource),
			zap.Strings("prev", ex.res.Sources),
			zap.Strings("next", res.Config.Sources),
		)
	}

	if ex != nil && !updateConfig && !updateSource {
		logger.Info("No changes necessary")
		// Nothing to do
		return errc
	}

	var op func() error

	if ex == nil {
		logger.Info("Creating resource")
		req := &resource.CreateRequest{
			Auth:   tempLocalAuthProvider{},
			Source: sourceList,
		}

		op = func() error {
			return res.Config.Def.Create(ctx, req)
		}
	} else {
		logger.Info("Updating resource")
		req := &resource.UpdateRequest{
			Auth:          tempLocalAuthProvider{},
			Source:        sourceList,
			Previous:      ex.res.Def,
			ConfigChanged: updateConfig,
			SourceChanged: updateSource,
		}

		op = func() error {
			return res.Config.Def.Update(ctx, req)
		}
	}

	algo := backoff.WithContext(j.backoff(), ctx)
	notify := func(err error, dur time.Duration) {
		logger.Info("Retrying", zap.Error(err), zap.Duration("duration", dur))
	}

	if err := backoff.RetryNotify(op, algo, notify); err != nil {
		// Log and return error. This handles the error twice but makes it
		// easier to pin-point what went wrong.
		logger.Error("Could not put resource", zap.Error(err), zap.Any("config", res.Config.Def))
		errc <- errors.Wrap(err, "put resource")
		return errc
	}

	// Collect dependencies that were used.
	for _, d := range res.Dependencies() {
		for _, p := range d.Parents() {
			res.Config.Deps = append(res.Config.Deps, resource.Dependency{
				Type: p.Config.Def.Type(),
				Name: p.Config.Name,
			})
		}
	}
	// Use new context so a cancelled context still stores the result.
	pctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	logger.Debug("Storing data")
	if err := j.state.Put(pctx, j.ns, j.project.Name, res.Config); err != nil {
		errc <- errors.Wrap(err, "store resource")
		return errc
	}

	return errc
}

func (j *job) Prune(ctx context.Context) error {
	rem := j.existing.Remaining()
	j.logger.Info("Removing previous resources", zap.Int("count", len(rem)))
	for _, e := range rem {
		logger := j.logger.With(
			zap.String("type", e.res.Def.Type()),
			zap.String("name", e.res.Name),
		)

		logger.Debug("Deleting resource")
		req := &resource.DeleteRequest{
			Auth: tempLocalAuthProvider{},
		}

		algo := backoff.WithContext(j.backoff(), ctx)

		err := backoff.Retry(func() error {
			return e.res.Def.Delete(ctx, req)
		}, algo)

		if err != nil {
			return errors.Wrap(err, "delete")
		}

		// Use new context so a cancelled context still stores the result.
		dctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		logger.Debug("Removing deleted resource from store")
		if err := j.state.Delete(dctx, j.ns, j.project.Name, e.res.Def.Type(), e.res.Name); err != nil {
			return errors.Wrap(err, "delete resource")
		}
	}
	return nil
}

package reconciler

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/imdario/mergo"
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
	for _, ref := range res.Dependencies() {
		dep := ref.Source.Resource
		log := logger.With(
			zap.String("type", dep.Config.Def.Type()),
			zap.String("name", dep.Config.Name),
		)
		g.Go(func() error {
			log.Debug("Waiting on dependency")
			err := <-j.processResource(ctx, dep)
			log.Debug("Dependency done", zap.Error(err))
			return err
		})
	}
	return g.Wait()
}

func (j *job) processResource(ctx context.Context, res *graph.Resource) <-chan error {
	logger := j.logger.With(
		zap.String("type", res.Config.Def.Type()),
		zap.String("name", res.Config.Name),
	)

	// Acquire Semaphore to limit concurrency
	done, err := j.acquireSem(ctx, logger)
	if err != nil {
		errc := make(chan error, 1)
		errc <- errors.Wrap(err, "acquire semaphore")
		return errc
	}
	defer done()

	if err := j.waitForDeps(ctx, res, logger); err != nil {
		errc := make(chan error, 1)
		errc <- errors.Wrap(err, "process deps")
		return errc
	}

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
		dd := res.Dependents()
		if len(dd) > 0 {
			logger.Debug("Updating dependents", zap.Int("cout", len(dd)))
			for _, ref := range dd {
				updateRef(ref)
			}
		}

		// Note: It is important to close _after_ updating refs as closing the
		// channel allows the flow to continue, which could cause a race
		// condition when the resource is being hashed (read) while refs being
		// updated (written).
		close(errc)
	}()

	hash := resource.Hash(res.Config.Def)

	logger.With(zap.String("config_hash", hash)).Info("Processing")

	srcs := res.Sources()
	sourceList := make([]resource.SourceCode, len(srcs))
	for i, src := range res.Sources() {
		sourceList[i] = &source{
			info:    src.Config,
			storage: j.source,
		}
		res.Config.Sources = append(res.Config.Sources, src.Config.SHA)
		logger.Debug("Set source code", zap.String("sha", src.Config.SHA))
	}

	ex := j.existing.Find(res.Config.Def.Type(), res.Config.Name)
	updateConfig := false
	updateSource := false
	if ex != nil {
		logger.Debug("Existing version of resource exists")
		j.existing.Keep(ex)

		if ex.hash == hash {
			// Resource config did not change.
			logger.Debug("Configuration did not change")
			// Set all dependent inputs from existing resource definition.
			for _, ref := range res.Dependents() {
				// Change ref source to deployed resource.
				ref.Source.Resource.Config.Def = ex.res.Def
			}
		} else {
			// Resource config did change.
			logger.Debug("Update configuration")
			updateConfig = true
			// Merge existing outputs into resource
			// Inputs set on the resource are not overwritten.
			if err := mergo.Merge(res.Config.Def, ex.res.Def); err != nil {
				errc <- errors.Wrap(err, "merge existing")
				return errc
			}
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

	if ex == nil {
		logger.Info("Creating resource")
		req := &resource.CreateRequest{
			Auth:   tempLocalAuthProvider{},
			Source: sourceList,
		}
		if err := res.Config.Def.Create(ctx, req); err != nil {
			// Log and return error. This handles the error twice but makes it
			// easier to pin-point what went wrong.
			logger.Error("Could not create resource", zap.Error(err), zap.Any("config", res.Config.Def))
			errc <- errors.Wrap(err, "create")
			return errc
		}
	} else {
		logger.Info("Updating resource")
		req := &resource.UpdateRequest{
			Auth:          tempLocalAuthProvider{},
			Source:        sourceList,
			Previous:      ex.res,
			ConfigChanged: updateConfig,
			SourceChanged: updateSource,
		}
		if err := res.Config.Def.Update(ctx, req); err != nil {
			// Log and return error. This handles the error twice but makes it
			// easier to pin-point what went wrong.
			logger.Error("Could not update resource", zap.Error(err), zap.Any("config", res.Config.Def))
			errc <- errors.Wrap(err, "update")
			return errc
		}
	}

	refs := res.Dependencies()
	if len(refs) > 0 {
		res.Config.Deps = make([]resource.Dependency, len(refs))
		for i, ref := range refs {
			res.Config.Deps[i] = resource.Dependency{
				Type: ref.Source.Resource.Config.Def.Type(),
				Name: ref.Source.Resource.Config.Name,
			}
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
		if err := e.res.Def.Delete(ctx, req); err != nil {
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

func updateRef(ref graph.Reference) {
	src := ref.Source.Value()
	dst := ref.Target.Value()
	srcType := src.Type()
	dstType := dst.Type()

	// Direct (or close enough; int32 -> int64 etc) match
	// string -> string
	if srcType.AssignableTo(dstType) {
		dst.Set(src)
		return
	}

	// Output Pointer to Input Value
	// *string -> string
	if src.Kind() == reflect.Ptr && src.Elem().Type() == dstType {
		dst.Set(src.Elem()) // Set value from pointer's underlying value
		return
	}

	// Output Value to Input Pointer
	// string -> *string
	if dstType.Kind() == reflect.Ptr && dstType.Elem() == srcType {
		ptr := reflect.New(dstType.Elem()) // Create new pointer
		ptr.Elem().Set(src)                // Set pointer value
		dst.Set(ptr)                       // Set destination to pointer
		return
	}

	// If the application ever reached this point, it is likely because input
	// validation was not performed correctly when the configs were parsed.
	panic(fmt.Sprintf("Cannot assign %s to %s", srcType, dstType))
}

package reconciler

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/cenkalti/backoff"
	"github.com/func/func/ctyext"
	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
	"github.com/func/func/resource/reconciler/internal/task"
	"github.com/func/func/resource/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// DefaultConcurrency is the default maximum concurrency to use.
//
// In practice, the reconciler is likely bound by network i/o.
var DefaultConcurrency = 10

// ResourceStorage persists resources.
type ResourceStorage interface {
	PutResource(ctx context.Context, project string, resource *resource.Resource) error
	DeleteResource(ctx context.Context, project string, resource *resource.Resource) error
	ListResources(ctx context.Context, project string) ([]*resource.Resource, error)
}

// SourceStorage provides resource source code.
type SourceStorage interface {
	Get(ctx context.Context, filename string) (io.ReadCloser, error)
}

// A Registry is able to provide type information for resources.
type Registry interface {
	Type(typename string) reflect.Type
}

// A Reconciler reconciles changes to a graph.
//
// See package doc for details.
type Reconciler struct {
	Resources ResourceStorage
	Source    SourceStorage
	Registry  Registry

	// Concurrency sets the maximum allowed concurrency to use.
	// If not set, DefaultConcurrency is used.
	Concurrency uint

	// Logger logs reconciliation updates. If not set, logs are discarded.
	Logger *zap.Logger

	// Backoff algorithm used for retries. If not set, exponential backoff is used.
	Backoff func() backoff.BackOff
}

// Reconcile reconciles changes to the graph.
func (r *Reconciler) Reconcile(ctx context.Context, id, proj string, graph *graph.Graph) error {
	logger := r.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	algo := r.Backoff
	if algo == nil {
		algo = func() backoff.BackOff {
			return backoff.NewExponentialBackOff()
		}
	}

	if id != "" {
		logger = logger.With(zap.String("id", id))
	}

	logger.Info("Reconcile", zap.String("project", proj))

	c := r.Concurrency
	if c == 0 {
		c = uint(DefaultConcurrency)
	}

	run := run{
		ID:        id,
		Project:   proj,
		Graph:     graph,
		Resources: r.Resources,
		Source:    r.Source,
		Registry:  r.Registry,
		Logger:    logger,
		Backoff:   algo,
		Sem:       semaphore.NewWeighted(int64(c)),
	}

	if err := run.GetExisting(ctx); err != nil {
		return errors.Wrap(err, "get existing resources")
	}

	if err := run.CreateUpdate(ctx); err != nil {
		return err
	}

	if err := run.RemovePrevious(ctx); err != nil {
		return errors.Wrap(err, "remove previous resources")
	}

	logger.Info(
		"Done",
		zap.Uint32("create", run.create),
		zap.Uint32("update", run.update),
		zap.Uint32("delete", run.delete),
	)

	return nil
}

type run struct {
	ID      string
	Project string
	Graph   *graph.Graph

	Resources ResourceStorage
	Source    SourceStorage
	Registry  Registry
	Logger    *zap.Logger
	Backoff   func() backoff.BackOff
	Sem       *semaphore.Weighted

	mu       sync.RWMutex
	existing []*resource.Resource // Existing resource from a previous deployment.

	tasks *task.Group // Maintains a list of actively processing resources.

	create, update, delete uint32
}

func (r *run) GetExisting(ctx context.Context) error {
	r.Logger.Debug("Get existing")
	ex, err := r.Resources.ListResources(ctx, r.Project)
	if err != nil {
		return errors.Wrap(err, "list")
	}
	r.existing = ex
	r.Logger.Debug("Got existing", zap.Int("count", len(ex)))
	return nil
}

func (r *run) CreateUpdate(ctx context.Context) error {
	r.Logger.Debug("Create/update")
	r.tasks = task.NewGroup()

	g, ctx := errgroup.WithContext(ctx)

	leaves := r.Graph.LeafResources()
	r.Logger.Debug("Leaf nodes", zap.Strings("names", leaves))
	for _, name := range leaves {
		res := r.Graph.Resources[name]
		g.Go(func() error {
			return r.processResource(ctx, res)
		})
	}

	return g.Wait()
}

func (r *run) processResource(ctx context.Context, res *resource.Resource) error {
	logger := r.Logger.With(zap.String("type", res.Type), zap.String("name", res.Name))

	return r.tasks.Do(res.Name, func() error {
		// Wait for dependencies to resolve.
		// Do this before acquiring a semaphore, as otherwise we can needlessly
		// block on low concurrency limits, and end up in a deadlock with
		// concurrency=1.
		if err := r.processDependencies(ctx, res.Name, logger); err != nil {
			return errors.Wrap(err, "process dependencies")
		}

		// Ready to process, wait for semaphore.
		err := r.Sem.Acquire(ctx, 1)
		if err != nil {
			return errors.Wrap(err, "acquire semaphore")
		}
		defer r.Sem.Release(1)

		// Create definition
		defType := r.Registry.Type(res.Type)
		if defType == nil {
			return errors.Errorf("type not registered: %q", res.Type)
		}

		if err := r.resolveDependencies(res); err != nil {
			return errors.Wrap(err, "resolve dependencies")
		}

		logger.Debug("Processing")

		// Compute hash based on current inputs.
		hash := res.Input.Hash()
		logger = logger.With(zap.Int("hash", hash))

		// Insert config into definition.
		val := reflect.New(defType)
		if err := ctyext.FromCtyValue(res.Input, val.Interface(), schema.FieldName); err != nil {
			return errors.Wrap(err, "set input")
		}
		def := val.Elem().Interface().(resource.Definition)

		logger.Debug("Config resolved")

		// Collect sources.
		sourceList := make([]resource.SourceCode, len(res.Sources))
		for i, src := range res.Sources {
			sourceList[i] = &source{key: src, storage: r.Source}
			n := len(src)
			if n > 7 {
				n = 7
			}
			logger.Debug("Set source code", zap.String("key", src[:n]))
		}

		// Find existing.
		r.mu.Lock()
		var existing *resource.Resource
		for i, ex := range r.existing {
			if ex.Type == res.Type && ex.Name == res.Name {
				existing = ex

				// Remove existing so it doesn't get deleted
				r.existing = append(r.existing[:i], r.existing[i+1:]...)
				break
			}
		}
		r.mu.Unlock()

		// Check what (if anything) needs to be updated.
		updateSource, updateConfig := false, false
		if existing != nil {
			exHash := existing.Input.Hash()
			logger.Debug("Existing version of resource exists")
			updateConfig = exHash != hash
			opts := []cmp.Option{
				cmpopts.SortSlices(func(a, b string) bool { return a < b }),
				cmpopts.EquateEmpty(),
			}
			updateSource = !cmp.Equal(existing.Sources, res.Sources, opts...)

			if updateConfig {
				logger.Debug("Config changed", zap.Int("prev_hash", exHash))
			}
			if updateSource {
				logger.Debug("Source changed", zap.Strings("prev_source", existing.Sources))
			}

			if !updateConfig && !updateSource {
				res.Output = existing.Output
				logger.Debug("No changes required")
				return nil
			}
		}

		var op func() error

		if existing != nil {
			logger.Info("Updating resource")

			// Create previous definition.
			val := reflect.New(r.Registry.Type(res.Type))
			if err := ctyext.FromCtyValue(existing.Output, val.Interface(), schema.FieldName); err != nil {
				return errors.Wrap(err, "set existing output")
			}
			if err := ctyext.FromCtyValue(existing.Input, val.Interface(), schema.FieldName); err != nil {
				return errors.Wrap(err, "set config")
			}
			prev := val.Elem().Interface().(resource.Definition)

			req := &resource.UpdateRequest{
				Auth:          tempLocalAuthProvider{},
				Source:        sourceList,
				Previous:      prev,
				ConfigChanged: updateConfig,
				SourceChanged: updateSource,
			}

			op = func() error {
				return def.Update(ctx, req)
			}
		} else {
			logger.Info("Creating resource")
			req := &resource.CreateRequest{
				Auth:   tempLocalAuthProvider{},
				Source: sourceList,
			}

			op = func() error {
				return def.Create(ctx, req)
			}
		}

		algo := backoff.WithContext(r.Backoff(), ctx)
		notify := func(err error, dur time.Duration) {
			logger.Info("Retrying", zap.Error(err), zap.Duration("duration", dur))
		}

		if err := backoff.RetryNotify(op, algo, notify); err != nil {
			opStr := "create"
			if existing != nil {
				opStr = "update"
			}
			return errors.Wrap(err, fmt.Sprintf("%s %s.%s", opStr, res.Type, res.Name))
		}

		// Capture generated output values
		if err := setOutput(res, def); err != nil {
			return errors.Wrap(err, "set output")
		}

		// Use new context so a cancelled context still stores the result.
		pctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		logger.Debug("Storing data")
		if err := r.Resources.PutResource(pctx, r.Project, res); err != nil {
			return errors.Wrap(err, "store resource")
		}

		if existing != nil {
			atomic.AddUint32(&r.update, 1)
		} else {
			atomic.AddUint32(&r.create, 1)
		}

		return nil
	})
}

func setOutput(res *resource.Resource, def resource.Definition) error {
	ty := reflect.TypeOf(def)
	outputType := schema.Fields(ty).Outputs().CtyType()
	outputs, err := ctyext.ToCtyValue(def, outputType, schema.FieldName)
	if err != nil {
		return errors.Wrap(err, "convert output values")
	}
	res.Output = outputs
	return nil
}

func (r *run) processDependencies(ctx context.Context, childName string, logger *zap.Logger) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, dep := range r.Graph.Dependencies[childName] {
		for _, parent := range dep.Parents() {
			res, ok := r.Graph.Resources[parent]
			if !ok {
				return fmt.Errorf("dependency on non-existing resource %q", parent)
			}
			logger.Debug("Waiting on dependency", zap.String("parent", parent))
			g.Go(func() error {
				err := r.processResource(ctx, res)
				logger.Debug("Dependency done", zap.String("parent", parent), zap.Bool("error", err != nil))
				return err
			})
		}
	}
	return g.Wait()
}

func (r *run) resolveDependencies(res *resource.Resource) error {
	deps := r.Graph.Dependencies[res.Name]

	if len(deps) == 0 {
		return nil
	}

	vars := make(map[string]cty.Value)
	for _, dep := range deps {
		for _, p := range dep.Parents() {
			vars[p] = r.Graph.Resources[p].Output
		}
	}

	ctx := &graph.EvalContext{Variables: vars}
	for _, dep := range deps {
		processed := false
		cfg, err := cty.Transform(res.Input, func(path cty.Path, val cty.Value) (cty.Value, error) {
			if !path.Equals(dep.Field) {
				return val, nil
			}
			v, err := dep.Expression.Value(ctx)
			if err != nil {
				return cty.NilVal, errors.Wrap(err, "eval expression")
			}
			processed = true
			return v, nil
		})
		if err != nil {
			return errors.Wrap(err, "transform input with dependencies")
		}
		if !processed {
			return fmt.Errorf("dependency %s was not found", ctyext.PathString(dep.Field))
		}
		res.Input = cfg
	}

	return nil
}

func (r *run) RemovePrevious(ctx context.Context) error {
	if len(r.existing) == 0 {
		r.Logger.Debug("No previous resources to remove")
		return nil
	}
	r.Logger.Debug("Remove previous")
	wgs := make(map[string]*sync.WaitGroup, len(r.existing))
	for _, res := range r.existing {
		for _, dep := range res.Deps {
			wg, ok := wgs[dep]
			if !ok {
				wg = &sync.WaitGroup{}
				wgs[dep] = wg
			}
			wg.Add(1)
		}
	}
	g, ctx := errgroup.WithContext(ctx)
	for _, res := range r.existing {
		res := res
		g.Go(func() error {
			if wg, ok := wgs[res.Name]; ok {
				// Wait for all dependents.
				wg.Wait()
			}
			err := r.removeResource(ctx, res)
			for _, dep := range res.Deps {
				pwg := wgs[dep]
				pwg.Done()
			}
			return err
		})
	}
	return g.Wait()
}

func (r *run) removeResource(ctx context.Context, res *resource.Resource) error {
	logger := r.Logger.With(zap.String("type", res.Type), zap.String("name", res.Name))

	// Ready to process, wait for semaphore.
	err := r.Sem.Acquire(ctx, 1)
	if err != nil {
		return errors.Wrap(err, "acquire semaphore")
	}
	defer r.Sem.Release(1)

	logger.Debug("Delete")

	// Create previous definition.
	val := reflect.New(r.Registry.Type(res.Type))
	if err := ctyext.FromCtyValue(res.Output, val.Interface(), schema.FieldName); err != nil {
		return errors.Wrap(err, "set existing output")
	}
	if err := ctyext.FromCtyValue(res.Input, val.Interface(), schema.FieldName); err != nil {
		return errors.Wrap(err, "set config")
	}
	def := val.Elem().Interface().(resource.Definition)

	req := &resource.DeleteRequest{Auth: tempLocalAuthProvider{}}
	err = backoff.RetryNotify(
		func() error {
			return def.Delete(ctx, req)
		},
		backoff.WithContext(r.Backoff(), ctx),
		func(err error, dur time.Duration) {
			logger.Info("Retrying", zap.Error(err), zap.Duration("duration", dur))
		},
	)
	if err != nil {
		return errors.Wrap(err, "delete")
	}

	atomic.AddUint32(&r.delete, 1)

	// Use new context so a cancelled context still stores the result.
	pctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Debug("Deleting data")
	if err := r.Resources.DeleteResource(pctx, r.Project, res); err != nil {
		return errors.Wrap(err, "delete resource")
	}

	return nil
}

type source struct {
	key     string
	storage SourceStorage
}

func (s *source) Key() string { return s.key }
func (s *source) Reader(ctx context.Context) (targz io.ReadCloser, err error) {
	return s.storage.Get(ctx, s.key)
}

type tempLocalAuthProvider struct{}

func (p tempLocalAuthProvider) AWS() (aws.CredentialsProvider, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	return cfg.Credentials, nil
}

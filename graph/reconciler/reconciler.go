package reconciler

import (
	"context"
	"io"
	"runtime"

	"github.com/cenkalti/backoff"
	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

// DefaultConcurrency is the default maximum concurrency to use.
var DefaultConcurrency = runtime.NumCPU() * 2

// StateStorage persists resources.
type StateStorage interface {
	// Put creates or updates a resource.
	// The type & name are used to match the resource.
	Put(ctx context.Context, namespace, project string, resource resource.Resource) error

	// Delete removes a resource.
	Delete(ctx context.Context, namespace, project, typename, name string) error

	// List returns all resources for a given project. Resources may be
	// returned in any order.
	List(ctx context.Context, namespace, project string) ([]resource.Resource, error)
}

// SourceStorage provides resource source code.
type SourceStorage interface {
	Get(ctx context.Context, filename string) (io.ReadCloser, error)
}

// A Reconciler reconciles changes to a graph.
//
// See package doc for details.
type Reconciler struct {
	// Concurrency sets the maximum allowed concurrency to use.
	// If not set, DefaultConcurrency is used.
	Concurrency int
	State       StateStorage
	Source      SourceStorage

	// Logger logs reconciliation updates. If not set, logs are discarded.
	Logger *zap.Logger

	// Backoff is the backoff algorithm used for retries. If not set,
	// exponential backoff is used.
	Backoff func() backoff.BackOff
}

// Reconcile reconciles changes to the graph.
func (r *Reconciler) Reconcile(ctx context.Context, ns string, project config.Project, desired *graph.Graph) error {
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

	logger = logger.With(
		zap.String("ns", ns),
		zap.String("project", project.Name),
		zap.String("job_id", ksuid.New().String()),
	)

	logger.Info("Reconcile")

	c := r.Concurrency
	if c == 0 {
		c = DefaultConcurrency
	}
	logger.Debug("Set concurrency", zap.Int("max", c))

	rr, err := r.State.List(ctx, ns, project.Name)
	if err != nil {
		return errors.Wrap(err, "list existing resources")
	}

	logger.Debug("Received existing resources", zap.Int("count", len(rr)))

	existing, err := newExisting(rr)
	if err != nil {
		return errors.Wrap(err, "create existing graph")
	}

	j := &job{
		sem:      make(chan int, c),
		ns:       ns,
		graph:    desired,
		existing: existing,
		project:  project,
		state:    r.State,
		source:   r.Source,
		process:  make(map[*graph.Resource]chan error),
		logger:   logger,
		backoff:  algo,
	}

	// Create/update resources.
	if err := j.CreateUpdate(ctx); err != nil {
		return errors.Wrap(err, "create")
	}

	// Clean up no old resources.
	if err := j.Prune(ctx); err != nil {
		return errors.Wrap(err, "prune")
	}

	logger.Info("Done")

	return nil
}

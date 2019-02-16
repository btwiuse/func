package reconciler

import (
	"context"
	"io"
	"runtime"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// DefaultConcurrency is the default maximum concurrency to use.
var DefaultConcurrency = runtime.NumCPU() * 2

// StateStorage persists resources.
type StateStorage interface {
	// Put creates or updates a resource.
	// The type & name are used to match the resource.
	Put(ctx context.Context, namespace, project string, resource resource.Resource) error

	// Delete removes a resource.
	Delete(ctx context.Context, namespace, project, name string) error

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
}

// Reconcile reconciles changes to the graph.
func (r *Reconciler) Reconcile(ctx context.Context, ns string, project config.Project, desired *graph.Graph) error {
	c := r.Concurrency
	if c == 0 {
		c = DefaultConcurrency
	}

	rr, err := r.State.List(ctx, ns, project.Name)
	if err != nil {
		return errors.Wrap(err, "list existing resources")
	}

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
		process:  make(map[*graph.Resource]chan error),
	}

	// Create/update resources.
	if err := j.CreateUpdate(ctx); err != nil {
		return errors.Wrap(err, "create")
	}

	// Clean up no old resources.
	if err := j.Prune(ctx); err != nil {
		return errors.Wrap(err, "prune")
	}

	return nil
}

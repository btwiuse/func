package core

import (
	"context"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/func/func/source"
	"go.uber.org/zap"
)

// API is the core API that must be implemented by all intermediate clients.
type API interface {
	Apply(context.Context, *ApplyRequest) (*ApplyResponse, error)
}

// A Reconciler reconciles changes to the graph.
type Reconciler interface {
	Reconcile(ctx context.Context, ns string, project config.Project, graph *graph.Graph) error
}

// A ResourceRegistry is used for matching resource type names to resource
// implementations.
type ResourceRegistry interface {
	New(typename string) (resource.Definition, error)
	SuggestType(typename string) string
}

// Func implements the core business logic.
type Func struct {
	Logger     *zap.Logger
	Source     source.Storage
	Resources  ResourceRegistry
	Reconciler Reconciler
}

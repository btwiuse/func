package api

import (
	"context"
	"reflect"

	"github.com/func/func/resource"
	"github.com/func/func/resource/reconciler"
	"github.com/func/func/source"
	"go.uber.org/zap"
)

// A Reconciler reconciles changes to the graph.
type Reconciler interface {
	Reconcile(ctx context.Context, id, project string, graph reconciler.Graph) error
}

// Storage persists resolved graphs.
type Storage interface {
	PutGraph(ctx context.Context, project string, g *resource.Graph) error
}

// A Registry is used for matching resource type names to resource
// implementations.
type Registry interface {
	Type(typename string) reflect.Type
	Typenames() []string
}

// A Validator validates user input.
type Validator interface {
	Validate(input interface{}, rule string) error
}

// Server provides the serverside api implementation.
type Server struct {
	Logger    *zap.Logger
	Registry  Registry
	Source    source.Storage
	Storage   Storage
	Validator Validator

	// If set, reconciliation is done synchronously.
	Reconciler Reconciler
}

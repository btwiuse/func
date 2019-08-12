package api

import (
	"context"
	"reflect"

	"github.com/func/func/resource/graph"
	"github.com/func/func/source"
	"go.uber.org/zap"
)

// API is the core interface for the API. The API is implemented by Func.
type API interface {
	Apply(context.Context, *ApplyRequest) (*ApplyResponse, error)
}

// A Reconciler reconciles changes to the graph.
type Reconciler interface {
	Reconcile(ctx context.Context, id, ns, project string, graph *graph.Graph) error
}

// A ResourceRegistry is used for matching resource type names to resource
// implementations.
type ResourceRegistry interface {
	Type(typename string) reflect.Type
	Types() []string
}

// A Validator validates user input.
type Validator interface {
	Validate(input interface{}, rule string) error
}

// Func implements the core business logic.
type Func struct {
	Logger     *zap.Logger
	Source     source.Storage
	Resources  ResourceRegistry
	Validator  Validator
	Reconciler Reconciler
}

package resource

import (
	"context"
)

// A Definition describes a resource.
//
// All resources must implement this interface.
type Definition interface {
	Create(ctx context.Context, req *CreateRequest) error
	Update(ctx context.Context, req *UpdateRequest) error
	Delete(ctx context.Context, req *DeleteRequest) error
}

// A Resource is an instance of a resource supplied by the user.
type Resource struct {
	Name string     // Name used in resource config.
	Type string     // Type used in resource config.
	Def  Definition // User specified configuration for resource.

	// Deps contains the names of the resources that are dependencies of this
	// resources, that is, one or more field refers to an input or an output in
	// it.
	//
	// Deps are used for traversing the graph backwards when deleting resources.
	Deps []string

	// Sources contain the source code hashes that were provided to the
	// resource. The value is only set for resources that have been created.
	Sources []string
}

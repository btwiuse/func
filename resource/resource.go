package resource

import (
	"context"

	"github.com/zclconf/go-cty/cty"
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
	Name string // Name used in resource config.
	Type string // Type used in resource config.

	// Input is the user specified static configuration for the resource. The
	// shape of this field will depend on the Type. When creating resources,
	// the creator is responsible for only setting data that is valid for the
	// given resource type.
	Input cty.Value

	// Output contains the outputs from the resource. The value is set after
	// the resource has been provisioned.
	Output cty.Value

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

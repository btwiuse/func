package resource

import (
	"github.com/zclconf/go-cty/cty"
)

// Desired represents the desired state of a resource.
type Desired struct {
	// Name used in resource config.
	//
	// The Name uniquely identifies the resource.
	Name string

	// Type used in resource config.
	//
	// The Type determines how to process the resource.
	Type string

	// Input is the user specified static configuration for the resource. The
	// shape of this field will depend on the Type. When creating resources,
	// the creator is responsible for only setting data that is valid for the
	// given resource type.
	Input cty.Value

	// Sources contain the source code hashes that were provided to the
	// resource. The value is only set for resources that have been created.
	Sources []string
}

// Deployed is a deployed resource.
type Deployed struct {
	// Desired state that resulted in the deployed resource.
	*Desired

	// Output contains the outputs from the resource. The value is set after
	// the resource has been provisioned.
	Output cty.Value

	// Deps contains the names of the resources that are dependencies of this
	// resources, that is, one or more field refers to an input or an output in
	// it.
	//
	// Deps are used for traversing the graph backwards when deleting resources.
	Deps []string
}

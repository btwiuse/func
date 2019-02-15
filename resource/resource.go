package resource

import "fmt"

// A Definition describes a resource.
//
// All resources must implement this interface.
type Definition interface {
	// Type returns the type name for the resource.
	//
	// The name will be used for matching the resource to the resource
	// configuration provided by the user.
	Type() string
}

// A Resource is an instance of a resource supplied by the user.
type Resource struct {
	Name string     // Name used in resource config.
	Def  Definition // Def is the resolved definition for resource, including user data.

	// Deps contain the dependencies of the resource that were used
	// when creating the resource. The value is only set after reading
	// resources from storage.
	Deps []Dependency
}

// A Dependency describes a resource dependency.
type Dependency struct {
	Type, Name string
}

func (d Dependency) String() string { return fmt.Sprintf("\"%s:%s\"", d.Type, d.Name) }

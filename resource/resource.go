package resource

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
}

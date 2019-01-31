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

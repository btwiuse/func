package resource

// A Resource provisions remote resources.
type Resource interface {
	// Type returns the type name for the resource.
	//
	// The name will be used for matching the resource to the resource
	// configuration provided by the user.
	Type() string
}

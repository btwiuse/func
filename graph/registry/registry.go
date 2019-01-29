package registry

import (
	"fmt"
	"reflect"

	"github.com/agext/levenshtein"
	"github.com/func/func/graph"
)

// NotSupportedError is returned when attempting to instantiate an unsupported
// resource.
type NotSupportedError struct {
	Type string
}

// NotSupported is a no-op method that allows the error to be asserted as an
// interface, rather than importing the registry package.
func (e NotSupportedError) NotSupported() {}

// Error implements error.
func (e NotSupportedError) Error() string { return fmt.Sprintf("resource %q not supported", e.Type) }

// A Registry maintains a list of registered resources.
type Registry struct {
	resources map[string]reflect.Type
}

// FromResources creates a new registry from a predefined list of resources. It
// should primarily used in tests to set up a registry.
func FromResources(resources ...graph.Resource) *Registry {
	r := &Registry{}
	for _, res := range resources {
		r.Register(res)
	}
	return r
}

// Register adds a new resource type.
//
// The graph.Resource interface must be implemented on a pointer receiver on a
// struct. Panics otherwise. If another resource with the same type is already
// registered, it is overwritten.
//
// Not safe for concurrent access.
func (r *Registry) Register(resource graph.Resource) {
	t := reflect.TypeOf(resource)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("Resource must be implemented on a pointer receiver on a struct, not %s", t))
	}

	if r.resources == nil {
		r.resources = make(map[string]reflect.Type)
	}

	typename := resource.Type()
	r.resources[typename] = t.Elem()
}

// New creates a new instance of a resource with the given type name. Returns
// NotSupportedError if a matching type is not found.
func (r *Registry) New(typename string) (graph.Resource, error) {
	t, ok := r.resources[typename]
	if !ok {
		return nil, NotSupportedError{Type: typename}
	}

	return reflect.New(t).Interface().(graph.Resource), nil
}

// SuggestType suggest the type of a provisioner that closely matches the
// requested name. Returns an empty string if no close match was found.
func (r *Registry) SuggestType(typename string) string {
	// Maximum characters that can differ
	maxDist := 5

	var str string
	dist := maxDist + 1

	for name := range r.resources {
		d := levenshtein.Distance(typename, name, nil)
		if d < dist {
			str = name
			dist = d
		}
	}

	if dist > maxDist {
		// Suggestion is very different, don't give it at all
		return ""
	}

	return str
}

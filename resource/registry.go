package resource

import (
	"fmt"
	"reflect"
	"sort"
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

// RegistryFromResources creates a new registry from a predefined list of
// resources. It should primarily used in tests to set up a registry.
func RegistryFromResources(defs map[string]Definition) *Registry {
	r := &Registry{}
	for n, def := range defs {
		r.Register(n, def)
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
func (r *Registry) Register(typename string, def Definition) {
	t := reflect.TypeOf(def)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("Resource must be implemented on a pointer receiver on a struct, not %s", t))
	}

	if r.resources == nil {
		r.resources = make(map[string]reflect.Type)
	}

	r.resources[typename] = t.Elem()
}

// New creates a new instance of a resource with the given type name. Returns
// NotSupportedError if a matching type is not found.
func (r *Registry) New(typename string) (Definition, error) {
	t, ok := r.resources[typename]
	if !ok {
		return nil, NotSupportedError{Type: typename}
	}

	return reflect.New(t).Interface().(Definition), nil
}

// Types returns the type names that have been registered. The results are
// lexicographically sorted.
func (r *Registry) Types() []string {
	tt := make([]string, 0, len(r.resources))
	for k := range r.resources {
		tt = append(tt, k)
	}
	sort.Strings(tt)
	return tt
}

package resource

import (
	"reflect"
	"sort"
)

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
	if r.resources == nil {
		r.resources = make(map[string]reflect.Type)
	}
	r.resources[typename] = t
}

// Type returns the registered type with a certain name. Returns nil if the
// type has not been registered.
func (r *Registry) Type(typename string) reflect.Type {
	return r.resources[typename]
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

package resource

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/agext/levenshtein"
	"github.com/pkg/errors"
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
func RegistryFromResources(defs ...Definition) *Registry {
	r := &Registry{}
	for _, def := range defs {
		r.Register(def)
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
func (r *Registry) Register(def Definition) {
	t := reflect.TypeOf(def)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("Resource must be implemented on a pointer receiver on a struct, not %s", t))
	}

	if r.resources == nil {
		r.resources = make(map[string]reflect.Type)
	}

	typename := def.Type()
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

type envelope struct {
	Type string          `json:"t"`
	Data json.RawMessage `json:"d"`
}

// Marshal marshals the given resource to a byte slice. The byte slice can be
// unmarshalled back to a Resource using Registry.Unmarshal.
//
// The resource is marshalled using json encoding, meaning `json` struct tags
// on the resource will be used. By convention, struct tags should not be set.
// However, if the struct tags are set, they cannot be changed to ensure
// backwards compatibility.
func Marshal(def Definition) ([]byte, error) {
	j, err := json.Marshal(def)
	if err != nil {
		return nil, errors.Wrap(err, "marshal config")
	}
	e := envelope{
		Type: def.Type(),
		Data: j,
	}
	j, err = json.Marshal(e)
	if err != nil {
		return nil, errors.Wrap(err, "marshal envelope")
	}
	return j, nil
}

// Unmarshal unmarshals a given byte slice to a resource definition.
//
// The resource can only be unmarshalled if the corresponding resource has been
// registered.
func (r *Registry) Unmarshal(b []byte) (Definition, error) {
	var e envelope
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, errors.Wrap(err, "unmarshal envelope")
	}
	res, err := r.New(e.Type)
	if err != nil {
		return nil, errors.Wrap(err, "create resource")
	}
	if err := json.Unmarshal(e.Data, &res); err != nil {
		return nil, errors.Wrap(err, "unmarshal config")
	}
	return res, nil
}

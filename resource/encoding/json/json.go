package json

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// The Registry is used to get the type of resources for decoding the definition.
type Registry interface {
	Type(name string) reflect.Type
}

// Encoder encodes and decodes resources using json encoding.
type Encoder struct {
	Registry Registry
}

type jsonResource struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Deps    []string        `json:"deps,omitempty"`
	Sources []string        `json:"srcs,omitempty"`
	Def     json.RawMessage `json:"def"`
}

// MarshalResource marshals a resource to json.
func (enc *Encoder) MarshalResource(res resource.Resource) ([]byte, error) {
	def, err := json.Marshal(res.Def)
	if err != nil {
		return nil, errors.Wrap(err, "marshal definition")
	}
	r := jsonResource{
		Name:    res.Name,
		Type:    res.Type,
		Deps:    res.Deps,
		Sources: res.Sources,
		Def:     def,
	}
	return json.Marshal(r)
}

// UnmarshalResource unmarshals a resource from json.
//
// The type embedded in the resource byte slice must be available in the
// registry.
func (enc *Encoder) UnmarshalResource(b []byte) (resource.Resource, error) {
	var res jsonResource
	if err := json.Unmarshal(b, &res); err != nil {
		return resource.Resource{}, errors.Wrap(err, "unmarshal")
	}
	t := enc.Registry.Type(res.Type)
	if t == nil {
		return resource.Resource{}, fmt.Errorf("type not registered: %q", res.Type)
	}

	v := reflect.New(t)
	if err := json.Unmarshal(res.Def, v.Interface()); err != nil {
		return resource.Resource{}, errors.Wrap(err, "unmarshal")
	}
	def := v.Interface().(resource.Definition)

	return resource.Resource{
		Name:    res.Name,
		Type:    res.Type,
		Deps:    res.Deps,
		Sources: res.Sources,
		Def:     def,
	}, nil
}

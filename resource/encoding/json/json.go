package json

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/func/func/resource"
	"github.com/func/func/resource/schema"
	"github.com/pkg/errors"
	ctyjson "github.com/zclconf/go-cty/cty/json"
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
	Input   json.RawMessage `json:"input"`
	Output  json.RawMessage `json:"output"`
}

// MarshalResource marshals a resource to json.
func (enc *Encoder) MarshalResource(res resource.Resource) ([]byte, error) {
	input, err := ctyjson.Marshal(res.Input, res.Input.Type())
	if err != nil {
		return nil, errors.Wrap(err, "marshal data")
	}
	output, err := ctyjson.Marshal(res.Output, res.Output.Type())
	if err != nil {
		return nil, errors.Wrap(err, "marshal data")
	}
	return json.Marshal(jsonResource{
		Name:    res.Name,
		Type:    res.Type,
		Deps:    res.Deps,
		Sources: res.Sources,
		Input:   input,
		Output:  output,
	})
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
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	fields := schema.Fields(t)

	it := fields.Inputs().CtyType()
	input, err := ctyjson.Unmarshal(res.Input, it)
	if err != nil {
		return resource.Resource{}, errors.Wrap(err, "unmarshal input")
	}

	ot := fields.Outputs().CtyType()
	output, err := ctyjson.Unmarshal(res.Output, ot)
	if err != nil {
		return resource.Resource{}, errors.Wrap(err, "unmarshal output")
	}

	return resource.Resource{
		Name:    res.Name,
		Type:    res.Type,
		Deps:    res.Deps,
		Sources: res.Sources,
		Input:   input,
		Output:  output,
	}, nil
}

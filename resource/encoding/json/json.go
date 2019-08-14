package json

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/func/func/resource"
	"github.com/func/func/resource/schema"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
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
	Input   json.RawMessage `json:"input,omitempty"`
	Output  json.RawMessage `json:"output,omitempty"`
}

// MarshalResource marshals a resource to json.
func (enc Encoder) MarshalResource(res resource.Resource) ([]byte, error) {
	typ := enc.Registry.Type(res.Type)
	if typ == nil {
		return nil, fmt.Errorf("type not registered")
	}
	fields := schema.Fields(typ)

	var input json.RawMessage
	if !res.Input.IsNull() {
		v, err := ctyjson.Marshal(cty.UnknownAsNull(res.Input), fields.Inputs().CtyType())
		if err != nil {
			return nil, errors.Wrap(err, "marshal input")
		}
		input = v
	}

	var output json.RawMessage
	if !res.Output.IsNull() {
		v, err := ctyjson.Marshal(res.Output, fields.Outputs().CtyType())
		if err != nil {
			return nil, errors.Wrap(err, "marshal output")
		}
		output = v
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
func (enc Encoder) UnmarshalResource(b []byte) (resource.Resource, error) {
	var res jsonResource
	if err := json.Unmarshal(b, &res); err != nil {
		return resource.Resource{}, errors.Wrap(err, "unmarshal")
	}

	typ := enc.Registry.Type(res.Type)
	if typ == nil {
		return resource.Resource{}, fmt.Errorf("type not registered")
	}
	fields := schema.Fields(typ)

	input := cty.EmptyObjectVal
	if len(res.Input) > 0 {
		v, err := ctyjson.Unmarshal(res.Input, fields.Inputs().CtyType())
		if err != nil {
			return resource.Resource{}, errors.Wrap(err, "unmarshal input")
		}
		input = v
	}

	var output cty.Value
	if len(res.Output) > 0 {
		v, err := ctyjson.Unmarshal(res.Output, fields.Outputs().CtyType())
		if err != nil {
			return resource.Resource{}, errors.Wrap(err, "unmarshal output")
		}
		output = v
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

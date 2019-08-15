package resource

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// A Resource is an instance of a resource supplied by the user.
type Resource struct {
	// Name used in resource config.
	//
	// The Name uniquely identifies the resource.
	Name string

	// Type used in resource config.
	//
	// The Type determines how to process the resource.
	Type string

	// Input is the user specified static configuration for the resource. The
	// shape of this field will depend on the Type. When creating resources,
	// the creator is responsible for only setting data that is valid for the
	// given resource type.
	Input cty.Value

	// Output contains the outputs from the resource. The value is set after
	// the resource has been provisioned.
	Output cty.Value

	// Deps contains the names of the resources that are dependencies of this
	// resources, that is, one or more field refers to an input or an output in
	// it.
	//
	// Deps are used for traversing the graph backwards when deleting resources.
	Deps []string

	// Sources contain the source code hashes that were provided to the
	// resource. The value is only set for resources that have been created.
	Sources []string
}

type jsonResource struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Input   json.RawMessage `json:"input,omitempty"`
	Output  json.RawMessage `json:"output,omitempty"`
	Deps    []string        `json:"deps,omitempty"`
	Sources []string        `json:"src,omitempty"`
}

// MarshalJSON marshals the resource to a json encoded string.
//
// Any unknown inputs are encoded as null.
func (r Resource) MarshalJSON() ([]byte, error) {
	var input json.RawMessage
	if !r.Input.IsNull() {
		iv := cty.UnknownAsNull(r.Input)
		v, err := ctyjson.Marshal(iv, iv.Type())
		if err != nil {
			return nil, errors.Wrap(err, "marshal input")
		}
		input = v
	}

	var output json.RawMessage
	if !r.Output.IsNull() {
		v, err := ctyjson.Marshal(r.Output, r.Output.Type())
		if err != nil {
			return nil, errors.Wrap(err, "marshal output")
		}
		output = v
	}

	return json.Marshal(jsonResource{
		Name:    r.Name,
		Type:    r.Type,
		Input:   input,
		Output:  output,
		Deps:    r.Deps,
		Sources: r.Sources,
	})
}

// UnmarshalJSON unmarshals a json encoded byte slice into the resource.
//
// If the json does have input or output defined, the resource MUST have a
// corresponding value, which determines the type and thus how to decode the
// json value. The value may be cty.Unknown.
//
//   r := &Resource{
//       Input:  cty.UnknownVal(inputType),
//       Output: cty.UnknownVal(outputType),
//   }
//   err := r.UnmarshalJSON(bytes)
func (r *Resource) UnmarshalJSON(b []byte) error {
	var res jsonResource
	if err := json.Unmarshal(b, &res); err != nil {
		return errors.Wrap(err, "unmarshal")
	}
	r.Name = res.Name
	r.Type = res.Type
	r.Deps = res.Deps
	r.Sources = res.Sources

	if len(res.Input) > 0 {
		if r.Input.IsNull() {
			return fmt.Errorf("input type is null")
		}
		v, err := ctyjson.Unmarshal(res.Input, r.Input.Type())
		if err != nil {
			return errors.Wrap(err, "unmarshal input")
		}
		r.Input = v
	} else {
		r.Input = cty.NullVal(r.Input.Type())
	}

	if len(res.Output) > 0 {
		if r.Output.IsNull() {
			return fmt.Errorf("output type is null")
		}
		v, err := ctyjson.Unmarshal(res.Output, r.Output.Type())
		if err != nil {
			return errors.Wrap(err, "unmarshal output")
		}
		r.Output = v
	} else {
		r.Output = cty.NullVal(r.Output.Type())
	}

	return nil
}

// UnmarshalJSONType extracts the type name from a json encoded resource.
func UnmarshalJSONType(b []byte) (string, error) {
	var res jsonResource
	if err := json.Unmarshal(b, &res); err != nil {
		return "", errors.Wrap(err, "unmarshal")
	}
	return res.Type, nil
}

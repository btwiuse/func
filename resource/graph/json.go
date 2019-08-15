package graph

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/func/func/resource"
	"github.com/func/func/resource/schema"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

// A Registry provides types by name when decoding a graph from json.
type Registry interface {
	Type(name string) reflect.Type
}

// A JSONDecoder can decode a json encoded graph.
type JSONDecoder struct {
	Target   *Graph
	Registry Registry
}

// MarshalJSON marshals a graph to json.
func (g Graph) MarshalJSON() ([]byte, error) {
	out := jsonGraph{
		Resources:    make([]jsonResource, 0, len(g.Resources)),
		Dependencies: g.Dependencies,
	}
	// Ensure deterministic order
	names := make([]string, 0, len(g.Resources))
	for name := range g.Resources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		res := g.Resources[name]
		b, err := res.MarshalJSON()
		if err != nil {
			return nil, errors.Wrap(err, "marshal resource")
		}
		out.Resources = append(out.Resources, b)
	}
	return json.Marshal(out)
}

// UnmarshalJSON panics when called. A graph cannot be directly unmarshalled.
//
// The method is implemented for symmetry with MarshalJSON and to prevent
// accidental use of json.Unmarshal() directly on the graph.
//
// The reason the graph cannot be decoded is because the types are not known
// and thus inputs/outputs cannot be decoded.
//
// See JSONDecoder for decoding the graph instead.
func (g *Graph) UnmarshalJSON(b []byte) error {
	panic("Graph cannot be directly unmarshalled using UnmarshalJSON. Use graph.JSONDecoder to unmarshal a json graph")
}

// UnmarshalJSON unmarshals a JSON encoded graph into the target graph. Input
// and output types are retrieved from the registry.
func (dec JSONDecoder) UnmarshalJSON(b []byte) error {
	var in jsonGraph
	if err := json.Unmarshal(b, &in); err != nil {
		return errors.Wrap(err, "unmarshal input")
	}

	// Reset
	dec.Target.Resources = make(map[string]*resource.Resource, len(in.Resources))
	dec.Target.Dependencies = make(map[string][]Dependency, len(in.Dependencies))

	for _, resData := range in.Resources {
		typename, err := resource.UnmarshalJSONType(resData)
		if err != nil {
			return errors.Wrap(err, "get type name")
		}
		typ := dec.Registry.Type(typename)
		if typ == nil {
			return fmt.Errorf("type %q not registered", typename)
		}
		fields := schema.Fields(typ)
		res := &resource.Resource{
			Input:  cty.UnknownVal(fields.Inputs().CtyType()),
			Output: cty.UnknownVal(fields.Outputs().CtyType()),
		}
		if err := res.UnmarshalJSON(resData); err != nil {
			return errors.Wrap(err, "resource")
		}
		dec.Target.AddResource(res)
	}
	for name, deps := range in.Dependencies {
		for _, dep := range deps {
			dec.Target.AddDependency(name, dep)
		}
	}
	return nil
}

type jsonGraph struct {
	Resources    []jsonResource          `json:"res"`
	Dependencies map[string][]Dependency `json:"deps,omitempty"`
}

type jsonResource []byte

func (res jsonResource) MarshalJSON() ([]byte, error) {
	// Prevent encoding already json encoded resource data to base64
	if json.Valid(res) {
		// Already JSON encoded
		return res, nil
	}
	return json.Marshal([]byte(res))
}

func (res *jsonResource) UnmarshalJSON(b []byte) error {
	if json.Valid(b) {
		// Not base64 encoded []byte
		*res = b
		return nil
	}
	// Unmarshal base64 bytes
	var tmp []byte
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*res = tmp
	return nil
}

package graph

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/func/func/resource"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// The ResourceCodec encodes resources.
type ResourceCodec interface {
	MarshalResource(resource.Resource) ([]byte, error)
	UnmarshalResource(b []byte) (resource.Resource, error)
}

// JSONEncoder encodes and decodes graphs to/from json.
type JSONEncoder struct {
	Codec ResourceCodec
}

// Prevent accidentally using json.Marshal on graph.

// MarshalJSON panics.
// Instead, use JSONEncoder.Marshal() to marshal a graph.
func (g Graph) MarshalJSON() ([]byte, error) { panic("Use JSONEncoder.Marshal() to marshal graph") }

// UnmarshalJSON panics.
// Instead, use JSONEncoder.Unmarshal() to unmarshal a graph.
func (g *Graph) UnmarshalJSON([]byte) error { panic("Use JSONEncoder.Unmarshal() to unmarshal graph") }

// Marshal marshals the graph into a json encoded byte slice.
func (enc JSONEncoder) Marshal(g *Graph) ([]byte, error) {
	out := jsonGraph{
		Resources:    make([]jsonResource, 0, len(g.Resources)),
		Dependencies: make(map[string][]jsonDep, len(g.Dependencies)),
	}
	// Ensure deterministic order
	names := make([]string, 0, len(g.Resources))
	for name := range g.Resources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		res := g.Resources[name]
		b, err := enc.Codec.MarshalResource(*res)
		if err != nil {
			return nil, errors.Wrap(err, "marshal resource")
		}
		out.Resources = append(out.Resources, b)

		for _, d := range g.Dependencies[name] {
			out.Dependencies[name] = append(out.Dependencies[name], jsonDep{
				Field:      jsonPath{Path: d.Field},
				Expression: jsonExpression{Expression: d.Expression},
			})
		}
	}
	return json.Marshal(out)
}

// Unmarshal decodes a json encoded graph.
func (enc JSONEncoder) Unmarshal(b []byte, g *Graph) error {
	var in jsonGraph
	if err := json.Unmarshal(b, &in); err != nil {
		return errors.Wrap(err, "unmarshal input")
	}

	// Reset
	g.Resources = make(map[string]*resource.Resource, len(in.Resources))
	g.Dependencies = make(map[string][]Dependency, len(in.Dependencies))

	for _, resData := range in.Resources {
		res, err := enc.Codec.UnmarshalResource(resData)
		if err != nil {
			return err
		}
		g.AddResource(&res)
	}
	for name, deps := range in.Dependencies {
		for _, dep := range deps {
			g.AddDependency(name, Dependency{
				Field:      dep.Field.Path,
				Expression: dep.Expression.Expression,
			})
		}
	}
	return nil
}

type jsonGraph struct {
	Resources    []jsonResource       `json:"res"`
	Dependencies map[string][]jsonDep `json:"deps,omitempty"`
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

type jsonDep struct {
	Field      jsonPath       `json:"field"`
	Expression jsonExpression `json:"expr"`
}

type jsonPath struct{ cty.Path }

func (p jsonPath) MarshalJSON() ([]byte, error) {
	out := make([]cty.Value, len(p.Path))
	for i, p := range p.Path {
		switch v := p.(type) {
		case cty.GetAttrStep:
			out[i] = cty.StringVal(v.Name)
		case cty.IndexStep:
			out[i] = v.Key
		default:
			return nil, fmt.Errorf("unsupported step %T", v)
		}
	}
	list := cty.TupleVal(out)
	return ctyjson.Marshal(list, list.Type())
}

func (p *jsonPath) UnmarshalJSON(b []byte) error {
	var v ctyjson.SimpleJSONValue
	if err := v.UnmarshalJSON(b); err != nil {
		return err
	}
	p.Path = nil
	it := v.ElementIterator()
	for it.Next() {
		_, v := it.Element()
		if v.Type() == cty.String {
			// Attribute
			p.Path = append(p.Path, cty.GetAttrStep{Name: v.AsString()})
			continue
		}
		// Index
		p.Path = append(p.Path, cty.IndexStep{Key: v})
	}
	return nil
}

type jsonExpression struct{ Expression }

type jsonExprKey string

const (
	jsonExprLit jsonExprKey = "lit"
	jsonExprRef jsonExprKey = "ref"
)

type jsonExprPart map[jsonExprKey]json.RawMessage // Key is l (literal) or r (reference)

func (ex jsonExpression) MarshalJSON() ([]byte, error) {
	parts := make([]jsonExprPart, len(ex.Expression))
	for i, e := range ex.Expression {
		switch v := e.(type) {
		case ExprLiteral:
			b, err := ctyjson.Marshal(v.Value, v.Value.Type())
			if err != nil {
				return nil, errors.Wrap(err, "marshal literal")
			}
			parts[i] = jsonExprPart{jsonExprLit: b}
		case ExprReference:
			p := jsonPath{Path: v.Path}
			b, err := json.Marshal(p)
			if err != nil {
				return nil, errors.Wrap(err, "marshal reference path")
			}
			parts[i] = jsonExprPart{jsonExprRef: b}
		default:
			return nil, errors.Errorf("unsupported type %T at %d", v, i)
		}
	}
	return json.Marshal(parts)
}

func (ex *jsonExpression) UnmarshalJSON(b []byte) error {
	var parts []jsonExprPart
	if err := json.Unmarshal(b, &parts); err != nil {
		return errors.Wrap(err, "unmarshal expression parts")
	}
	ex.Expression = make(Expression, len(parts))
	for i, p := range parts {
		if lit, ok := p[jsonExprLit]; ok {
			var v ctyjson.SimpleJSONValue
			if err := v.UnmarshalJSON(lit); err != nil {
				return err
			}
			ex.Expression[i] = ExprLiteral{Value: v.Value}
			continue
		}
		if ref, ok := p[jsonExprRef]; ok {
			var p jsonPath
			if err := p.UnmarshalJSON(ref); err != nil {
				return err
			}
			ex.Expression[i] = ExprReference{Path: p.Path}
			continue
		}
		return errors.Errorf("unknown expression at %d: %v", i, p)
	}
	return nil
}

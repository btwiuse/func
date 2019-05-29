package graph

import (
	"encoding/json"
	"fmt"

	"github.com/func/func/resource"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// MarshalJSON marshals the graph into a json encoded byte slice.
//
// To save space, the dependencies of the graph are encoded inline with the
// resources.
func (g Graph) MarshalJSON() ([]byte, error) {
	r := jsonRoot{
		Resources: make(map[string]jsonResource, len(g.Resources)),
	}
	for name, res := range g.Resources {
		x := jsonResource{
			Type:    res.Type,
			Sources: res.Sources,
			Deps:    res.Deps,
		}
		if !res.Input.IsNull() {
			x.Input = &jsonData{Value: res.Input}
		}
		deps := g.Dependencies[name]
		x.Edges = make([]jsonDep, len(deps))
		for i, d := range deps {
			x.Edges[i] = jsonDep{
				Field:      jsonPath{Path: d.Field},
				Expression: jsonExpression{Expression: d.Expression},
			}
		}
		r.Resources[name] = x
	}
	return json.Marshal(r)
}

// UnmarshalJSON decodes a json encoded graph.
func (g *Graph) UnmarshalJSON(b []byte) error {
	var r jsonRoot
	if err := json.Unmarshal(b, &r); err != nil {
		return errors.WithStack(err)
	}
	g.Resources = make(map[string]*resource.Resource, len(r.Resources))
	edges := make(map[string][]Dependency)
	for name, res := range r.Resources {
		x := &resource.Resource{
			Name:    name,
			Type:    res.Type,
			Sources: res.Sources,
			Deps:    res.Deps,
		}
		if res.Input != nil {
			x.Input = res.Input.Value
		}
		for _, e := range res.Edges {
			edges[name] = append(edges[name], Dependency{
				Field:      e.Field.Path,
				Expression: e.Expression.Expression,
			})
		}
		g.AddResource(x)
	}
	if len(edges) > 0 {
		g.Dependencies = make(map[string][]Dependency)
		for name, dd := range edges {
			for _, d := range dd {
				g.AddDependency(name, d)
			}
		}
	}
	return nil
}

type jsonRoot struct {
	Resources map[string]jsonResource `json:"res"`
}

type jsonResource struct {
	Type    string    `json:"type"`
	Sources []string  `json:"srcs,omitempty"`
	Input   *jsonData `json:"input,omitempty"`
	Deps    []string  `json:"deps,omitempty"`  // Parent resource names.
	Edges   []jsonDep `json:"edges,omitempty"` // Dependency expressions.
}

type jsonData struct{ cty.Value }

func (d jsonData) MarshalJSON() ([]byte, error) {
	if d.IsNull() {
		return []byte("null"), nil
	}
	return ctyjson.Marshal(d.Value, d.Value.Type())
}

func (d *jsonData) UnmarshalJSON(b []byte) error {
	var v ctyjson.SimpleJSONValue
	if err := v.UnmarshalJSON(b); err != nil {
		return err
	}
	d.Value = v.Value
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

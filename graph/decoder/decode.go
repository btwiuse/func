package decoder

import (
	"fmt"
	"reflect"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

var rootSchema, _ = gohcl.ImpliedBodySchema(config.Root{})

// A ResourceRegistry is used for matching resource type names to resource
// implementations.
type ResourceRegistry interface {
	New(typename string) (resource.Resource, error)
	SuggestType(typename string) string
}

// DecodeContext is the context to use when decoding.
//
// For now, only the resource names can be provided.
type DecodeContext struct {
	Resources ResourceRegistry
}

// DecodeBody decodes a given raw configuration into the target graph.
//
func DecodeBody(body hcl.Body, ctx *DecodeContext, target *graph.Graph) hcl.Diagnostics {
	cont, diags := body.Content(rootSchema)
	if diags.HasErrors() {
		return diags
	}

	dec := &decode{
		graph:   target,
		outputs: makeOutputs(),
	}

	for _, b := range cont.Blocks {
		switch b.Type {
		case "project":
			diags = append(diags, dec.addProject(b)...)
		case "resource":
			diags = append(diags, dec.addResource(b, ctx)...)
		}
	}

	diags = append(diags, dec.connectRefs()...)

	return diags
}

// decode is a single decoding job
type decode struct {
	graph       *graph.Graph
	outputs     outputs
	pendingRefs []pendingRef
}

type pendingRef struct {
	resource resource.Resource
	ref      ref
}

func (p *pendingRef) fieldVal() interface{} {
	return p.ref.val.Field(p.ref.field.index).Addr().Interface()
}

func (d *decode) addProject(block *hcl.Block) hcl.Diagnostics {
	name := block.Labels[0]
	err := d.graph.SetProject(config.Project{
		Name: name,
	})
	if err != nil {
		return hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Cannot add project: %v", err),
			Subject:  block.DefRange.Ptr(),
		}}
	}
	return nil
}

type output struct {
	resource resource.Resource
	field    field
}

var outputType = cty.Capsule("output", reflect.TypeOf(output{}))

func (d *decode) addResource(block *hcl.Block, ctx *DecodeContext) hcl.Diagnostics {
	typename := block.Labels[0]
	resname := block.Labels[1]

	res, err := ctx.Resources.New(typename)
	if err != nil {
		diag := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Resource not supported",
			Subject:  block.LabelRanges[0].Ptr(),
		}
		type notsupported interface{ NotSupported() }
		if _, ok := err.(notsupported); ok {
			if s := ctx.Resources.SuggestType(typename); s != "" {
				diag.Detail = fmt.Sprintf("Did you mean %q?", s)
			}
		}
		return hcl.Diagnostics{diag}
	}

	var resBody config.Resource
	diags := gohcl.DecodeBody(block.Body, nil, &resBody)
	if diags.HasErrors() {
		return diags
	}

	v := reflect.Indirect(reflect.ValueOf(res))
	vals, refs, morediags := decodeInput(v, resBody.Config)
	diags = append(diags, morediags...)

	// create resource node
	d.graph.AddResource(res)

	// collect refs, we'll need to connect them later
	for _, ref := range refs {
		d.pendingRefs = append(d.pendingRefs, pendingRef{
			resource: res,
			ref:      ref,
		})
	}

	// add all outputs
	outputs := make(map[string]cty.Value)
	for name, val := range vals {
		outputs[name] = val
	}
	for _, field := range fieldsByTag(v.Type(), "output") {
		outputs[field.name] = cty.CapsuleVal(outputType, &output{
			resource: res,
			field:    field,
		})
	}
	d.outputs.add(typename, resname, outputs)

	if resBody.Source != nil {
		d.graph.AddSource(res, *resBody.Source)
	}

	return diags
}

func (d *decode) connectRefs() hcl.Diagnostics {
	ctx := &hcl.EvalContext{Variables: d.outputs.variables()}
	var diags hcl.Diagnostics
	for _, p := range d.pendingRefs {
		v, dd := p.ref.attr.Expr.Value(ctx)
		diags = append(diags, dd...)
		if dd.HasErrors() {
			continue
		}

		if !v.Type().IsCapsuleType() {
			val := p.fieldVal()
			err := gocty.FromCtyValue(v, val)
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Cannot set %s from %s, %v", p.ref.field.name, v.Type().FriendlyName(), err),
					Subject:  p.ref.attr.Range.Ptr(),
				})
			}
			continue
		}

		out, ok := v.EncapsulatedValue().(*output)
		if !ok {
			// If this panics, there is a bug in encapsulating the output
			// field.
			panic("Referenced output value not a capsule for *output")
		}

		d.graph.AddDependency(graph.Reference{
			Parent:      out.resource,
			ParentIndex: []int{out.field.index},
			Child:       p.resource,
			ChildIndex:  []int{p.ref.field.index},
		})
	}

	// Delete expressions, otherwise diagnostic cannot be unmarshalled from
	// json. There's probably a better way but this works.
	for _, d := range diags {
		d.Expression = nil
	}

	return diags
}

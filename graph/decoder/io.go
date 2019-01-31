package decoder

import (
	"reflect"

	"github.com/func/func/resource"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

// decodeInput decodes the user supplied configuration into a given target,
// based on input struct tags set in the target.
//
// val must be a struct
func decodeInput(val reflect.Value, config hcl.Body) (vals map[string]cty.Value, refs []ref, diags hcl.Diagnostics) {
	t := val.Type()

	inputs := resource.Fields(t, resource.Input)
	schema := inputSchema(inputs)

	cont, diags := config.Content(schema)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	vals = make(map[string]cty.Value)
	for _, input := range inputs {
		attr, ok := cont.Attributes[input.Name]
		if !ok {
			// Optional attribute was not set
			continue
		}

		if len(attr.Expr.Variables()) > 0 {
			// At this point, we only decode fixed values. Add the field as a
			// reference
			refs = append(refs, ref{
				attr:  attr,
				val:   val,
				field: input,
			})
			continue
		}

		fieldVal := val.Field(input.Index)
		ptr := fieldVal.Addr().Interface()

		diags = append(diags,
			gohcl.DecodeExpression(attr.Expr, nil, ptr)...,
		)

		val, morediags := attr.Expr.Value(nil)
		vals[input.Name] = val
		diags = append(diags, morediags...)
	}

	return vals, refs, diags
}

type ref struct {
	attr  *hcl.Attribute
	val   reflect.Value
	field resource.Field
}

func inputSchema(ff []resource.Field) *hcl.BodySchema {
	schema := &hcl.BodySchema{}
	for _, f := range ff {
		schema.Attributes = append(schema.Attributes, hcl.AttributeSchema{
			Name:     f.Name,
			Required: f.Type.Kind() != reflect.Ptr,
		})
	}
	return schema
}

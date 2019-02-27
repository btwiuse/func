package decoder

import (
	"fmt"
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

	// Attributes
	for _, input := range inputs {
		if isBlock(input) {
			continue
		}

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

	// Blocks
	blocksByType := cont.Blocks.ByType()
	for _, input := range inputs {
		if !isBlock(input) {
			continue
		}
		blocks := blocksByType[input.Name]

		if len(blocks) == 0 {
			if input.Type.Kind() == reflect.Ptr {
				// Missing optional block
				continue
			}
			// Missing required block
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Missing %s block", input.Name),
				Detail:   fmt.Sprintf("A %s block is required.", input.Name),
				Subject:  config.MissingItemRange().Ptr(),
			})
			continue
		}
		if len(blocks) > 1 {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Duplicate %s block", input.Name),
				Detail: fmt.Sprintf(
					"Only one %s block is allowed. Another was defined on line %d",
					input.Name,
					blocks[0].DefRange.Start.Line,
				),
				Subject: blocks[1].DefRange.Ptr(),
			})
			continue
		}

		if input.Type.Kind() == reflect.Ptr {
			// Initialize struct
			v := reflect.New(input.Type.Elem())
			val.Field(input.Index).Set(v)
		}

		field := reflect.Indirect(val.Field(input.Index))

		if !field.IsValid() {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("huh"),
				Subject:  blocks[0].DefRange.Ptr(),
			})
			continue
		}

		block := blocks[0]
		kv, r, d := decodeInput(field, block.Body)
		for k, v := range kv {
			vals[k] = v
		}
		refs = append(refs, r...)
		diags = append(diags, d...)
	}

	return vals, refs, diags
}

type ref struct {
	attr  *hcl.Attribute
	val   reflect.Value
	field resource.Field
}

func isBlock(f resource.Field) bool {
	if f.Type.Kind() == reflect.Struct {
		// Required block
		return true
	}
	if f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
		// Optional block
		return true
	}
	return f.Type.Kind() == reflect.Struct
}

func inputSchema(ff []resource.Field) *hcl.BodySchema {
	schema := &hcl.BodySchema{}
	for _, f := range ff {
		if isBlock(f) {
			schema.Blocks = append(schema.Blocks, hcl.BlockHeaderSchema{
				Type: f.Name,
			})
			continue
		}
		schema.Attributes = append(schema.Attributes, hcl.AttributeSchema{
			Name:     f.Name,
			Required: f.Type.Kind() != reflect.Ptr,
		})
	}
	return schema
}

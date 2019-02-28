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
		field := val.Type().Field(input.Index)

		ty := field.Type
		isPtr := false
		if ty.Kind() == reflect.Ptr {
			// Pointers are optional
			isPtr = true
			ty = ty.Elem()
		}

		isSlice := false
		if ty.Kind() == reflect.Slice {
			// Slices can capture multiple blocks
			isSlice = true
			ty = ty.Elem()
		}

		if len(blocks) > 1 && !isSlice {
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

		if len(blocks) == 0 {
			if isPtr || isSlice {
				// Missing optional block, or empty slice
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

		fieldVal := val.Field(input.Index)

		if isSlice {
			elemType := ty
			slice := reflect.MakeSlice(reflect.SliceOf(elemType), len(blocks), len(blocks))

			for i, block := range blocks {
				sliceIndex := slice.Index(i)
				if elemType.Kind() == reflect.Ptr {
					// Initialize struct pointer in slice
					v := reflect.New(elemType.Elem())
					slice.Index(i).Set(v)
					sliceIndex = v.Elem()
				}

				kv, r, d := decodeInput(sliceIndex, block.Body)
				for k, v := range kv {
					vals[k] = v
				}
				refs = append(refs, r...)
				diags = append(diags, d...)
			}

			if isPtr {
				// Set slice pointer
				slicePtr := reflect.New(slice.Type())
				slicePtr.Elem().Set(slice)
				fieldVal.Set(slicePtr)
				continue
			}

			// Set slice value
			fieldVal.Set(slice)
			continue
		}

		if isPtr {
			// Initialize struct pointer
			v := reflect.New(input.Type.Elem())
			fieldVal.Set(v)
		}

		block := blocks[0]
		kv, r, d := decodeInput(reflect.Indirect(fieldVal), block.Body)
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
	t := f.Type
	if t.Kind() == reflect.Ptr {
		// Optional
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		return true
	}
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Struct {
		return true
	}
	return false
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

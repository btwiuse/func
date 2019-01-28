package decoder

import (
	"fmt"
	"reflect"
	"strings"

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

	inputs := fieldsByTag(t, "input")
	schema := inputSchema(inputs)

	cont, diags := config.Content(schema)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	vals = make(map[string]cty.Value)
	for _, input := range inputs {
		attr, ok := cont.Attributes[input.name]
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

		fieldVal := val.Field(input.index)
		ptr := fieldVal.Addr().Interface()

		diags = append(diags,
			gohcl.DecodeExpression(attr.Expr, nil, ptr)...,
		)

		val, morediags := attr.Expr.Value(nil)
		vals[input.name] = val
		diags = append(diags, morediags...)
	}

	return vals, refs, diags
}

type ref struct {
	attr  *hcl.Attribute
	val   reflect.Value
	field field
}

type field struct {
	name  string
	attr  string // anything after a comma
	index int
	typ   reflect.Type
}

// inputFields returns all fields that have an input tag on them.
// Fields that have `,optional` are marked optional.
//
// Panics if an input tag is found on an unexported field, or if the string
// after the comma in the tag is not recognized.
func fieldsByTag(t reflect.Type, tag string) []field {
	var fields []field
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag, ok := f.Tag.Lookup(tag)
		if !ok {
			continue
		}
		if f.PkgPath != "" {
			panic(fmt.Sprintf("%s.%s: input set on unexported field", t, f.Name))
		}
		res := field{name: tag, index: i, typ: f.Type}
		if comma := strings.Index(tag, ","); comma >= 0 {
			res.name = tag[:comma]
			res.attr = tag[comma+1:]
		}
		fields = append(fields, res)
	}
	return fields
}

func inputSchema(ff []field) *hcl.BodySchema {
	schema := &hcl.BodySchema{}
	for _, f := range ff {
		schema.Attributes = append(schema.Attributes, hcl.AttributeSchema{
			Name:     f.name,
			Required: f.typ.Kind() != reflect.Ptr,
		})
	}
	return schema
}

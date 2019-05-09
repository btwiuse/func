package decoder

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/gocty"
)

// a decoder maintains the state of a single decode job.
type decoder struct {
	graph  *graph.Graph
	fields map[graph.Field]field
	names  map[string]*hcl.Range
}

func (d *decoder) decodeResource(block *hcl.Block, ctx *DecodeContext) hcl.Diagnostics {
	resname := block.Labels[0]

	if ex, ok := d.names[resname]; ok {
		return []*hcl.Diagnostic{{
			Severity: hcl.DiagError,
			Summary:  "Duplicate resource",
			Detail:   fmt.Sprintf("Another resource %q was defined on in %s on line %d", resname, ex.Filename, ex.Start.Line),
			Subject:  block.DefRange.Ptr(),
		}}
	}
	d.names[resname] = block.DefRange.Ptr()

	// Decode resource body. Will return errors for syntax errors.
	var spec config.Resource
	diags := gohcl.DecodeBody(block.Body, nil, &spec)
	if diags.HasErrors() {
		return diags
	}

	// Get resource definition based on resource type.
	def, err := ctx.Resources.New(spec.Type)
	if err != nil {
		diag := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Resource not supported",
			Subject:  block.DefRange.Ptr(), // TODO: set range on type attribute
		}
		type notsupported interface{ NotSupported() }
		if _, ok := err.(notsupported); ok {
			if s := ctx.Resources.SuggestType(spec.Type); s != "" {
				diag.Detail = fmt.Sprintf("Did you mean %q?", s)
			}
		}
		return hcl.Diagnostics{diag}
	}

	// Create resource node.
	// The resource definition is currently "empty"; the field values are not set.
	node := d.graph.AddResource(resource.Resource{Name: resname, Def: def})

	if spec.Source != "" {
		// Add source to resource.
		src, err := config.DecodeSourceString(spec.Source)
		if err != nil {
			return []*hcl.Diagnostic{{
				Severity: hcl.DiagError,
				Summary:  "Could not decode source information",
				Detail:   "Error: string must contain 3 parts separated by ':'. This is always a bug.",
				Subject:  block.DefRange.Ptr(),
			}}
		}
		d.graph.AddSource(node, src)
	}

	val := reflect.Indirect(reflect.ValueOf(def))
	fields, diags := d.decodeResBody(spec.Config, val)
	if diags.HasErrors() {
		// An error occurred in decoding attributes/blocks. Do not add the
		// resource to the graph.
		return diags
	}

	// Collect extracted fields with resource information so inputs/outputs can
	// be looked up later.
	for f, expr := range fields {
		var e *expression

		// Only inputs have expressions.
		if f.Dir == resource.Input {
			// Convert hclpack expression as the internals rely on known hcl
			// types for replacing values.
			if packexpr, ok := expr.(*hclpack.Expression); ok {
				// Parse into hcl.Expression
				ex, morediags := packexpr.Parse()
				if morediags.HasErrors() {
					diags = append(diags, morediags...)
					continue
				}
				expr = ex
			}

			e = &expression{Expression: expr}
			if morediags := e.validate(); morediags.HasErrors() {
				diags = append(diags, morediags...)
				continue
			}
		}

		target := graph.Field{Name: resname, Field: f.Name}
		d.fields[target] = field{def: def, info: f, expr: e}
	}

	return diags
}

// decodeResBody extracts top-level inputs and outputs from the body.
// Nested blocks are decoded directly into the target value.
func (d *decoder) decodeResBody(body hcl.Body, val reflect.Value) (map[resource.Field]hcl.Expression, hcl.Diagnostics) {
	// Get schema from target inputs.
	fields := resource.Fields(val.Type(), resource.Input)
	schema := inputSchema(fields)

	// Decode body with given schema.
	cont, diags := body.Content(schema)
	if diags.HasErrors() {
		return nil, diags
	}

	values := make(map[resource.Field]hcl.Expression)

	// Attributes
	for _, f := range fields {
		if isBlock(f) {
			continue
		}
		attr, ok := cont.Attributes[f.Name]
		if !ok {
			// Optional attribute was not set
			continue
		}
		values[f] = attr.Expr
	}

	// Blocks
	blocksByType := cont.Blocks.ByType()
	for _, f := range fields {
		if !isBlock(f) {
			continue
		}
		blocks := blocksByType[f.Name]
		val := val.Field(f.Index)
		diags = append(diags, decodeStaticBlocks(f.Name, body, blocks, val)...)
	}

	// Outputs
	for _, out := range resource.Fields(val.Type(), resource.Output) {
		values[out] = nil
	}

	return values, diags
}

// decodeStaticBlock decodes the static values from the body into the given
// struct. No dynamic expressions are allowed.
func decodeStaticBlocks(name string, parent hcl.Body, blocks hcl.Blocks, val reflect.Value) hcl.Diagnostics {
	typ := val.Type()

	isPtr := false
	if typ.Kind() == reflect.Ptr {
		// Pointers are optional
		isPtr = true
		typ = typ.Elem()
	}

	isSlice := false
	if typ.Kind() == reflect.Slice {
		// Slices can capture multiple blocks
		isSlice = true
		typ = typ.Elem()
	}

	var diags hcl.Diagnostics
	if isSlice && len(blocks) > 0 {
		elemType := typ
		// Create new slice to hold elements
		slice := reflect.MakeSlice(reflect.SliceOf(elemType), len(blocks), len(blocks))

		for i, b := range blocks {
			// Get value to block index.
			sliceIndex := slice.Index(i)

			if elemType.Kind() == reflect.Ptr {
				// Slice is a slice of pointers to blocks, initialize struct first.
				v := reflect.New(elemType.Elem())
				slice.Index(i).Set(v)
				sliceIndex = v.Elem()
			}

			// Decode slice block.
			diags = append(diags, decodeStaticBlocks(name, parent, hcl.Blocks{b}, sliceIndex)...)
		}

		if isPtr {
			// Set slice pointer
			slicePtr := reflect.New(slice.Type())
			slicePtr.Elem().Set(slice)
			val.Set(slicePtr)
			return diags
		}

		// Set field value with created slice.
		val.Set(slice)
		return diags
	}

	if len(blocks) > 1 {
		// Target field is not a slice of structs but multiple blocks were
		// defined.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Duplicate %s block", name),
			Detail: fmt.Sprintf(
				"Only one %s block is allowed. Another was defined on line %d",
				name,
				blocks[0].DefRange.Start.Line,
			),
			Subject: blocks[1].DefRange.Ptr(),
		})
		return diags
	}

	if len(blocks) == 0 {
		if isPtr {
			// Missing optional block, or zero blocks to insert into a
			// slice of structs.
			return diags
		}
		// Missing required block
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Missing %s block", name),
			Detail:   fmt.Sprintf("A %s block is required.", name),
			Subject:  parent.MissingItemRange().Ptr(),
		})
		return diags
	}

	// Decode block

	// We know there is exactly one block to decode.
	block := blocks[0]

	fields := resource.Fields(typ, resource.Input)
	schema := inputSchema(fields)
	cont, diags := block.Body.Content(schema)
	if diags.HasErrors() {
		return diags
	}

	if isPtr {
		// Initialize struct pointer
		v := reflect.New(typ)
		val.Set(v)
	}

	// Attributes
	for _, f := range fields {
		if isBlock(f) {
			continue
		}
		attr, ok := cont.Attributes[f.Name]
		if !ok {
			// Optional attribute was not set
			continue
		}

		fieldVal := val.Field(f.Index)
		ptr := fieldVal.Addr().Interface()

		// NOTE(akupila): Dynamic values are not supported in nested struct
		// blocks. This is a bit unfortunate and would be nice to fix.
		//
		//   1. Static values cannot be resolved from other inputs as they are
		//      now known; a resource that provides the input may be defined
		//      later.
		//   2. Dynamic inputs don't have a way of setting a deep value within
		//      a struct due to graph.Field only supporting field level access.
		//
		// This can probably be fixed by updating graph.Field to support nested
		// values, which would allow static values to be resolved as they
		// currently are for top-level inputs and dynamic values would be
		// possible to add too.
		//
		// At least if the user provides variables, they will get a nice error
		// message telling them they cannot do that.
		val, morediags := attr.Expr.Value(nil)
		diags = append(diags, morediags...)
		if morediags.HasErrors() {
			continue
		}

		convTy, err := gocty.ImpliedType(ptr)
		if err != nil {
			// NOTE: note sure what (if anything) can trigger this branch.
			panic(fmt.Sprintf("unsuitable target: %s", err))
		}

		value, err := convert.Convert(val, convTy)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   fmt.Sprintf("Unsuitable value: %s", err.Error()),
				Subject:  attr.Expr.StartRange().Ptr(),
				Context:  attr.Expr.Range().Ptr(),
			})
			continue
		}

		// Assign value to field.
		if err := gocty.FromCtyValue(value, ptr); err != nil {
			// This should not happen as the conversion to the target type was successful.
			panic(fmt.Sprintf("Assign value: %v", err))
		}
	}

	// Blocks
	blocksByType := cont.Blocks.ByType()
	for _, f := range fields {
		if !isBlock(f) {
			continue
		}
		blocks := blocksByType[f.Name]
		val := reflect.Indirect(val).Field(f.Index)
		diags = append(diags, decodeStaticBlocks(f.Name, block.Body, blocks, val)...)
	}

	return diags
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

func isBlock(f resource.Field) bool {
	t := f.Type
	if t.Kind() == reflect.Ptr {
		// Optional
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		return true
	}
	if t.Kind() == reflect.Slice {
		if t.Elem().Kind() == reflect.Struct {
			// Slice of structs
			return true
		}
		if t.Elem().Kind() == reflect.Ptr && t.Elem().Elem().Kind() == reflect.Struct {
			// Slice of struct pointers
			return true
		}
	}
	return false
}

// resolveValues resolves the static values and referenced values for all
// inputs.
func (d *decoder) resolveValues() hcl.Diagnostics {
	var diags hcl.Diagnostics
	for target, f := range d.fields {
		if f.output() {
			// Outputs don't get a static value assigned
			continue
		}

		// Resolve source value value
		static, dynamic, morediags := d.fieldValue(f)
		diags = append(diags, morediags...)
		if morediags.HasErrors() {
			continue
		}

		if dynamic != nil {
			err := d.graph.AddDependency(target, dynamic)
			if err != nil {
				// This will happen if an invalid value is passed into
				// AddDependency, which is always a bug in the decoder.
				panic(fmt.Sprintf("Cannot add dependency: %v", err))
			}
			continue
		}

		// Get destination field
		dstVal := f.value().Addr().Interface()
		convTy, err := gocty.ImpliedType(dstVal)
		if err != nil {
			// NOTE: note sure what (if anything) can trigger this branch.
			panic(fmt.Sprintf("unsuitable target: %s", err))
		}

		// Convert source value to value that can be assigned to field, for
		// example string(123) from int(123).
		value, err := convert.Convert(static, convTy)
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   fmt.Sprintf("Unsuitable value: %s", err.Error()),
				Subject:  f.expr.StartRange().Ptr(),
				Context:  f.expr.Range().Ptr(),
			})
			continue
		}

		// Assign value to field.
		err = gocty.FromCtyValue(value, dstVal)
		if err != nil {
			// This should not happen as the conversion to the target type was successful.
			panic(fmt.Sprintf("Assign value: %v", err))
		}

	}
	return diags
}

func (d *decoder) fieldValue(f field) (cty.Value, graph.Expression, hcl.Diagnostics) {
	if f.output() {
		// Outputs don't have a known value, return expression as is.
		return cty.NilVal, f.expr, nil
	}
	for _, r := range f.expr.Fields() {
		// TODO(akupila): prevent infinite recursion if field refers to self
		parent, ok := d.fields[r]
		if !ok {
			// Reference to a non-existing field.
			// Attempt to find object with matching name to find which field was not set.
			for par := range d.fields {
				if par.Name == r.Name {
					return cty.NilVal, nil, hcl.Diagnostics{
						&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Referenced value not found",
							Detail:   fmt.Sprintf("Object %s does not have a field %q", r.Name, r.Field),
							Subject:  f.expr.StartRange().Ptr(),
							Context:  f.expr.Range().Ptr(),
						},
					}
				}
			}
			// No object matched the name.
			return cty.NilVal, nil, hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Referenced value not found",
					Detail:   fmt.Sprintf("An object with name %q is not defined", r.Name),
					Subject:  f.expr.StartRange().Ptr(),
					Context:  f.expr.Range().Ptr(),
				},
			}
		}
		// Attempt to resolve parent field to static value by traversing tree
		// upwards if needed.
		parentVal, parentExpr, diags := d.fieldValue(parent)
		if diags.HasErrors() {
			// Change diagnostics context to point to expression, rather than referenced expression
			for _, d := range diags {
				d.Detail = strings.Replace(d.Detail, "An object", "A nested object", 1)
				d.Subject = f.expr.StartRange().Ptr()
				d.Context = f.expr.Range().Ptr()
			}
			return cty.NilVal, nil, diags
		}
		if parentExpr != nil {
			// Parent value could not be statically resolved, keep dynamic
			// field and attempt to resolve remaining fields.
			continue
		}
		if err := f.expr.setRef(r, parentVal); err != nil {
			panic(fmt.Sprintf("Update referenced value: %v", err))
		}
	}

	// Check if expression has dynamic fields left.
	// It it doesn't, the expression can be statically resolved.
	if len(f.expr.Fields()) == 0 {
		// Static input
		v, diags := f.expr.Value(nil)
		if diags.HasErrors() {
			// NOTE(akupila): unsure what could cause this to happen.
			return cty.NilVal, nil, diags
		}
		return v, nil, nil
	}

	// Expression was either fully or partially resolved. Regardless, it still
	// contains dynamic fields the need to be resolved at runtime.
	return cty.NilVal, f.expr, nil
}

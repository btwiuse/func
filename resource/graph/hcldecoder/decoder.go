package hcldecoder

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/func/func/config"
	"github.com/func/func/ctyext"
	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
	"github.com/func/func/resource/graph/hcldecoder/internal/expr"
	"github.com/func/func/resource/schema"
	"github.com/func/func/suggest"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// A ResourceRegistry is used for matching resource type names to resource
// implementations.
type ResourceRegistry interface {
	Type(typename string) reflect.Type
	Typenames() []string
}

// A Validator validates user input.
type Validator interface {
	// Validate is called for every input set by the user.
	Validate(input interface{}, rule string) error
}

// Decoder is the context to use when decoding.
//
// The same instance of Decoder must not be used more than once.
type Decoder struct {
	Resources ResourceRegistry
	Validator Validator

	resources map[string]*res
	sources   []*config.SourceInfo
}

// DecodeBody decodes a given raw configuration body into the target graph.
//
// Dependencies between resources are created as required and added to the
// graph. If expressions can be statically resolved, either directly or by
// following dependencies, they are not added as dependencies to the graph.
//
// References to fields with different but convertible type are allowed. For
// example, a string can receive its value from an int.
//
// A resource may declare an expression that is a combination of string
// literals and references as an input to a string field. This allows
// concatenating strings that will dynamically be resolved on runtime, based on
// outputs from parent resources.
//
// The returned Sources contains all source information that was decoded from
// the body. The resources added to the graph will only have the key attached
// to them.
func (d *Decoder) DecodeBody(body hcl.Body, target *graph.Graph) (*config.Project, []*config.SourceInfo, hcl.Diagnostics) { // nolint: lll
	var hclSchema, _ = gohcl.ImpliedBodySchema(config.Root{})

	if d.resources != nil {
		panic("DecodeBody must only be called once")
	}
	d.resources = make(map[string]*res)

	cont, diags := body.Content(hclSchema)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	var project *config.Project
	for _, b := range cont.Blocks {
		switch b.Type {
		case "project":
			if b.Labels[0] == "" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Project name not set",
					Subject:  b.LabelRanges[0].Ptr(),
					Context:  b.DefRange.Ptr(),
				})
			}
			project = &config.Project{}
			diags = append(diags, gohcl.DecodeBody(b.Body, nil, project)...)
			project.Name = b.Labels[0]
		case "resource":
			if b.Labels[0] == "" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Resource name not set",
					Subject:  b.LabelRanges[0].Ptr(),
					Context:  b.DefRange.Ptr(),
				})
			}
			diags = append(diags, d.decodeResource(b)...)
		}
	}

	diags = append(diags, d.resolveValues()...)

	if diags.HasErrors() {
		return project, d.sources, diags
	}

	if err := d.addResources(target); err != nil {
		// This only happens if there's a bug within the decoder, which
		// hopefully another test would catch.
		diags = append(diags, &hcl.Diagnostic{
			Summary: "Cannot add resource. This is always a bug.",
			Detail:  fmt.Sprintf("Error: %v", err),
		})
	}

	return project, d.sources, diags
}

func (d *Decoder) addResources(g *graph.Graph) error {
	// deps keep track of dependencies to add. The dependencies must be added
	// after all resources have been added.
	deps := make(map[string][]graph.Dependency)
	for name, res := range d.resources {
		r := &resource.Resource{
			Name:    name,
			Type:    res.Type,
			Sources: res.Sources,
			Deps:    res.Deps,
		}
		v, err := cty.Transform(res.Input, func(p cty.Path, v cty.Value) (cty.Value, error) {
			if !v.Type().IsCapsuleType() {
				return v, nil
			}

			// Reference
			expr := v.EncapsulatedValue().(*expression)
			deps[name] = append(deps[name], graph.Dependency{
				Field:      p,
				Expression: expr.Expression,
			})

			return cty.UnknownVal(expr.inputType), nil
		})
		if err != nil {
			// Should never happen as the transform does not return an error.
			return err
		}
		r.Input = v
		g.AddResource(r)
	}
	for name, dd := range deps {
		for _, dep := range dd {
			g.AddDependency(name, dep)
		}
	}
	return nil
}

// res contains temporary data for a decoded resource.
type res struct {
	Name     string
	DefRange *hcl.Range

	Type    string
	Sources []string
	Deps    []string

	// Inputs
	Input cty.Value

	// Outputs
	Outputs cty.Type
}

// expression wraps a graph expression with the source range.
type expression struct {
	field     schema.Field
	inputType cty.Type
	graph.Expression
	hcl.Range
}

// exprType is the type for an encapsulated expression when the expression is
// added as an input attribute to a resource. The capsule does not leave this
// package, the encapsulated value is extracted when values are resolved.
var exprType = cty.Capsule("expression", reflect.TypeOf(expression{}))

// decodeResource decodes a resource block and adds it to the decoder.
func (d *Decoder) decodeResource(block *hcl.Block) hcl.Diagnostics {
	res := &res{
		Name:     block.Labels[0],
		DefRange: block.DefRange.Ptr(),
	}

	// Check that another resource with the same name has not already been defined.
	if ex, ok := d.resources[res.Name]; ok {
		return []*hcl.Diagnostic{{
			Severity: hcl.DiagError,
			Summary:  "Duplicate resource",
			Detail: fmt.Sprintf(
				"Another resource %q was defined in %s on line %d.",
				res.Name, ex.DefRange.Filename, ex.DefRange.Start.Line,
			),
			Subject: res.DefRange.Ptr(),
		}}
	}

	// Decode resource body. The dynamic config will remain in resConfig.Config.
	// This will catch missing/incorrect high level attributes (type, source).
	var resConfig config.Resource
	diags := gohcl.DecodeBody(block.Body, nil, &resConfig)
	if diags.HasErrors() {
		// Only return first diagnostic. If an expression was set on the type
		// attribute, it would otherwise return two diagnostics: one for the
		// variable not being allowed and another for the variable not being
		// defined.
		return diags[:1]
	}
	res.Type = resConfig.Type

	// Add source to resource.
	if resConfig.Source != "" {
		src, err := config.DecodeSourceString(resConfig.Source)
		if err != nil {
			rng := hcldec.SourceRange(block.Body, &hcldec.AttrSpec{Name: "source", Type: cty.String})
			return []*hcl.Diagnostic{{
				Severity: hcl.DiagError,
				Summary:  "Could not decode source information",
				Detail:   fmt.Sprintf("Error: %v. This is always a bug.", err),
				Subject:  rng.Ptr(),
			}}
		}
		res.Sources = append(res.Sources, src.Key)
		d.sources = append(d.sources, &src)
	}

	// Get resource definition based on resource type.
	t := d.Resources.Type(resConfig.Type)
	if t == nil {
		rng := hcldec.SourceRange(block.Body, &hcldec.AttrSpec{Name: "type", Type: cty.String})
		diag := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Resource not supported",
			Subject:  rng.Ptr(),
		}
		availableTypes := d.Resources.Typenames()
		if s := suggest.String(resConfig.Type, availableTypes); s != "" {
			diag.Detail = fmt.Sprintf("Did you mean %q?", s)
		}
		return hcl.Diagnostics{diag}
	}

	fields := schema.Fields(t)

	// Decode inputs
	inputs, deps, morediags := d.decodeInputs(resConfig.Config, fields.Inputs())
	diags = append(diags, morediags...)
	res.Input = inputs
	res.Deps = uniqueStringSlice(deps)

	// Decode outputs
	res.Outputs = fields.Outputs().CtyType()

	// Add resource
	d.resources[res.Name] = res

	return diags
}

func uniqueStringSlice(ss []string) []string {
	got := make(map[string]struct{})
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := got[s]; ok {
			continue
		}
		out = append(out, s)
		got[s] = struct{}{}
	}
	return out
}

// deocdeInputs decodes inputs from the body using the given type as schema.
//
// The resolved values are converted to the target type if required, and
// validated if validation tags are returned from parsing the schema.
//
// The returned diagnostics may contain warnings, which should be displayed to
// the user but still result in valid inputs.
func (d *Decoder) decodeInputs(body hcl.Body, fields schema.FieldSet) (input cty.Value, deps []string, diags hcl.Diagnostics) { // nolint: lll
	schema := d.bodySchema(fields)

	cont, diags := body.Content(schema)

	// NOTE(akupila): We need to proceed even if diags contain errors.
	// - If cty.NilVal is returned and another object contains a reference to
	//   this object's input, it will panic.
	// - If cty.EmptyObjectVal is returned, inputs are not registered and
	//   references to the object will incorrectly say the object doesn't have
	//   an input.

	inputs := make(map[string]cty.Value)

	// Attributes
	deps, morediags := d.decodeAttributes(cont, fields, inputs)
	diags = append(diags, morediags...)

	// Blocks
	moredeps, morediags := d.decodeBlocks(cont, fields, inputs)
	diags = append(diags, morediags...)

	deps = append(deps, moredeps...)

	return cty.ObjectVal(inputs), deps, diags
}

func (d *Decoder) decodeAttributes(cont *hcl.BodyContent, ff schema.FieldSet, in map[string]cty.Value) ([]string, hcl.Diagnostics) { // nolint: lll
	var parents []string
	var diags hcl.Diagnostics
	for name, f := range ff {
		if d.isBlock(f.Type) {
			continue
		}

		typ := schema.ImpliedType(f.Type)

		attr, ok := cont.Attributes[name]
		if !ok {
			// Optional attribute was not set.
			in[name] = cty.NullVal(typ)
			continue
		}

		// Check if attribute contains dynamic references to other fields.
		if len(attr.Expr.Variables()) > 0 {
			for _, v := range attr.Expr.Variables() {
				parents = append(parents, v.RootName())
			}
			in[name] = cty.CapsuleVal(exprType, &expression{
				field:      f,
				inputType:  typ,
				Expression: expr.MustConvert(attr.Expr),
				Range:      attr.Range,
			})
			continue
		}

		// Get static value.
		v, morediags := attr.Expr.Value(nil)
		diags = append(diags, morediags...)
		if morediags.HasErrors() {
			continue
		}

		// If type does not match 1:1, check if it can be converted (int -> string etc).
		if !v.Type().Equals(typ) {
			converted, morediags := d.convertVal(v, typ, attr.Range.Ptr())
			diags = append(diags, morediags...)
			if morediags.HasErrors() {
				continue
			}
			v = converted
		}

		// Validate static input
		diags = append(diags, d.validate(v, f, attr.Expr.Range())...)

		in[name] = v
	}
	return parents, diags
}

func (d *Decoder) validate(val cty.Value, field schema.Field, exprRange hcl.Range) hcl.Diagnostics {
	rule := field.Tags["validate"]
	if rule == "" {
		// No validation rule
		return nil
	}
	var diags hcl.Diagnostics
	t := field.Type
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	goval := reflect.New(t)
	if err := ctyext.FromCtyValue(val, goval.Interface(), nil); err != nil {
		panic(err)
	}
	if err := d.Validator.Validate(goval.Elem().Interface(), rule); err != nil {
		detail := fmt.Sprintf("%+v", err)
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Validation error",
			Detail:   strings.ToUpper(detail[0:1]) + detail[1:],
			Subject:  exprRange.Ptr(),
		})
	}
	return diags
}

func (d *Decoder) decodeBlocks(cont *hcl.BodyContent, ff schema.FieldSet, in map[string]cty.Value) ([]string, hcl.Diagnostics) { // nolint: lll
	var deps []string // nolint: prealloc
	var diags hcl.Diagnostics

	blocksByType := cont.Blocks.ByType()
	for name, f := range ff {
		if !d.isBlock(f.Type) {
			continue
		}

		blocks := blocksByType[name]
		if f.Type.Kind() == reflect.Slice {
			// Multiple blocks
			list := make([]cty.Value, len(blocks))
			for i, b := range blocks {
				fields := schema.Fields(f.Type.Elem()) // Do not limit to inputs -- only top level input required
				v, moredeps, morediags := d.decodeInputs(b.Body, fields)
				deps = append(deps, moredeps...)
				diags = append(diags, morediags...)
				list[i] = v
			}
			in[name] = cty.ListVal(list)
			continue
		}

		if len(blocks) > 1 {
			// Target field is not a slice of structs but multiple blocks were
			// defined.
			prev, next := blocks[0], blocks[1]
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate block",
				Detail: fmt.Sprintf(
					"Only one %q block is allowed. Another was defined on line %d.",
					name, prev.DefRange.Start.Line,
				),
				Subject: next.DefRange.Ptr(),
				Context: hcl.RangeBetween(prev.DefRange, next.DefRange).Ptr(),
			})
			continue
		}

		if len(blocks) == 0 && d.isRequired(f.Type) {
			// Missing required block
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required block",
				Detail:   fmt.Sprintf("A %s block is required.", name),
				Subject:  cont.MissingItemRange.Ptr(),
			})
			continue
		}

		if len(blocks) == 0 {
			// Optional block was not set
			in[name] = cty.NullVal(schema.ImpliedType(f.Type))
			continue
		}

		// Single block
		b := blocks[0]
		fields := schema.Fields(f.Type) // Do not limit to inputs -- only top level input required
		v, moredeps, morediags := d.decodeInputs(b.Body, fields)
		deps = append(deps, moredeps...)
		diags = append(diags, morediags...)
		in[name] = v
	}

	return deps, diags
}

func (d *Decoder) resolveValues() hcl.Diagnostics {
	remainingRefs := 1 // ensure at least one cycle
	for remainingRefs > 0 {
		remainingRefs = 0
		for _, r := range d.resources {
			v, err := cty.Transform(r.Input, func(p cty.Path, v cty.Value) (cty.Value, error) {
				if !v.Type().IsCapsuleType() {
					// Not an expression
					return v, nil
				}

				expr := v.EncapsulatedValue().(*expression)
				exprRefs := 0
				for i, part := range expr.Expression {
					ref, ok := part.(graph.ExprReference)
					if !ok {
						continue
					}
					path := ref.Path
					exprRefs++

					// Get resource name
					root, ok := path[0].(cty.GetAttrStep)
					if !ok {
						diag := &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "First step must be an object name",
							Subject:  expr.Range.Ptr(),
						}
						return cty.NilVal, hcl.Diagnostics{diag}
					}

					// Find parent resource
					parent, ok := d.resources[root.Name]
					if !ok {
						diag := &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Referenced value not found",
							Detail:   fmt.Sprintf("An object named %q is not defined.", root.Name),
							Subject:  expr.Range.Ptr(),
						}
						names := make([]string, 0, len(d.resources))
						for k := range d.resources {
							names = append(names, k)
						}
						if s := suggest.String(root.Name, names); s != "" {
							diag.Detail += fmt.Sprintf(" Did you mean %q?", s)
						}
						return cty.NilVal, hcl.Diagnostics{diag}
					}

					// Get field name
					field, ok := path[1].(cty.GetAttrStep)
					if !ok {
						diag := &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Second step must be a field name",
							Subject:  expr.Range.Ptr(),
						}
						return cty.NilVal, hcl.Diagnostics{diag}
					}

					// Check output
					outputs := parent.Outputs.AttributeTypes()
					outputType, ok := outputs[field.Name]
					if ok {
						// Reference to output
						// Ensure the remaining path is valid, in case
						// reference is to a nested field in an output.
						_, err := ctyext.ApplyTypePath(outputType, path[2:])
						if err != nil {
							diag := &hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "Invalid reference",
								Detail:   fmt.Sprintf("Object %s (%s): %v.", parent.Name, parent.Type, err),
								Subject:  expr.Range.Ptr(),
							}
							return cty.NilVal, hcl.Diagnostics{diag}
						}
						// TODO: Do we need to check if types match? Maybe if expression has a length of 1?
						continue
					}

					// Check input
					inputs := parent.Input.AsValueMap()
					inputVal, ok := inputs[field.Name]
					if !ok {
						diag := &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "No such field",
							Detail: fmt.Sprintf(
								"Object %s (%s) does not have a field %q.",
								root.Name, parent.Type, field.Name,
							),
							Subject: expr.Range.Ptr(),
						}
						// Find suggestion
						var names []string
						for k := range inputs {
							names = append(names, k)
						}
						for k := range outputs {
							names = append(names, k)
						}
						if s := suggest.String(field.Name, names); s != "" {
							diag.Detail += fmt.Sprintf(" Did you mean %q?", s)
						}
						return cty.NilVal, hcl.Diagnostics{diag}
					}

					if inputVal.Type().IsCapsuleType() {
						// Reference to other reference that has not been resolved (yet).
						remainingRefs++
						continue
					}

					expr.Expression[i] = graph.ExprLiteral{Value: inputVal}
					exprRefs--
				}

				// References to other inputs enable a reference to be
				// statically resolved and replaced with the literal value.
				// Merge any consecutive literals into one.
				expr.Expression = expr.Expression.MergeLiterals()

				if exprRefs == 0 {
					// Expression can now be statically resolved.
					v, err := expr.Value(nil)
					if err != nil {
						return cty.NilVal, err
					}
					// Validate resolved value
					diags := d.validate(v, expr.field, expr.Range)
					if diags.HasErrors() {
						return cty.NilVal, diags
					}
					return v, nil
				}

				return v, nil
			})
			if err != nil {
				// If this panics, the Transform above did not return
				// diagnostics as its error.
				return err.(hcl.Diagnostics)
			}
			r.Input = v
		}
	}
	return nil
}

func (d *Decoder) convertVal(input cty.Value, want cty.Type, rng *hcl.Range) (cty.Value, hcl.Diagnostics) {
	got := input.Type()

	// Get conversion.
	conv := convert.GetConversion(got, want)
	if conv == nil {
		return cty.NilVal, []*hcl.Diagnostic{{
			Severity: hcl.DiagError,
			Summary:  "Unsuitable value type",
			Detail: fmt.Sprintf(
				"The value must be a %s, conversion from %s is not possible.",
				want.FriendlyName(),
				got.FriendlyNameForConstraint(),
			),
			Subject: rng,
		}}
	}

	// Convert.
	converted, err := conv(input)
	if err != nil {
		// This should not happen unless there's a bug in package convert.
		panic(fmt.Sprintf("Conversion failed: %v", err))
	}

	// Do not show warnings for safe conversions that the user cannot do anything about.
	if got.IsTupleType() && want.IsListType() {
		return converted, nil
	}
	if got.IsObjectType() && want.IsMapType() {
		return converted, nil
	}

	// Add warning that conversion was necessary.
	diags := []*hcl.Diagnostic{{
		Severity: hcl.DiagWarning,
		Summary: fmt.Sprintf(
			"Value is converted from %s to %s",
			got.FriendlyNameForConstraint(),
			want.FriendlyName(),
		),
		Subject: rng,
	}}

	return converted, diags
}

func (d *Decoder) bodySchema(fields schema.FieldSet) *hcl.BodySchema {
	s := &hcl.BodySchema{}
	for name, f := range fields {
		if d.isBlock(f.Type) {
			s.Blocks = append(s.Blocks, hcl.BlockHeaderSchema{
				Type: name,
			})
			continue
		}
		s.Attributes = append(s.Attributes, hcl.AttributeSchema{
			Name:     name,
			Required: d.isRequired(f.Type),
		})
	}
	return s
}

func (d *Decoder) isBlock(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
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

func (d *Decoder) isRequired(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		return false
	default:
		return true
	}
}

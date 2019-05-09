package decoder

import (
	"github.com/func/func/graph"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

type expression struct {
	hcl.Expression
}

func (e *expression) validate() hcl.Diagnostics {
	var diags hcl.Diagnostics
	for _, v := range e.Variables() {
		_, morediags := traversalField(v)
		diags = append(diags, morediags...)
	}
	return diags
}

// Fields returns all fields that are referenced by the field's expression.
func (e *expression) Fields() []graph.Field {
	vars := e.Expression.Variables()
	ff := make([]graph.Field, len(vars))
	for i, v := range vars {
		f, diags := traversalField(v)
		if diags.HasErrors() {
			// Assumes that validate() has been called first.
			panic(diags)
		}
		ff[i] = f
	}
	return ff
}

// Eval evaluates the expression with data into the target.
func (e *expression) Eval(data map[graph.Field]interface{}, target interface{}) error {
	m := make(map[string]map[string]cty.Value)
	for field, val := range data {
		if m[field.Name] == nil {
			m[field.Name] = make(map[string]cty.Value)
		}
		typ, err := gocty.ImpliedType(val)
		if err != nil {
			return errors.Wrap(err, "get implied type")
		}
		val, err := gocty.ToCtyValue(val, typ)
		if err != nil {
			return errors.Wrap(err, "get cty value")
		}
		m[field.Name][field.Field] = val
	}

	vars := make(map[string]cty.Value)
	for n, vv := range m {
		vars[n] = cty.MapVal(vv)
	}

	ctx := &hcl.EvalContext{
		Variables: vars,
	}

	result, diags := e.Expression.Value(ctx)
	if diags.HasErrors() {
		return diags
	}

	return gocty.FromCtyValue(result, target)
}

func traversalField(t hcl.Traversal) (graph.Field, hcl.Diagnostics) {
	if len(t) != 2 {
		return graph.Field{}, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   "A reference must have 2 fields: {name}.{field}.",
				Subject:  t.SourceRange().Ptr(),
			},
		}
	}

	// Empty name or field will not get this far as decoding the expression will fail.

	return graph.Field{
		Name:  t.RootName(),
		Field: t[1].(hcl.TraverseAttr).Name,
	}, nil
}

// matchesField returns true if the given expression is equal to the field.
func matchesField(expr hcl.Expression, field graph.Field) bool {
	ste, ok := expr.(*hclsyntax.ScopeTraversalExpr)
	if !ok {
		return false
	}
	target, diags := traversalField(ste.Traversal)
	if diags.HasErrors() {
		// Assumes validate() was called first.
		panic(diags)
	}
	return target == field
}

// setRef replaces a reference with a resolved value.
func (e *expression) setRef(field graph.Field, value cty.Value) error {
	if matchesField(e.Expression, field) {
		// Root matches exactly
		e.Expression = &hclsyntax.LiteralValueExpr{
			Val:      value,
			SrcRange: e.Expression.Range(),
		}
		return nil
	}

	tmpl, ok := e.Expression.(*hclsyntax.TemplateExpr)
	if !ok {
		return errors.Errorf("Unsupported expression type %T", e.Expression)
	}
	match := false
	for i, p := range tmpl.Parts {
		if !matchesField(p, field) {
			continue
		}
		tmpl.Parts[i] = &hclsyntax.LiteralValueExpr{
			Val:      value,
			SrcRange: p.StartRange(),
		}
		match = true
	}
	if !match {
		return errors.New("no such field")
	}
	return nil
}

package expr

import (
	"fmt"

	"github.com/func/func/resource"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/zclconf/go-cty/cty"
)

// MustConvert converts a HCL expression into a graph expression.
//
// Only simple expression containing template literals or traversals are
// supported.
//
// Panics if conversion is not possible. This indicates that an expression is
// not supported.
func MustConvert(input hcl.Expression) resource.Expression {
	if len(input.Variables()) == 0 {
		val, diags := input.Value(nil)
		if diags.HasErrors() {
			// Should not happen; the expression has no variables so it should
			// be resolvable with a nil eval context.
			panic(fmt.Sprintf("Get static value for expression conversion: %v", diags))
		}
		return resource.Expression{resource.ExprLiteral{Value: val}}
	}

	// Special case for hclpack.Expression: convert to hclsyntax.Expression.
	if packexpr, ok := input.(*hclpack.Expression); ok {
		// Parse into hcl.Expression
		ex, diags := packexpr.Parse()
		if diags.HasErrors() {
			// This will only happen if the packed expression is not valid, indicating a bug in hclpack.
			panic(fmt.Sprintf("Parse hclpack expression: %v", diags))
		}
		input = ex
	}

	if expr, ok := input.(*hclsyntax.RelativeTraversalExpr); ok {
		src := MustConvert(expr.Source)

		// The collection will always resolve to a reference value, use the
		// path from it as a starting point.
		path := src[0].(resource.ExprReference).Path
		path = append(path, traversalAsPath(expr.Traversal)...)

		return resource.Expression{resource.ExprReference{Path: path}}
	}

	if expr, ok := input.(*hclsyntax.ScopeTraversalExpr); ok {
		path := traversalAsPath(expr.Traversal)
		return resource.Expression{resource.ExprReference{Path: path}}
	}

	if expr, ok := input.(*hclsyntax.IndexExpr); ok {
		col := MustConvert(expr.Collection)
		key := MustConvert(expr.Key)

		// The collection will always resolve to a reference value, use the
		// path from it as a starting point.
		path := col[0].(resource.ExprReference).Path

		// Append key(s) as indices
		for _, k := range key {
			lit := k.(resource.ExprLiteral)
			path = path.Index(lit.Value)
		}

		return resource.Expression{resource.ExprReference{Path: path}}
	}

	if expr, ok := input.(*hclsyntax.TemplateWrapExpr); ok {
		return MustConvert(expr.Wrapped)
	}

	if expr, ok := input.(*hclsyntax.TemplateExpr); ok {
		var out resource.Expression
		for _, p := range expr.Parts {
			out = append(out, MustConvert(p)...)
		}
		return out
	}

	panic(fmt.Sprintf("Unsupported: %T", input))
}

func traversalAsPath(traversal hcl.Traversal) cty.Path {
	var path cty.Path
	for _, part := range traversal {
		switch pt := part.(type) {
		case hcl.TraverseRoot:
			path = append(path, cty.GetAttrStep{Name: pt.Name})
		case hcl.TraverseAttr:
			path = append(path, cty.GetAttrStep{Name: pt.Name})
		case hcl.TraverseIndex:
			path = append(path, cty.IndexStep{Key: pt.Key})
		default:
			panic(fmt.Sprintf("not supported: %T", part))
		}
	}
	return path
}

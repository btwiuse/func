package expr_test

import (
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/resource/hcldecoder/internal/expr"
	"github.com/go-stack/stack"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/zclconf/go-cty/cty"
)

func TestMustConvert(t *testing.T) {
	tests := []struct {
		name string
		expr func(t *testing.T) hcl.Expression
		want resource.Expression
	}{
		{
			"StaticExpr",
			func(t *testing.T) hcl.Expression {
				return hcl.StaticExpr(cty.StringVal("foo"), hcl.Range{})
			},
			resource.Expression{
				resource.ExprLiteral{Value: cty.StringVal("foo")},
			},
		},
		{
			"HCLSyntax_static",
			func(t *testing.T) hcl.Expression {
				ex, diags := hclsyntax.ParseExpression([]byte(`"foo"`), "", hcl.InitialPos)
				if diags.HasErrors() {
					t.Fatal(diags)
				}
				return ex
			},
			resource.Expression{
				resource.ExprLiteral{Value: cty.StringVal("foo")},
			},
		},
		{
			"HCLSyntax_ref",
			func(t *testing.T) hcl.Expression {
				ex, diags := hclsyntax.ParseExpression([]byte(`foo.bar[2]`), "", hcl.InitialPos)
				if diags.HasErrors() {
					t.Fatal(diags)
				}
				return ex
			},
			resource.Expression{
				resource.ExprReference{Path: cty.GetAttrPath("foo").GetAttr("bar").Index(cty.NumberIntVal(2))},
			},
		},
		{
			"HCLSyntax_wrapped",
			func(t *testing.T) hcl.Expression {
				ex, diags := hclsyntax.ParseExpression([]byte(`"${foo.bar}"`), "", hcl.InitialPos)
				if diags.HasErrors() {
					t.Fatal(diags)
				}
				return ex
			},
			resource.Expression{
				resource.ExprReference{Path: cty.GetAttrPath("foo").GetAttr("bar")},
			},
		},
		{
			"HCLSyntax_mapAccess",
			func(t *testing.T) hcl.Expression {
				ex, diags := hclsyntax.ParseExpression([]byte(`foo["baz"]`), "", hcl.InitialPos)
				if diags.HasErrors() {
					t.Fatal(diags)
				}
				return ex
			},
			resource.Expression{
				resource.ExprReference{Path: cty.GetAttrPath("foo").Index(cty.StringVal("baz"))},
			},
		},
		{
			"HCLPack_simple",
			func(t *testing.T) hcl.Expression {
				return &hclpack.Expression{
					Source:     []byte(`"foo"`),
					SourceType: hclpack.ExprNative,
				}
			},
			resource.Expression{
				resource.ExprLiteral{Value: cty.StringVal("foo")},
			},
		},
		{
			"HCLPack_realistic",
			func(t *testing.T) hcl.Expression {
				src := `"arn:aws:execute-api:${api.region}:${me.account}:${api.id}/*/${get_world.http_method}${world.path}"`
				return &hclpack.Expression{
					Source:     []byte(src),
					SourceType: hclpack.ExprNative,
				}
			},
			resource.Expression{
				resource.ExprLiteral{Value: cty.StringVal("arn:aws:execute-api:")},
				resource.ExprReference{Path: cty.GetAttrPath("api").GetAttr("region")},
				resource.ExprLiteral{Value: cty.StringVal(":")},
				resource.ExprReference{Path: cty.GetAttrPath("me").GetAttr("account")},
				resource.ExprLiteral{Value: cty.StringVal(":")},
				resource.ExprReference{Path: cty.GetAttrPath("api").GetAttr("id")},
				resource.ExprLiteral{Value: cty.StringVal("/*/")},
				resource.ExprReference{Path: cty.GetAttrPath("get_world").GetAttr("http_method")},
				resource.ExprReference{Path: cty.GetAttrPath("world").GetAttr("path")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer checkPanic(t)

			got := expr.MustConvert(tt.expr(t))

			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
				cmp.Transformer("Name", func(v cty.GetAttrStep) string { return v.Name }),
				cmp.Transformer("GoString", func(v cty.IndexStep) string { return v.GoString() }),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("MustConvert() (-got +want) %s", diff)
			}
		})
	}
}

func checkPanic(t *testing.T) {
	t.Helper()
	if err := recover(); err != nil {
		c := stack.Caller(3)
		t.Fatalf("Panic: %v: %v", c, err)
	}
}

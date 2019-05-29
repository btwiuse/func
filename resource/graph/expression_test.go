package graph_test

import (
	"fmt"
	"testing"

	"github.com/func/func/resource/graph"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"
)

// Dummy variable to hold the value from Example
var expr graph.Expression

func ExampleExpression_forTest() {
	// Construct a new expression for test
	expr = graph.Expression{
		graph.ExprLiteral{Value: cty.StringVal("hello")},
		graph.ExprReference{Path: cty.GetAttrPath("other").GetAttr("output")},
		graph.ExprLiteral{Value: cty.NumberIntVal(123)},
	}

	// expr is equivalent to "hello${other.output}123"
}

func ExampleExpression_MergeLiterals() {
	input := graph.Expression{
		graph.ExprLiteral{Value: cty.StringVal("foo")},
		graph.ExprLiteral{Value: cty.StringVal("bar")},
		graph.ExprReference{Path: cty.GetAttrPath("abc")},
		graph.ExprLiteral{Value: cty.StringVal("baz")},
		graph.ExprLiteral{Value: cty.StringVal("qux")},
	}

	merged := input.MergeLiterals()

	output := graph.Expression{
		graph.ExprLiteral{Value: cty.StringVal("foobar")},
		graph.ExprReference{Path: cty.GetAttrPath("abc")},
		graph.ExprLiteral{Value: cty.StringVal("bazqux")},
	}

	fmt.Println(output.Equals(merged))
	// output: true
}

func TestExpression_Value(t *testing.T) {
	tests := []struct {
		name    string
		expr    graph.Expression
		ctx     *graph.EvalContext
		want    cty.Value
		wantErr bool
	}{
		{
			name: "Empty",
			expr: graph.Expression{},
			want: cty.NilVal,
		},
		{
			name: "Literal",
			expr: graph.Expression{
				graph.ExprLiteral{cty.StringVal("hello")},
			},
			ctx:  nil,
			want: cty.StringVal("hello"),
		},
		{
			name: "LiteralNum",
			expr: graph.Expression{
				graph.ExprLiteral{cty.NumberUIntVal(123)},
			},
			ctx:  nil,
			want: cty.NumberUIntVal(123),
		},
		{
			name: "Reference",
			expr: graph.Expression{
				graph.ExprReference{cty.GetAttrPath("foo").GetAttr("bar")},
			},
			ctx: &graph.EvalContext{
				Variables: map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.NumberIntVal(123),
					}),
				},
			},
			want: cty.NumberIntVal(123),
		},
		{
			name: "Mixed",
			expr: graph.Expression{
				graph.ExprReference{cty.GetAttrPath("foo").GetAttr("bar")},
				graph.ExprLiteral{cty.NumberIntVal(456)},
				graph.ExprReference{cty.GetAttrPath("bar").GetAttr("baz")},
			},
			ctx: &graph.EvalContext{
				Variables: map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{"bar": cty.NumberIntVal(123)}),
					"bar": cty.ObjectVal(map[string]cty.Value{"baz": cty.NumberFloatVal(789.0)}),
				},
			},
			want: cty.StringVal("123456789"), // Always a string
		},
		{
			name: "Unknown",
			expr: graph.Expression{
				graph.ExprLiteral{cty.StringVal("known")},
				graph.ExprReference{cty.GetAttrPath("foo").GetAttr("output")},
			},
			ctx: &graph.EvalContext{
				Variables: map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{"output": cty.UnknownVal(cty.String)}),
				},
			},
			want: cty.UnknownVal(cty.String),
		},
		{
			name: "NotFoundRef",
			expr: graph.Expression{
				graph.ExprReference{cty.GetAttrPath("foo")},
			},
			ctx: &graph.EvalContext{
				Variables: map[string]cty.Value{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.expr.Value(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Value() err = %v, wantErr = %t", err, tt.wantErr)
			}
			if tt.wantErr {
				t.Logf("Got expected error: %v", err)
				return
			}
			if tt.want.IsWhollyKnown() {
				if !got.Equals(tt.want).True() {
					t.Errorf("Value does not match\nGot  %s\nWant %s", got.GoString(), tt.want.GoString())
				}
				return
			}
			if got.IsWhollyKnown() {
				t.Errorf("Got known value, want unknown value")
			}
		})
	}
}

func TestExpression_MergeLiterals(t *testing.T) {
	tests := []struct {
		name string
		expr graph.Expression
		want graph.Expression
	}{
		{"Empty", nil, nil},
		{
			"SingleLiteral",
			graph.Expression{graph.ExprLiteral{Value: cty.StringVal("foo")}},
			graph.Expression{graph.ExprLiteral{Value: cty.StringVal("foo")}},
		},
		{
			"Consecutive",
			graph.Expression{
				graph.ExprLiteral{Value: cty.StringVal("foo")},
				graph.ExprLiteral{Value: cty.StringVal("bar")},
			},
			graph.Expression{
				graph.ExprLiteral{Value: cty.StringVal("foobar")},
			},
		},
		{
			"ConsecutiveStringNumber",
			graph.Expression{
				graph.ExprLiteral{Value: cty.StringVal("foo")},
				graph.ExprLiteral{Value: cty.NumberIntVal(123)},
			},
			graph.Expression{
				graph.ExprLiteral{Value: cty.StringVal("foo123")},
			},
		},
		{
			"Mixed",
			graph.Expression{
				graph.ExprLiteral{Value: cty.StringVal("foo")},
				graph.ExprLiteral{Value: cty.StringVal("bar")},
				graph.ExprReference{Path: cty.GetAttrPath("abc")},
				graph.ExprLiteral{Value: cty.StringVal("baz")},
				graph.ExprLiteral{Value: cty.StringVal("qux")},
			},
			graph.Expression{
				graph.ExprLiteral{Value: cty.StringVal("foobar")},
				graph.ExprReference{Path: cty.GetAttrPath("abc")},
				graph.ExprLiteral{Value: cty.StringVal("bazqux")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.expr.MergeLiterals()
			opts := []cmp.Option{
				cmpopts.EquateEmpty(),
				cmp.Transformer("GoString", func(v cty.Value) string { return v.GoString() }),
				cmp.Transformer("Name", func(v cty.GetAttrStep) string { return v.Name }),
				cmp.Transformer("GoString", func(v cty.IndexStep) string { return v.GoString() }),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("MergeLiterals() (-got +want)\n%s", diff)
			}
		})
	}
}

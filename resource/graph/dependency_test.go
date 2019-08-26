package graph_test

import (
	"testing"

	"github.com/func/func/resource/graph"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestDependency_Parents(t *testing.T) {
	dep := graph.Dependency{
		Field: cty.GetAttrPath("input"),
		Expression: graph.Expression{
			graph.ExprLiteral{Value: cty.StringVal("foo")},
			graph.ExprReference{Path: cty.GetAttrPath("parent1").GetAttr("output")},
			graph.ExprLiteral{Value: cty.StringVal("bar")},
			graph.ExprReference{Path: cty.GetAttrPath("parent2").GetAttr("output")},
		},
	}
	want := []string{"parent1", "parent2"}

	got := dep.Parents()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Parents() (-got +want)\n%s", diff)
	}
}

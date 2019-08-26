package resource_test

import (
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestDependency_Parents(t *testing.T) {
	dep := resource.Dependency{
		Field: cty.GetAttrPath("input"),
		Expression: resource.Expression{
			resource.ExprLiteral{Value: cty.StringVal("foo")},
			resource.ExprReference{Path: cty.GetAttrPath("parent1").GetAttr("output")},
			resource.ExprLiteral{Value: cty.StringVal("bar")},
			resource.ExprReference{Path: cty.GetAttrPath("parent2").GetAttr("output")},
		},
	}
	want := []string{"parent1", "parent2"}

	got := dep.Parents()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Parents() (-got +want)\n%s", diff)
	}
}

package graph_test

import (
	"testing"

	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestGraph_AddResource(t *testing.T) {
	g := graph.New()
	res := g.AddResource(resource.Resource{Name: "foo", Def: &mockDef{Input: "foo"}})

	got := g.Resources()
	want := []*graph.Resource{res}
	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(graph.Resource{}),
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Diff() (-got, +want)\n%s", diff)
	}
}

func TestGraph_AddDependency(t *testing.T) {
	g := graph.New()
	res1 := g.AddResource(resource.Resource{Name: "foo", Def: &mockDef{Input: "foo"}})
	res2 := g.AddResource(resource.Resource{Name: "bar", Def: &mockDef{}})
	expr := noopExpr{Parents: []graph.Field{{Name: "foo", Field: "output"}}}

	target := graph.Field{Name: "bar", Field: "input"}
	if err := g.AddDependency(target, expr); err != nil {
		t.Fatalf("Add dependency: %v", err)
	}

	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(graph.Resource{}),
		cmpopts.IgnoreUnexported(graph.Dependency{}),
		cmpopts.EquateEmpty(),
	}

	got := res1.Dependencies()
	want := []*graph.Dependency{}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("res1.Dependencies() (-got, +want)\n%s", diff)
	}

	got = res1.Dependents()
	want = []*graph.Dependency{
		{Target: target, Expr: expr},
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("res1.Dependents() (-got, +want)\n%s", diff)
	}

	got = res2.Dependencies()
	want = []*graph.Dependency{
		{Target: target, Expr: expr},
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("res2.Dependencies() (-got, +want)\n%s", diff)
	}

	got = res2.Dependents()
	want = []*graph.Dependency{}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("res2.Dependents() (-got, +want)\n%s", diff)
	}
}

type mockDef struct {
	resource.Definition
	Input  string
	Output string
}

func (m mockDef) Type() string { return "mock" }

type noopExpr struct {
	Parents []graph.Field
}

func (x noopExpr) Fields() []graph.Field                               { return x.Parents }
func (x noopExpr) Eval(map[graph.Field]interface{}, interface{}) error { return nil }

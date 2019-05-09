package graph_test

import (
	"testing"

	"github.com/func/func/config"
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

func TestGraph_AddSource(t *testing.T) {
	g := graph.New()
	res := g.AddResource(resource.Resource{Name: "foo", Def: &mockDef{Input: "foo"}})
	src := g.AddSource(res, config.SourceInfo{Key: "123"})

	got := g.Sources()
	want := []*graph.Source{src}
	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(graph.Source{}),
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Diff() (-got, +want)\n%s", diff)
	}
}

func TestGraph_AddDependency(t *testing.T) {
	g := graph.New()
	res1 := g.AddResource(resource.Resource{Name: "foo", Def: &mockDef{Input: "foo"}})
	res2 := g.AddResource(resource.Resource{Name: "bar", Def: &mockDef{}})
	expr := noopExpr{Parents: []graph.Field{{Type: "mock", Name: "foo", Field: "output"}}}

	target := graph.Field{Type: "mock", Name: "bar", Field: "input"}
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

func TestGraph_reverse(t *testing.T) {
	g := graph.New()
	res := g.AddResource(resource.Resource{Name: "foo", Def: &mockDef{Input: "foo"}})
	g.AddSource(res, config.SourceInfo{Key: "abc"})

	// Traverse to source, then back to resource
	resSources := res.Sources()
	if len(resSources) != 1 {
		t.Fatalf("resource sources = %v, want = %v", len(resSources), 1)
	}
	for _, s := range res.Sources() {
		got := s.Resource()
		if got != res {
			t.Errorf("source.Resource() = %v, want = %v", got, res)
		}
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

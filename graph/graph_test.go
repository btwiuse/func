package graph_test

import (
	"testing"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestGraph_AddResource(t *testing.T) {
	g := graph.New()
	res := g.AddResource(&mockRes{Value: "foo"})

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
	res := g.AddResource(&mockRes{Value: "foo"})
	src := g.AddSource(res, config.SourceInfo{SHA: "123"})

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
	res1 := g.AddResource(&mockRes{Value: "foo"})
	res2 := g.AddResource(&mockRes{Value: "bar"})
	ref := graph.Reference{
		Source: graph.Field{Resource: res1, Index: []int{0}},
		Target: graph.Field{Resource: res2, Index: []int{0}},
	}
	g.AddDependency(ref)

	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(graph.Resource{}),
		cmpopts.EquateEmpty(),
	}

	got := res1.Dependencies()
	want := []graph.Reference{}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("res1.Dependencies() (-got, +want)\n%s", diff)
	}

	got = res1.Dependents()
	want = []graph.Reference{ref}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("res1.Dependents() (-got, +want)\n%s", diff)
	}

	got = res2.Dependencies()
	want = []graph.Reference{ref}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("res2.Dependencies() (-got, +want)\n%s", diff)
	}

	got = res2.Dependents()
	want = []graph.Reference{}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("res2.Dependents() (-got, +want)\n%s", diff)
	}
}

func TestGraph_reverse(t *testing.T) {
	g := graph.New()
	res := g.AddResource(&mockRes{Value: "foo"})
	g.AddSource(res, config.SourceInfo{SHA: "abc"})

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

type mockRes struct {
	Value string
}

func (mockRes) Type() string { return "mock" }

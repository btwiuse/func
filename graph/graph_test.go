package graph_test

import (
	"fmt"
	"testing"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestGraph(t *testing.T) {
	g := graph.New()

	proj := config.Project{Name: "test"}
	def1 := &mockResource{Str: "a"}
	def2 := &mockResource{Str: "b"}
	def3 := &mockResource{Str: "c"}
	srcInfo := config.SourceInfo{SHA: "abc"}

	err := g.SetProject(proj)
	if err != nil {
		t.Fatalf("SetProject() error = %v", err)
	}

	res1 := g.AddResource(def1)
	res2 := g.AddResource(def2)
	res3 := g.AddResource(def3)
	ref12 := graph.Reference{
		Source: graph.Field{Resource: res1, Index: []int{1}},
		Target: graph.Field{Resource: res2, Index: []int{2}},
	}
	ref13 := graph.Reference{
		Source: graph.Field{Resource: res1, Index: []int{2}},
		Target: graph.Field{Resource: res3, Index: []int{1}},
	}
	ref23 := graph.Reference{
		Source: graph.Field{Resource: res2, Index: []int{3}},
		Target: graph.Field{Resource: res3, Index: []int{4}},
	}
	g.AddDependency(ref12)
	g.AddDependency(ref13)
	g.AddDependency(ref23)
	src := g.AddSource(res1, srcInfo)

	// digraph G {
	//     project -> {res1, res2, res3};
	//     res1 -> {res2, res3};
	//     res2 -> res3;
	//     source -> res1;
	// }

	tests := []struct {
		name   string
		assert func(t *testing.T, g *graph.Graph)
	}{
		{"Project", func(t *testing.T, g *graph.Graph) {
			got := g.Project()
			want := proj
			if got == nil {
				t.Fatal("Project() got = nil")
			}
			if diff := cmp.Diff(got.Project, want); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		}},
		{"Resources", func(t *testing.T, g *graph.Graph) {
			got := g.Resources()
			want := []*graph.Resource{res1, res2, res3}
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		}},
		{"References", func(t *testing.T, g *graph.Graph) { // Parents
			refTests := []struct {
				name       string
				res        *graph.Resource
				upstream   []graph.Reference
				downstream []graph.Reference
			}{
				{
					"res1", res1,
					[]graph.Reference{},
					[]graph.Reference{ref12, ref13},
				},
				{
					"res2", res2,
					[]graph.Reference{ref12},
					[]graph.Reference{ref23},
				},
				{
					"res3", res3,
					[]graph.Reference{ref13, ref23},
					[]graph.Reference{},
				},
			}
			for _, tt := range refTests {
				t.Run(fmt.Sprintf("Dependents(%s)", tt.name), func(t *testing.T) {
					parents := g.Dependencies(tt.res)
					children := g.Dependents(tt.res)
					t.Logf("%s has %d parents, %d children", tt.name, len(parents), len(children))

					if diff := cmp.Diff(parents, tt.upstream, cmpopts.EquateEmpty()); diff != "" {
						t.Errorf("(-got, +want)\n%s", diff)
					}
					if diff := cmp.Diff(children, tt.downstream, cmpopts.EquateEmpty()); diff != "" {
						t.Errorf("(-got, +want)\n%s", diff)
					}
				})
			}
		}},
		{"AllSources", func(t *testing.T, g *graph.Graph) {
			got := g.Sources()
			want := []*graph.Source{src}
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		}},
		{"SourceForResource", func(t *testing.T, g *graph.Graph) {
			got := g.Source(res1)
			want := []*graph.Source{src}
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		}},
		{"NoSource", func(t *testing.T, g *graph.Graph) {
			got := g.Source(res2)
			want := []*graph.Source{}
			if diff := cmp.Diff(got, want, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, g)
		})
	}
}

func TestGraph_SetProject_multiple(t *testing.T) {
	g := graph.New()

	proj := config.Project{Name: "test"}

	err := g.SetProject(proj)
	if err != nil {
		t.Fatalf("SetProject() error = %v", err)
	}

	err = g.SetProject(proj)
	if err == nil {
		t.Fatalf("SetProject() error = %v", err)
	}
}

type mockResource struct{ Str string }

func (m *mockResource) Type() string { return "mock" }

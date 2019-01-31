package graph_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestGraph(t *testing.T) {

	sortByContent := cmpopts.SortSlices(func(a, b interface{}) bool {
		j1, err := json.Marshal(a)
		if err != nil {
			panic(err)
		}
		j2, err := json.Marshal(b)
		if err != nil {
			panic(err)
		}
		return bytes.Compare(j1, j2) > 0
	})

	g := graph.New()

	proj := config.Project{Name: "test"}
	res1 := &mockResource{Str: "a"}
	res2 := &mockResource{Str: "b"}
	res3 := &mockResource{Str: "c"}
	src := config.SourceInfo{SHA: "abc"}

	err := g.SetProject(proj)
	if err != nil {
		t.Fatalf("SetProject() error = %v", err)
	}

	g.AddResource(res1)
	g.AddResource(res2)
	g.AddResource(res3)
	g.AddDependency(graph.Reference{Parent: res1, Child: res2, ParentIndex: []int{1}, ChildIndex: []int{2}})
	g.AddDependency(graph.Reference{Parent: res1, Child: res3, ParentIndex: []int{2}, ChildIndex: []int{1}})
	g.AddDependency(graph.Reference{Parent: res2, Child: res3, ParentIndex: []int{3}, ChildIndex: []int{4}})
	g.AddSource(res1, src)

	// Project
	gotProj, ok := g.Project()
	if !ok {
		t.Fatal("Project() returned !ok")
	}
	wantProj := proj
	if diff := cmp.Diff(gotProj, wantProj); diff != "" {
		t.Errorf("Project() (-got, +want)\n%s", diff)
	}

	// All resources
	gotRes := g.Resources()
	wantRes := []resource.Definition{res1, res2, res3}
	if diff := cmp.Diff(gotRes, wantRes, sortByContent); diff != "" {
		t.Errorf("Resources() (-got, +want)\n%s", diff)
	}

	// Dependents (children)
	refTests := []struct {
		def  resource.Definition
		want []graph.Reference
	}{
		{res1, []graph.Reference{
			{Parent: res1, Child: res2, ParentIndex: []int{1}, ChildIndex: []int{2}},
			{Parent: res1, Child: res3, ParentIndex: []int{2}, ChildIndex: []int{1}},
		}},
		{res2, []graph.Reference{
			{Parent: res2, Child: res3, ParentIndex: []int{3}, ChildIndex: []int{4}},
		}},
		{res3, nil},
	}
	for _, tt := range refTests {
		t.Run(fmt.Sprintf("Dependents(%+v)", tt.def), func(t *testing.T) {
			got := g.Dependents(tt.def)
			if diff := cmp.Diff(got, tt.want, sortByContent); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		})
	}

	// Dependencies (parents)
	refTests = []struct {
		def  resource.Definition
		want []graph.Reference
	}{
		{res1, nil},
		{res2, []graph.Reference{
			{Parent: res1, Child: res2, ParentIndex: []int{1}, ChildIndex: []int{2}},
		}},
		{res3, []graph.Reference{
			{Parent: res1, Child: res3, ParentIndex: []int{2}, ChildIndex: []int{1}},
			{Parent: res2, Child: res3, ParentIndex: []int{3}, ChildIndex: []int{4}},
		}},
	}
	for _, tt := range refTests {
		t.Run(fmt.Sprintf("Dependencies(%+v)", tt.def), func(t *testing.T) {
			got := g.Dependencies(tt.def)
			if diff := cmp.Diff(got, tt.want, sortByContent); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		})
	}

	// All sources
	gotSources := g.Sources()
	wantSources := []config.SourceInfo{src}
	if diff := cmp.Diff(gotSources, wantSources); diff != "" {
		t.Errorf("Sources() (-got, +want)\n%s", diff)
	}

	// Sources for a specific resource
	gotSource := g.Source(res1)
	wantSource := []config.SourceInfo{src}
	if diff := cmp.Diff(gotSource, wantSource); diff != "" {
		t.Errorf("Source() (-got, +want)\n%s", diff)
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

func TestGraph_Project_noProject(t *testing.T) {
	g := graph.New()

	_, ok := g.Project()
	if ok != false {
		t.Errorf("Project() ok = %v, want = %v", ok, false)
	}
}

type mockResource struct{ Str string }

func (m *mockResource) Type() string { return "mock" }

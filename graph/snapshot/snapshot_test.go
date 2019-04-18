package snapshot_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/func/func/config"
	"github.com/func/func/graph/snapshot"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

func TestSnapshot_roundtrip(t *testing.T) {
	start := snapshot.Snap{
		// Nodes
		Resources: []resource.Resource{
			{Name: "foo", Def: &mockDef{Value: "foo"}},
			{Name: "bar", Def: &mockDef{Value: "bar"}},
			{Name: "baz", Def: &mockDef{Value: "baz"}},
		},
		Sources: []config.SourceInfo{
			{SHA: "123456789"},
		},

		// Edges
		ResourceSources: map[int][]int{
			0: {0},
		},
		References: []snapshot.Ref{
			{Source: 0, Target: 1, SourceIndex: []int{0}, TargetIndex: []int{1}},
		},
	}

	g, err := start.Graph()
	if err != nil {
		t.Fatalf("FromSnapshot() err = %v", err)
	}

	dot, err := dot.MarshalMulti(g, "Graph", "", "\t")
	if err != nil {
		panic(err)
	}
	t.Logf("Generated graph\n%s", string(dot))

	end := snapshot.Take(g)

	if diff := cmp.Diff(start, end); diff != "" {
		t.Errorf("Diff() (-start, +end)\n%s", diff)
	}
}

func TestFromSnapshot_errors(t *testing.T) {
	tests := []struct {
		name string
		snap snapshot.Snap
	}{
		{
			"NoResource",
			snapshot.Snap{
				Resources: nil,
				Sources:   []config.SourceInfo{{SHA: "123"}},
				ResourceSources: map[int][]int{
					0: {0}, // No resource at index 0
				},
			},
		},
		{
			"NoSourceOwner",
			snapshot.Snap{
				Resources:       []resource.Resource{{Name: "foo", Def: &mockDef{Value: "foo"}}},
				Sources:         []config.SourceInfo{{SHA: "123"}},
				ResourceSources: map[int][]int{}, // empty
			},
		},
		{
			"NoResourceSource",
			snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &mockDef{Value: "foo"}},
				},
				References: []snapshot.Ref{
					{Source: 1, Target: 0, SourceIndex: []int{0}, TargetIndex: []int{0}}, // Invalid Source
				},
			},
		},
		{
			"NoResourceTarget",
			snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &mockDef{Value: "foo"}},
				},
				References: []snapshot.Ref{
					{Source: 0, Target: 1, SourceIndex: []int{0}, TargetIndex: []int{0}}, // Invalid Target
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.snap.Graph()
			t.Log(err)
			if err == nil {
				t.Errorf("Want error")
			}
		})
	}
}

// Output not asserted as the dot marshalling will quickly change and it's not
// too relevant for this example.
func ExampleSnap_Graph() {
	// digraph {
	//   proj   -> {foo, bar}
	//   source -> foo        // sha: 123
	//   foo    -> bar        // index {0} -> {1}
	// }

	snap := snapshot.Snap{
		// Nodes
		Resources: []resource.Resource{
			{Name: "foo", Def: &mockDef{Value: "foo"}},
			{Name: "bar", Def: &mockDef{Value: "bar"}},
		},
		Sources: []config.SourceInfo{
			{SHA: "123"},
		},

		// Edges
		ResourceSources: map[int][]int{
			0: {0}, // 123 -> foo
		},
		References: []snapshot.Ref{
			{Source: 0, Target: 1, SourceIndex: []int{0}, TargetIndex: []int{1}}, // foo@0 -> bar@1
		},
	}

	g, err := snap.Graph()
	if err != nil {
		log.Fatal(err)
	}

	d, err := dot.MarshalMulti(g, "", "", "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(d))
}

func ExampleSnap_Diff() {
	snap1 := snapshot.Snap{
		Resources: []resource.Resource{
			{Name: "foo", Def: &mockDef{Value: "foo"}},
			{Name: "bar", Def: &mockDef{Value: "bar"}},
		},
		Sources: []config.SourceInfo{
			{SHA: "123"},
		},
	}

	snap2 := snapshot.Snap{
		Resources: []resource.Resource{
			{Name: "foo", Def: &mockDef{Value: "foo"}},
		},
		Sources: []config.SourceInfo{
			{SHA: "abc"},
		},
	}

	fmt.Println(snap1.Diff(snap2))
	// Output:
	// {snapshot.Snap}.Resources[1->?]:
	// 	-: resource.Resource{Name: "bar", Def: &snapshot_test.mockDef{Value: "bar"}}
	// 	+: <non-existent>
	// {snapshot.Snap}.Sources[0].SHA:
	// 	-: "123"
	// 	+: "abc"
}

type mockDef struct {
	resource.Definition
	Value string
}

func (mockDef) Type() string { return "mock" }

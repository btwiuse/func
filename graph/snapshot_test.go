package graph_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

func TestSnapshot_roundtrip(t *testing.T) {
	start := graph.Snapshot{
		// Nodes
		Resources: []resource.Definition{
			&mockResource{Value: "foo"},
			&mockResource{Value: "bar"},
			&mockResource{Value: "baz"},
		},
		Sources: []config.SourceInfo{
			{SHA: "123"},
		},

		// Edges
		ResourceSources: map[int][]int{
			0: {0},
		},
		References: []graph.SnapshotRef{
			{Source: 0, Target: 1, SourceIndex: []int{0}, TargetIndex: []int{1}},
		},
	}

	g, err := graph.FromSnapshot(start)
	if err != nil {
		t.Fatalf("FromSnapshot() err = %v", err)
	}

	dot, err := dot.MarshalMulti(g, "Graph", "", "\t")
	if err != nil {
		panic(err)
	}
	t.Logf("Generated graph\n%s", string(dot))

	end := g.Snapshot()

	if diff := cmp.Diff(start, end); diff != "" {
		t.Errorf("Diff() (-start, +end)\n%s", diff)
	}
}

func TestFromSnapshot_errors(t *testing.T) {
	tests := []struct {
		name string
		snap graph.Snapshot
	}{
		{
			"NoResource",
			graph.Snapshot{
				Resources: nil,
				Sources:   []config.SourceInfo{{SHA: "123"}},
				ResourceSources: map[int][]int{
					0: {0}, // No resource at index 0
				},
			},
		},
		{
			"NoSourceOwner",
			graph.Snapshot{
				Resources:       []resource.Definition{&mockResource{Value: "foo"}},
				Sources:         []config.SourceInfo{{SHA: "123"}},
				ResourceSources: map[int][]int{}, // empty
			},
		},
		{
			"NoResourceSource",
			graph.Snapshot{
				Resources: []resource.Definition{
					&mockResource{Value: "foo"},
				},
				References: []graph.SnapshotRef{
					{Source: 1, Target: 0, SourceIndex: []int{0}, TargetIndex: []int{0}}, // Invalid Source
				},
			},
		},
		{
			"NoResourceTarget",
			graph.Snapshot{
				Resources: []resource.Definition{
					&mockResource{Value: "foo"},
				},
				References: []graph.SnapshotRef{
					{Source: 0, Target: 1, SourceIndex: []int{0}, TargetIndex: []int{0}}, // Invalid Target
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := graph.FromSnapshot(tt.snap)
			t.Log(err)
			if err == nil {
				t.Errorf("Want error")
			}
		})
	}
}

// Output not asserted as the dot marshalling will quickly change and it's not
// too relevant for this example.
func ExampleFromSnapshot() {
	// digraph {
	//   proj   -> {foo, bar}
	//   source -> foo        // sha: 123
	//   foo    -> bar        // index {0} -> {1}
	// }

	snap := graph.Snapshot{
		// Nodes
		Resources: []resource.Definition{
			&mockResource{Value: "foo"},
			&mockResource{Value: "bar"},
		},
		Sources: []config.SourceInfo{
			{SHA: "123"},
		},

		// Edges
		ResourceSources: map[int][]int{
			0: {0}, // 123 -> foo
		},
		References: []graph.SnapshotRef{
			{Source: 0, Target: 1, SourceIndex: []int{0}, TargetIndex: []int{1}}, // foo@0 -> bar@1
		},
	}

	g, err := graph.FromSnapshot(snap)
	if err != nil {
		log.Fatal(err)
	}

	d, err := dot.MarshalMulti(g, "", "", "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(d))
}

func ExampleSnapshot_Diff() {
	snap1 := graph.Snapshot{
		Resources: []resource.Definition{
			&mockResource{Value: "foo"},
			&mockResource{Value: "bar"},
		},
		Sources: []config.SourceInfo{
			{SHA: "123"},
		},
	}

	snap2 := graph.Snapshot{
		Resources: []resource.Definition{
			&mockResource{Value: "foo"},
		},
		Sources: []config.SourceInfo{
			{SHA: "abc"},
		},
	}

	fmt.Println(snap1.Diff(snap2))
	// Output:
	// {graph.Snapshot}.Resources[1->?]:
	// 	-: &graph_test.mockResource{Value: "bar"}
	// 	+: <non-existent>
	// {graph.Snapshot}.Sources[0].SHA:
	// 	-: "123"
	// 	+: "abc"
}

type mockResource struct{ Value string }

func (mockResource) Type() string { return "mock" }

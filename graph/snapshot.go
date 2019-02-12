package graph

import (
	"sort"

	"github.com/func/func/config"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
)

// A Snapshot is a snapshot of the contents of a graph
//
// Snapshots can be used for buildings graphs in tests, or asserting the state
// of them in tests.
type Snapshot struct {
	// Nodes
	Resources []resource.Definition
	Sources   []config.SourceInfo

	// Edges
	ResourceSources map[int][]int // Resource index -> Source indices.
	References      []SnapshotRef // Dependencies between resources.
}

// A SnapshotRef is a reference in a snapshot.
//
// Unlike a graph reference, the resources are references by index, rather than
// pointer.
type SnapshotRef struct {
	Source, Target           int
	SourceIndex, TargetIndex []int
}

// FromSnapshot creates a new graph from the given snapshot.
func FromSnapshot(s Snapshot) (*Graph, error) {
	g := New()

	resLookup := make(map[int]*Resource)
	srcLookup := make(map[int]*Source)

	for i, data := range s.Resources {
		res := g.AddResource(data)
		resLookup[i] = res
	}
	for i, data := range s.Sources {
		resIndex := findParent(i, s.ResourceSources)
		if resIndex < 0 {
			return nil, errors.Errorf("no resource found for source %d", i)
		}
		res, ok := resLookup[resIndex]
		if !ok {
			return nil, errors.Errorf("add source %d to resource %d: no such resource", i, resIndex)
		}
		src := g.AddSource(res, data)
		srcLookup[i] = src
	}
	for i, ref := range s.References {
		from, ok := resLookup[ref.Source]
		if !ok {
			return nil, errors.Errorf("add reference %d: source resource %d does not exist", i, ref.Source)
		}
		to, ok := resLookup[ref.Target]
		if !ok {
			return nil, errors.Errorf("add reference %d: target resource %d does not exist", i, ref.Target)
		}
		g.AddDependency(Reference{
			Source: Field{Resource: from, Index: ref.SourceIndex},
			Target: Field{Resource: to, Index: ref.TargetIndex},
		})
	}

	return g, nil
}

func findParent(want int, edges map[int][]int) int {
	for src, targets := range edges {
		for _, got := range targets {
			if got == want {
				return src
			}
		}
	}
	return -1
}

// Snapshot takes are snapshot of the graph.
func (g *Graph) Snapshot() Snapshot {
	s := Snapshot{
		ResourceSources: make(map[int][]int),
	}

	// Nodes
	rr := g.Resources()
	sort.Slice(rr, func(i, j int) bool { return rr[i].ID() < rr[j].ID() })
	resIndex := make(map[*Resource]int)
	for _, n := range rr {
		resIndex[n] = len(s.Resources)
		s.Resources = append(s.Resources, n.Definition)
	}

	ss := g.Sources()
	sort.Slice(ss, func(i, j int) bool { return ss[i].ID() < ss[j].ID() })
	srcIndex := make(map[*Source]int)
	for _, n := range ss {
		srcIndex[n] = len(s.Sources)
		s.Sources = append(s.Sources, n.SourceInfo)
	}

	// Edges
	for _, res := range g.Resources() {
		ri := resIndex[res]

		for _, src := range res.Sources() {
			si := srcIndex[src]
			s.ResourceSources[ri] = append(s.ResourceSources[ri], si)
		}

		for _, dep := range res.Dependencies() {
			di := resIndex[dep.Source.Resource]
			s.References = append(s.References, SnapshotRef{
				Source:      di,
				Target:      ri,
				SourceIndex: dep.Source.Index,
				TargetIndex: dep.Target.Index,
			})
		}
	}

	// Sort edges
	for _, list := range s.ResourceSources {
		sort.Ints(list)
	}

	return s
}

// Diff computes the difference between two snapshots. Returns an empty string
// if the snapshots are equal.
func (s Snapshot) Diff(other Snapshot) string {
	return cmp.Diff(s, other, cmpopts.EquateEmpty())
}

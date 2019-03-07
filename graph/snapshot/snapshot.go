package snapshot

import (
	"sort"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
)

// A Snap is a snapshot of the contents of a graph
//
// Snapshots can be used for buildings graphs in tests, or asserting the state
// of them in tests.
type Snap struct {
	// Nodes
	Resources []resource.Resource
	Sources   []config.SourceInfo

	// Edges
	ResourceSources map[int][]int // Resource index -> Source indices.
	References      []Ref         // References between resources.
}

// A Ref is a reference in a snapshot.
//
// Unlike a graph reference, the resources are references by index, rather than
// pointer.
type Ref struct {
	Source, Target           int
	SourceIndex, TargetIndex []int
}

// Graph creates a new graph from the snapshot.
func (s Snap) Graph() (*graph.Graph, error) {
	g := graph.New()

	resLookup := make(map[int]*graph.Resource)
	srcLookup := make(map[int]*graph.Source)

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
		g.AddDependency(graph.Reference{
			Source: graph.Field{Resource: from, Index: ref.SourceIndex},
			Target: graph.Field{Resource: to, Index: ref.TargetIndex},
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

// Take takes are snapshot of the graph.
func Take(g *graph.Graph) Snap {
	s := Snap{
		ResourceSources: make(map[int][]int),
	}

	// Nodes
	rr := g.Resources()
	sort.Slice(rr, func(i, j int) bool { return rr[i].ID() < rr[j].ID() })
	resIndex := make(map[*graph.Resource]int)
	for _, n := range rr {
		resIndex[n] = len(s.Resources)
		s.Resources = append(s.Resources, n.Config)
	}

	ss := g.Sources()
	sort.Slice(ss, func(i, j int) bool { return ss[i].ID() < ss[j].ID() })
	srcIndex := make(map[*graph.Source]int)
	for _, n := range ss {
		srcIndex[n] = len(s.Sources)
		s.Sources = append(s.Sources, n.Config)
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
			s.References = append(s.References, Ref{
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
func (s Snap) Diff(other Snap) string {
	return cmp.Diff(s, other, cmpopts.EquateEmpty())
}

package snapshot

import (
	"fmt"
	"sort"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
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
	Dependencies    map[Expr]Expr // Dependencies between resources.
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
	for target, expr := range s.Dependencies {
		fields := target.Fields()
		if len(fields) != 1 {
			return nil, errors.Errorf("%s must contain one target", target)
		}
		if err := g.AddDependency(fields[0], expr); err != nil {
			return nil, errors.Wrap(err, "add dependency")
		}
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
		Dependencies:    make(map[Expr]Expr),
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
			to := ExprFrom(dep.Target)
			data := make(map[graph.Field]interface{})
			for _, f := range dep.Expr.Fields() {
				data[f] = string(ExprFrom(f))
			}
			var exprStr string
			if err := dep.Expr.Eval(data, &exprStr); err != nil {
				panic(fmt.Sprintf("evaluate expression %s: %v", dep.Expr, err))
			}
			s.Dependencies[to] = Expr(exprStr)
		}
	}

	// Sort edges
	for _, list := range s.ResourceSources {
		sort.Ints(list)
	}

	return s
}

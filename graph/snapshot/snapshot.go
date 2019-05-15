package snapshot

import (
	"fmt"
	"sort"

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

	// Edges
	Dependencies map[Expr]Expr // Dependencies between resources.
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

	for i, data := range s.Resources {
		res := g.AddResource(data)
		resLookup[i] = res
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

// Take takes are snapshot of the graph.
func Take(g *graph.Graph) Snap {
	s := Snap{
		Dependencies: make(map[Expr]Expr),
	}

	// Nodes
	rr := g.Resources()
	sort.Slice(rr, func(i, j int) bool { return rr[i].ID() < rr[j].ID() })
	resIndex := make(map[*graph.Resource]int)
	for _, n := range rr {
		resIndex[n] = len(s.Resources)
		s.Resources = append(s.Resources, n.Config)
	}

	// Edges
	for _, res := range g.Resources() {
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

	return s
}

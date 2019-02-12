package graph

import (
	"github.com/func/func/config"
	"github.com/func/func/resource"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/multi"
)

// A Graph maintains the resources and their dependency order.
//
// The Graph should be created with New().
type Graph struct {
	*multi.DirectedGraph
}

// New creates a new graph.
func New() *Graph {
	return &Graph{
		DirectedGraph: multi.NewDirectedGraph(),
	}
}

// AddResource adds a new resource definition to the graph.
func (g *Graph) AddResource(def resource.Definition) *Resource {
	res := &Resource{
		g:          g,
		Node:       g.NewNode(),
		Definition: def,
	}
	g.AddNode(res)
	return res
}

// AddSource adds a source input to a given resource. The resource must be
// added to the graph before adding source.
func (g *Graph) AddSource(res *Resource, info config.SourceInfo) *Source {
	n := &Source{
		g:          g,
		Node:       g.NewNode(),
		SourceInfo: info,
	}
	g.AddNode(n)
	g.SetLine(g.NewLine(n, res))
	return n
}

// AddDependency adds a dependency between two resources. Both resources in the
// reference must have been added to the graph.
func (g *Graph) AddDependency(reference Reference) {
	g.SetLine(&ref{
		Line:      g.NewLine(reference.Source.Resource, reference.Target.Resource),
		Reference: reference,
	})
}

// Resources returns all resources in the graph.
//
// The order of the results is not deterministic.
func (g *Graph) Resources() []*Resource {
	var list []*Resource
	it := g.Nodes()
	for it.Next() {
		if x, ok := it.Node().(*Resource); ok {
			list = append(list, x)
		}
	}
	return list
}

// Sources returns all sources in the graph.
//
// The order of the results is not deterministic.
func (g *Graph) Sources() []*Source {
	var list []*Source
	it := g.Nodes()
	for it.Next() {
		if src, ok := it.Node().(*Source); ok {
			list = append(list, src)
		}
	}
	return list
}

func (g *Graph) linesFrom(node graph.Node) []graph.Line {
	var lines []graph.Line
	it := g.From(node.ID())
	for it.Next() {
		childID := it.Node().ID()
		if e, ok := g.Edge(node.ID(), childID).(multi.Edge); ok {
			for e.Lines.Next() {
				lines = append(lines, e.Lines.Line())
			}
		}
	}
	return lines
}

func (g *Graph) linesTo(node graph.Node) []graph.Line {
	var lines []graph.Line
	it := g.To(node.ID())
	for it.Next() {
		parentID := it.Node().ID()
		if e, ok := g.Edge(parentID, node.ID()).(multi.Edge); ok {
			for e.Lines.Next() {
				lines = append(lines, e.Lines.Line())
			}
		}
	}
	return lines
}

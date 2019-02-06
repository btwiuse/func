package graph

import (
	"errors"
	"sort"

	"github.com/func/func/config"
	"github.com/func/func/resource"
	"gonum.org/v1/gonum/graph/multi"
)

// A Graph maintains the resources and their dependency order.
//
// The Graph should be created with New().
type Graph struct {
	*multi.DirectedGraph
	project *Project

	// resources added before a project.
	pendingResources []*Resource
}

// New creates a new graph.
func New() *Graph {
	return &Graph{
		DirectedGraph: multi.NewDirectedGraph(),
	}
}

// SetProject sets the project node, which is the root node for everything in
// the graph.
//
// If resources were added before the project was added, the resources are
// connected to the project node.
func (g *Graph) SetProject(project config.Project) error {
	if g.project != nil {
		return errors.New("project already set")
	}

	g.project = &Project{
		Node:    g.NewNode(),
		Project: project,
	}
	g.AddNode(g.project)

	// Connect any existing resources to the project
	for _, res := range g.pendingResources {
		g.SetLine(g.NewLine(g.project, res))
	}

	return nil
}

// Project returns the project set in the graph. Returns nil if no project has
// been set.
func (g *Graph) Project() *Project {
	return g.project
}

// AddResource adds a new resource to the graph.
func (g *Graph) AddResource(def resource.Definition) *Resource {
	res := &Resource{
		Node:       g.NewNode(),
		Definition: def,
	}
	g.AddNode(res)

	if g.project != nil {
		g.SetLine(g.NewLine(g.project, res))
	} else {
		g.pendingResources = append(g.pendingResources, res)
	}
	return res
}

// AddSource adds a source input to a given resource. The resource must be
// added to the graph before adding source.
func (g *Graph) AddSource(res *Resource, info config.SourceInfo) *Source {
	n := &Source{
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
// Resources are returned in the order they were added to the graph.
func (g *Graph) Resources() []*Resource {
	var list []*Resource
	it := g.Nodes()
	for it.Next() {
		if r, ok := it.Node().(*Resource); ok {
			list = append(list, r)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID() < list[j].ID() })
	return list
}

func (g *Graph) refs(parentID, childID int64) []Reference {
	var refs []Reference
	if e, ok := g.Edge(parentID, childID).(multi.Edge); ok {
		for e.Lines.Next() {
			r, ok := e.Lines.Line().(*ref)
			if !ok {
				continue
			}
			refs = append(refs, r.Reference)
		}
	}
	return refs
}

func sortRefs(refs []Reference) {
	key := func(r Reference) int64 {
		return r.Source.Resource.ID()<<8 + r.Target.Resource.ID()
	}
	sort.Slice(refs, func(i, j int) bool {
		return key(refs[i]) < key(refs[j])
	})
}

// Dependencies returns the parent resources that a given resource depends on.
//
// The results are sorted in a deterministic order but the order itself should
// not be relied on.
func (g *Graph) Dependencies(res *Resource) []Reference {
	var list []Reference
	it := g.To(res.ID())
	for it.Next() {
		parent := it.Node()
		list = append(list, g.refs(parent.ID(), res.ID())...)
	}
	sortRefs(list)
	return list
}

// Dependents returns the child resources that are dependent on the given
// resource.
//
// The results are sorted in a deterministic order but the order itself should
// not be relied on.
func (g *Graph) Dependents(res *Resource) []Reference {
	var list []Reference
	it := g.From(res.ID())
	for it.Next() {
		child := it.Node()
		list = append(list, g.refs(res.ID(), child.ID())...)
	}
	sortRefs(list)
	return list
}

// Source returns all source code inputs for a given resource.
//
// The results are sorted by when they were added to the graph.
func (g *Graph) Source(res *Resource) []*Source {
	var list []*Source
	it := g.To(res.ID())
	for it.Next() {
		if src, ok := it.Node().(*Source); ok {
			list = append(list, src)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID() < list[j].ID() })
	return list
}

// Sources returns all sources in the graph.
//
// The results are sorted by when they were added to the graph.
func (g *Graph) Sources() []*Source {
	var list []*Source
	it := g.Nodes()
	for it.Next() {
		if src, ok := it.Node().(*Source); ok {
			list = append(list, src)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID() < list[j].ID() })
	return list
}

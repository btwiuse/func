package graph

import (
	"errors"

	"github.com/func/func/config"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/multi"
)

// A Graph maintains the resources and their dependency order.
//
// The Graph should be created with New().
type Graph struct {
	*multi.DirectedGraph
	resources map[Resource]*resourceNode
	project   *projectNode
}

// New creates a new graph.
func New() *Graph {
	return &Graph{
		DirectedGraph: multi.NewDirectedGraph(),
		resources:     make(map[Resource]*resourceNode),
	}
}

type projectNode struct {
	graph.Node
	config.Project
}

// SetProject sets the project node, which is the root node for everything in
// the graph.
//
// If resources were added after the project was added, the resources are
// connected to the project node.
func (g *Graph) SetProject(project config.Project) error {
	if g.project != nil {
		return errors.New("project already set")
	}

	g.project = &projectNode{
		Node:    g.NewNode(),
		Project: project,
	}
	g.AddNode(g.project)

	// Connect any existing resources to the project
	for _, res := range g.resources {
		g.SetLine(g.NewLine(g.project, res))
	}

	return nil
}

// Project returns the project set in the graph. Returns false if no project has
// been set.
func (g *Graph) Project() (config.Project, bool) {
	if g.project == nil {
		return config.Project{}, false
	}
	return g.project.Project, true
}

type resourceNode struct {
	graph.Node
	Resource
}

// AddResource adds a new resource to the graph.
func (g *Graph) AddResource(resource Resource) {
	n := &resourceNode{
		Node:     g.NewNode(),
		Resource: resource,
	}
	g.AddNode(n)
	g.resources[resource] = n

	if g.project != nil {
		g.SetLine(g.NewLine(g.project, n))
	}
}

type sourceNode struct {
	graph.Node
	config.SourceInfo
}

// AddSource adds a source input to a given resource. The resource must be
// added to the graph before adding source.
func (g *Graph) AddSource(resource Resource, info config.SourceInfo) {
	resNode := g.resources[resource]
	n := &sourceNode{
		Node:       g.NewNode(),
		SourceInfo: info,
	}
	g.AddNode(n)
	g.SetLine(g.NewLine(n, resNode))
}

type ref struct {
	graph.Line
	Reference
}

// A Reference describes a dependency relationship for a single field between
// two resources.
type Reference struct {
	Parent      Resource
	ParentIndex []int

	Child      Resource
	ChildIndex []int
}

// AddDependency adds a dependency between two resources. Both resources in the
// reference must have been added to the graph.
func (g *Graph) AddDependency(reference Reference) {
	from := g.resources[reference.Parent]
	to := g.resources[reference.Child]
	g.SetLine(&ref{
		Line:      g.NewLine(from, to),
		Reference: reference,
	})
}

// Resources returns all resources in the graph.
//
// The order of the returned results is not deterministic.
func (g *Graph) Resources() []Resource {
	list := make([]Resource, 0, len(g.resources))
	for _, r := range g.resources {
		list = append(list, r.Resource)
	}
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

// Dependencies returns the parent resources that a given resource depends on.
// Panics if the given resource does not exist in the graph.
//
// The order of the returned results is not deterministic.
func (g *Graph) Dependencies(resource Resource) []Reference {
	var list []Reference
	child := g.resources[resource]
	it := g.To(child.ID())
	for it.Next() {
		parent := it.Node()
		list = append(list, g.refs(parent.ID(), child.ID())...)
	}
	return list
}

// Dependents returns the child resources that are dependent on the given
// resource. Panics if the given resource does not exist in the graph.
//
// The order of the returned results is not deterministic.
func (g *Graph) Dependents(resource Resource) []Reference {
	var list []Reference
	parent := g.resources[resource]
	it := g.From(parent.ID())
	for it.Next() {
		child := it.Node()
		list = append(list, g.refs(parent.ID(), child.ID())...)
	}
	return list
}

// Source returns all source code inputs for a given resource. Panics if the
// given resource does not exist in the graph.
//
// The order of the returned results is not deterministic.
func (g *Graph) Source(resource Resource) []config.SourceInfo {
	n := g.resources[resource]
	var list []config.SourceInfo
	it := g.To(n.ID())
	for it.Next() {
		if src, ok := it.Node().(*sourceNode); ok {
			list = append(list, src.SourceInfo)
		}
	}
	return list
}

// Sources returns all sources in the graph.
//
// The order of the returned results is not deterministic.
func (g *Graph) Sources() []config.SourceInfo {
	var list []config.SourceInfo
	it := g.Nodes()
	for it.Next() {
		if src, ok := it.Node().(*sourceNode); ok {
			list = append(list, src.SourceInfo)
		}
	}
	return list
}

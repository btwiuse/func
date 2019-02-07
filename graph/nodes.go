package graph

import (
	"github.com/func/func/config"
	"github.com/func/func/resource"
	"gonum.org/v1/gonum/graph"
)

// A Project is a project node in the graph.
type Project struct {
	graph.Node
	g *Graph
	config.Project
}

// Resources returns all resources that belong to a project.
func (n *Project) Resources() []*Resource {
	var ret []*Resource
	for _, l := range n.g.linesFrom(n) {
		if r, ok := l.To().(*Resource); ok {
			ret = append(ret, r)
		}
	}
	return ret
}

// A Resource is an instance of a resource definition added to the graph.
type Resource struct {
	graph.Node
	g *Graph
	resource.Definition
}

// Project returns the resource's project.
func (n *Resource) Project() *Project {
	for _, l := range n.g.linesTo(n) {
		if p, ok := l.From().(*Project); ok {
			return p
		}
	}
	// In practice this should not happen, a resource node cannot exist in the
	// graph without being attached to a project.
	return nil
}

// Sources return all sources belonging to a resource.
func (n *Resource) Sources() []*Source {
	var ret []*Source
	for _, l := range n.g.linesTo(n) {
		if x, ok := l.From().(*Source); ok {
			ret = append(ret, x)
		}
	}
	return ret
}

// Dependencies returns references containing edges to parent resources.
//
//   A -> B
//
//   A is a dependency of B.
func (n *Resource) Dependencies() []Reference {
	var list []Reference
	for _, l := range n.g.linesTo(n) {
		if x, ok := l.(*ref); ok {
			list = append(list, x.Reference)
		}
	}
	return list
}

// Dependents returns references containing edges to child resources that
// depend on the resource.
//
//   A -> B
//
//   B is a dependent on A.
func (n *Resource) Dependents() []Reference {
	var list []Reference
	for _, l := range n.g.linesFrom(n) {
		if x, ok := l.(*ref); ok {
			list = append(list, x.Reference)
		}
	}
	return list
}

// A Source node contains the source code for a resource.
type Source struct {
	graph.Node
	g *Graph
	config.SourceInfo
}

// Resource returns the resource the source belongs to.
func (n *Source) Resource() *Resource {
	for _, l := range n.g.linesFrom(n) {
		if p, ok := l.To().(*Resource); ok {
			return p
		}
	}
	// In practice this should not happen, a source node cannot exist in the
	// graph without being attached to a resource.
	return nil
}

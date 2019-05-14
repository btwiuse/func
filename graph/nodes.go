package graph

import (
	"fmt"

	"github.com/func/func/config"
	"github.com/func/func/resource"
	"gonum.org/v1/gonum/graph/encoding"
)

// A Resource is an instance of a resource definition added to the graph.
type Resource struct {
	id     int64
	graph  *Graph
	Config resource.Resource
}

// ID returns the unique identifier for a resource node.
func (n *Resource) ID() int64 { return n.id }

// Attributes returns attributes for the node when the graph is marshalled to
// graphviz dot format.
func (n *Resource) Attributes() []encoding.Attribute {
	return []encoding.Attribute{
		{Key: "label", Value: fmt.Sprintf("Resource\n%s.%s", n.Config.Type, n.Config.Name)},
	}
}

// Sources return all sources belonging to a resource.
func (n *Resource) Sources() []*Source {
	var ret []*Source
	for _, l := range n.graph.linesTo(n) {
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
func (n *Resource) Dependencies() []*Dependency {
	var list []*Dependency
	for _, l := range n.graph.linesTo(n) {
		if x, ok := l.From().(*Dependency); ok {
			list = append(list, x)
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
func (n *Resource) Dependents() []*Dependency {
	var list []*Dependency
	for _, l := range n.graph.linesFrom(n) {
		if x, ok := l.To().(*Dependency); ok {
			list = append(list, x)
		}
	}
	return list
}

// A Dependency is a dependency between two or more resources.
type Dependency struct {
	id     int64
	graph  *Graph
	Target Field
	Expr   Expression
}

// ID returns the unique identifier for a dependency node.
func (n *Dependency) ID() int64 { return n.id }

// Attributes returns attributes for the node when the graph is marshalled to
// graphviz dot format.
func (n *Dependency) Attributes() []encoding.Attribute {
	return []encoding.Attribute{
		{Key: "label", Value: fmt.Sprintf("Dependency\n%s", n.Target.String())},
	}
}

// Parents returns the parent resources that are referenced from the
// dependency.
func (n *Dependency) Parents() []*Resource {
	var list []*Resource
	for _, l := range n.graph.linesTo(n) {
		if x, ok := l.From().(*Resource); ok {
			list = append(list, x)
		}
	}
	return list
}

// Child returns the child resource that will receive the value when the
// dependency is resolved.
func (n *Dependency) Child() *Resource {
	for _, l := range n.graph.linesFrom(n) {
		if x, ok := l.To().(*Resource); ok {
			return x
		}
	}
	return nil
}

// A Source node contains the source code for a resource.
type Source struct {
	id     int64
	graph  *Graph
	Config config.SourceInfo
}

// ID returns the unique identifier for a source node.
func (n *Source) ID() int64 { return n.id }

// Attributes returns attributes for the node when the graph is marshalled to
// graphviz dot format.
func (n *Source) Attributes() []encoding.Attribute {
	return []encoding.Attribute{
		{Key: "label", Value: fmt.Sprintf("Source\n%s", n.Config.Key[:7])},
	}
}

// Resource returns the resource the source belongs to.
func (n *Source) Resource() *Resource {
	for _, l := range n.graph.linesFrom(n) {
		if p, ok := l.To().(*Resource); ok {
			return p
		}
	}
	// In practice this should not happen, a source node cannot exist in the
	// graph without being attached to a resource.
	return nil
}

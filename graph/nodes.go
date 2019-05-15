package graph

import (
	"fmt"

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

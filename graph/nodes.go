package graph

import (
	"github.com/func/func/config"
	"github.com/func/func/resource"
	"gonum.org/v1/gonum/graph"
)

// A Project is a project node in the graph.
type Project struct {
	graph.Node
	config.Project
}

// A Resource is an instance of a resource definition added to the graph.
type Resource struct {
	graph.Node
	resource.Definition
}

// A Source node contains the source code for a resource.
type Source struct {
	graph.Node
	config.SourceInfo
}

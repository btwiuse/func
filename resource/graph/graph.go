package graph

import (
	"fmt"

	"github.com/func/func/resource"
	"github.com/zclconf/go-cty/cty"
)

// A Graph contains a resource graph of user defined configurations for
// resources and their dependencies.
type Graph struct {
	Resources    map[string]*resource.Resource
	Dependencies map[string][]Dependency
}

// New creates a new empty graph.
func New() *Graph {
	return &Graph{
		Resources:    make(map[string]*resource.Resource),
		Dependencies: make(map[string][]Dependency),
	}
}

// AddResource adds a new resource to the graph.
//
// Panics if the name is blank, or if the resource has no type. These must be
// checked by the calling code before adding a resource, failing to do so
// indicates a bug in the calling code.
//
// The resource is allowed to contain a parent reference that does not (yet)
// exist.
//
// If another resource with the same name is added, it overwrites the existing
// resource and likely makes dependencies to it not match, unless the new
// resource has an identical type. No checking is done for this, it is the
// responsibility of the caller to ensure this does not happen.
// Replacing an existing resource is likely the wrong thing to do. Instead, the
// pending resource should be kept in memory, and only added to the graph when
// it is final.
func (g *Graph) AddResource(res *resource.Resource) {
	if res.Name == "" {
		panic("Resource has no name")
	}
	if res.Type == "" {
		panic("Resource has no type")
	}
	g.Resources[res.Name] = res
}

// AddDependency adds a dependency to a resource.
//
// Panics if a resource with the given name does not exist. This indicates a
// bug in the calling code.
//
// The dependency is checked for invalid references to resources (that do not
// exist). Failing this precondition will cause a panic. Beyond that, no
// validation is done on the dependency (such as ensuring the field exists).
func (g *Graph) AddDependency(resourceName string, dep Dependency) {
	if _, ok := g.Resources[resourceName]; !ok {
		panic("Resource does not exist")
	}
	for i, r := range dep.Expression.References() {
		attr, ok := r[0].(cty.GetAttrStep)
		if !ok {
			panic(fmt.Sprintf("Reference %d in expression does not start with resource name", i))
		}
		if _, ok := g.Resources[attr.Name]; !ok {
			panic(fmt.Sprintf("Cannot add reference to non-existing resource %q", attr.Name))
		}
	}
	g.Dependencies[resourceName] = append(g.Dependencies[resourceName], dep)
}

// LeafResources returns all resources that have no children. The results are
// returned in an arbitrary order.
func (g *Graph) LeafResources() []string {
	parents := make(map[string]struct{})

	// Mark resources that are dependencies to child resources.
	// For every dependency:
	for _, deps := range g.Dependencies {
		// For every dependency for a resource:
		for _, d := range deps {
			// For every reference in the dependency expression:
			for _, r := range d.Expression.References() {
				// Safe to assert, check was done when adding dependency.
				name := r[0].(cty.GetAttrStep).Name
				parents[name] = struct{}{}
			}
		}
	}

	// Collect remaining resources that were not marked.
	out := make([]string, 0, len(g.Resources)-len(parents))
	for name := range g.Resources {
		_, isParent := parents[name]
		if !isParent {
			out = append(out, name)
		}
	}
	return out

}

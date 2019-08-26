package resource

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// A Graph contains a resources and relationships between resources.
type Graph struct {
	Resources    []*Resource
	Dependencies []*Dependency
}

// AddResource adds a new resource to the graph.
//
// Returns an error if another resource with an identical name already exists.
func (g *Graph) AddResource(res *Resource) error {
	if res.Name == "" {
		return fmt.Errorf("resource has no name")
	}
	if res.Type == "" {
		return fmt.Errorf("resource has no type")
	}
	if ex := g.Resource(res.Name); ex != nil {
		return fmt.Errorf("resource %q already exists", res.Name)
	}
	g.Resources = append(g.Resources, res)
	return nil
}

// Resource returns a resource with a given name from the graph.
// Returns nil if the resource does not exist.
func (g *Graph) Resource(name string) *Resource {
	for _, r := range g.Resources {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// AddDependency adds a dependency to a resource.
//
// The dependency is checked for invalid references to resources (that do not
// exist). Failing this precondition will return an error. Beyond that, no
// validation is done on the dependency (such as ensuring the field exists).
func (g *Graph) AddDependency(dep *Dependency) error {
	if res := g.Resource(dep.Child); res == nil {
		return fmt.Errorf("child resource does not exist")
	}
	for i, r := range dep.Expression.References() {
		attr, ok := r[0].(cty.GetAttrStep)
		if !ok {
			return fmt.Errorf("reference %d in expression does not start with resource name", i)
		}
		if res := g.Resource(attr.Name); res == nil {
			return fmt.Errorf("cannot add reference %d to non-existing resource %q", i, attr.Name)
		}
	}
	g.Dependencies = append(g.Dependencies, dep)
	return nil
}

// ParentResources returns the parent resources that are are a dependency to
// the given child resource. In case multiple references exist to the parent
// resource, it is included only once.
func (g *Graph) ParentResources(child string) []*Resource {
	added := make(map[string]struct{}) // Avoid adding the same dependency twice
	var parents []*Resource
	for _, d := range g.Dependencies {
		if d.Child != child {
			continue
		}
		for _, parent := range d.Parents() {
			if _, ok := added[parent]; ok {
				continue
			}
			parents = append(parents, g.Resource(parent))
			added[parent] = struct{}{}
		}
	}
	return parents
}

// DependenciesOf returns the dependencies for a given child.
func (g *Graph) DependenciesOf(child string) []*Dependency {
	var deps []*Dependency
	for _, d := range g.Dependencies {
		if d.Child == child {
			deps = append(deps, d)
		}
	}
	return deps
}

// LeafResources returns all resources that have no children.
func (g *Graph) LeafResources() []*Resource {
	parents := make(map[string]struct{})

	// Mark resources that are dependencies to child resources.
	// For every dependency:
	for _, d := range g.Dependencies {
		// For every reference in the dependency expression:
		for _, r := range d.Expression.References() {
			// Safe to assert, check was done when adding dependency.
			name := r[0].(cty.GetAttrStep).Name
			parents[name] = struct{}{}
		}
	}

	// Collect remaining resources that were not marked.
	out := make([]*Resource, 0, len(g.Resources)-len(parents))
	for _, res := range g.Resources {
		_, isParent := parents[res.Name]
		if !isParent {
			out = append(out, res)
		}
	}
	return out

}

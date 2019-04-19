package graph

import (
	"github.com/func/func/config"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
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
func (g *Graph) AddResource(res resource.Resource) *Resource {
	node := &Resource{
		id:     g.NewNode().ID(),
		graph:  g,
		Config: res,
	}
	g.AddNode(node)
	return node
}

// AddSource adds a source input to a given resource. The resource must be
// added to the graph before adding source.
func (g *Graph) AddSource(res *Resource, info config.SourceInfo) *Source { // nolint: interfacer
	node := &Source{
		id:     g.NewNode().ID(),
		graph:  g,
		Config: info,
	}
	g.AddNode(node)
	g.SetLine(g.NewLine(node, res))
	return node
}

// A Field represents the address to a specific field in a resource with a
// certain type and name.
type Field struct {
	Type  string // Resource type.
	Name  string // Resource name.
	Field string // Field name based on input/output name.
}

func (f Field) String() string {
	return f.Type + "." + f.Name + "." + f.Field
}

// An Expression describes a dynamic or static value for a field.
type Expression interface {
	// Fields returns the dependent fields that must be resolved in order to be
	// able to evaluate the expression.
	Fields() []Field

	// Eval evaluates the expression into v. The data will contain fields
	// matching the ones returned from Fields().
	Eval(data map[Field]interface{}, v interface{}) error
}

// AddDependency adds a dependency to the given resource. This ensures the
// dependency is resolved first, and that the value is passed into the
// resource.
//
// The expr is the expression to resolve and set on the target field. The
// parent resources must be resolved first are retrieved from the expression.
func (g *Graph) AddDependency(target Field, expr Expression) error {
	node := &Dependency{
		id:     g.NewNode().ID(),
		graph:  g,
		Target: target,
		Expr:   expr,
	}
	g.AddNode(node)

	res := g.LookupResource(target.Type, target.Name)
	if res == nil {
		return errors.Errorf("target resource %s.%s does not exist", target.Type, target.Name)
	}
	g.SetLine(g.NewLine(node, res))

	fields := expr.Fields()
	for _, f := range fields {
		parent := g.LookupResource(f.Type, f.Name)
		if parent == nil {
			return errors.Errorf("expression has a dependency on %s.%s, which does not exist", f.Type, f.Name)
		}
		g.SetLine(g.NewLine(parent, node))
	}

	return nil
}

// LookupResource finds a resource with a certain type and name. Returns nil if
// no such resource exists in the graph.
func (g *Graph) LookupResource(typeName, resourceName string) *Resource {
	it := g.Nodes()
	for it.Next() {
		res, ok := it.Node().(*Resource)
		if !ok {
			continue
		}
		if res.Config.Def.Type() == typeName && res.Config.Name == resourceName {
			return res
		}
	}
	return nil
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

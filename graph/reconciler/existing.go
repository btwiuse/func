package reconciler

import (
	"fmt"
	"sync"

	"github.com/func/func/resource"
	"github.com/pkg/errors"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type existingResources struct {
	*simple.DirectedGraph

	mu   sync.RWMutex
	keep map[int64]bool
}

type existing struct {
	graph.Node
	res  resource.Resource
	hash string
}

func newExisting(resources []resource.Resource) (*existingResources, error) {
	ee := &existingResources{
		DirectedGraph: simple.NewDirectedGraph(),
		keep:          make(map[int64]bool),
	}

	lookup := make(map[resource.Dependency]graph.Node)
	for _, r := range resources {
		node := &existing{
			Node: ee.NewNode(),
			res:  r,
			hash: resource.Hash(r.Def),
		}
		ee.AddNode(node)

		child := resource.Dependency{Type: r.Def.Type(), Name: r.Name}
		lookup[child] = node
	}

	for _, r := range resources {
		childDep := resource.Dependency{Type: r.Def.Type(), Name: r.Name}
		child, ok := lookup[childDep]
		if !ok {
			return nil, errors.Errorf("No resource found for child %s", childDep)
		}

		for _, dep := range r.Deps {
			parent, ok := lookup[dep]
			if !ok {
				return nil, errors.Errorf("No resource found for parent %s", dep)
			}
			ee.SetEdge(ee.NewEdge(parent, child))
		}
	}
	return ee, nil
}

func (ee *existingResources) Find(typename, name string) *existing {
	ee.mu.RLock()
	defer ee.mu.RUnlock()
	it := ee.Nodes()
	for it.Next() {
		e := it.Node().(*existing)
		if e.res.Def.Type() == typename && e.res.Name == name {
			return e
		}
	}
	return nil
}

func (ee *existingResources) Keep(ex *existing) {
	ee.mu.Lock()
	ee.keep[ex.ID()] = true
	ee.mu.Unlock()
}

func (ee *existingResources) Remaining() []*existing {
	sorted, err := topo.Sort(ee)
	if err != nil {
		// An error will only be returned if the graph is not sortable, which
		// should never happen.
		panic(fmt.Sprintf("Cyclical existing resources: %v", err))
	}
	reverse(sorted)

	var ret []*existing
	for _, n := range sorted {
		if !ee.keep[n.ID()] {
			ret = append(ret, n.(*existing))
		}
	}
	return ret
}

func reverse(nodes []graph.Node) {
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}

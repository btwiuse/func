package graph_test

import (
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"
)

var opts = []cmp.Option{
	cmpopts.EquateEmpty(),
	cmp.Transformer("GoString", func(v cty.Value) string { return v.GoString() }),
	cmp.Transformer("Name", func(v cty.GetAttrStep) string { return v.Name }),
	cmp.Transformer("GoString", func(v cty.IndexStep) string { return v.GoString() }),
}

func TestGraph_AddResource(t *testing.T) {
	defer checkPanic(t, false)

	g := graph.New()
	a := &resource.Resource{Type: "foo", Name: "a"}
	b := &resource.Resource{Type: "bar", Name: "b"}
	g.AddResource(a)
	g.AddResource(b)

	want := &graph.Graph{
		Resources: map[string]*resource.Resource{
			"a": a,
			"b": b,
		},
	}
	if diff := cmp.Diff(g, want, opts...); diff != "" {
		t.Errorf("Resources not added (-got, +want)\n%s", diff)
	}
}

func TestGraph_AddResource_panicNoName(t *testing.T) {
	defer checkPanic(t, true)

	g := graph.New()
	g.AddResource(&resource.Resource{
		Type: "foo",
		Name: "", // No name
	})
}

func TestGraph_AddResource_panicNoType(t *testing.T) {
	defer checkPanic(t, true)

	g := graph.New()
	g.AddResource(&resource.Resource{
		Type: "", // No type
		Name: "foo",
	})
}

func TestGraph_AddResource_noPanicNonExistingParent(t *testing.T) {
	defer checkPanic(t, false)

	g := graph.New()
	g.AddResource(&resource.Resource{
		Type: "foo",
		Name: "foo",
		Deps: []string{"nonexisting"},
	})
}

func TestGraph_AddDependency(t *testing.T) {
	defer checkPanic(t, false)

	g := graph.New()
	a := &resource.Resource{Type: "foo", Name: "a"}
	b := &resource.Resource{Type: "bar", Name: "b"}
	g.AddResource(a)
	g.AddResource(b)
	dep := graph.Dependency{
		Field: cty.GetAttrPath("input"),
		Expression: graph.Expression{
			graph.ExprReference{
				Path: cty.GetAttrPath("a").GetAttr("output").Index(cty.NumberIntVal(2)),
			},
		},
	}
	g.AddDependency("b", dep)

	want := &graph.Graph{
		Resources: map[string]*resource.Resource{
			"a": a,
			"b": b,
		},
		Dependencies: map[string][]graph.Dependency{
			"b": {dep},
		},
	}
	if diff := cmp.Diff(g, want, opts...); diff != "" {
		t.Errorf("Dependencies do not match (-got, +want)\n%s", diff)
	}
}

func TestGraph_AddDependency_panicNonExisting(t *testing.T) {
	defer checkPanic(t, true)

	g := graph.New()
	g.AddResource(&resource.Resource{Type: "foo"})
	g.AddDependency("bar", graph.Dependency{ // Target does not exist
		Expression: graph.Expression{
			graph.ExprReference{
				Path: cty.GetAttrPath("nonexisting").GetAttr("output"),
			},
		},
	})
}

func TestGraph_AddDependency_panicRefNoName(t *testing.T) {
	defer checkPanic(t, true)

	g := graph.New()
	g.AddResource(&resource.Resource{Type: "foo"})
	g.AddDependency("foo", graph.Dependency{
		Expression: graph.Expression{
			graph.ExprReference{
				Path: cty.IndexPath(cty.NumberIntVal(0)), // First part must be GetAttrStep
			},
		},
	})
}

func TestGraph_AddDependency_panicNonExistingRef(t *testing.T) {
	defer checkPanic(t, true)

	g := graph.New()
	g.AddResource(&resource.Resource{Type: "foo"})
	g.AddDependency("foo", graph.Dependency{
		Expression: graph.Expression{
			graph.ExprReference{
				Path: cty.GetAttrPath("nonexisting").GetAttr("output"), // Referenced resource does not exist
			},
		},
	})
}

func TestGraph_LeafResources(t *testing.T) {
	g := graph.New()
	g.AddResource(&resource.Resource{Type: "foo", Name: "a"})
	g.AddResource(&resource.Resource{Type: "bar", Name: "b"})
	g.AddResource(&resource.Resource{Type: "baz", Name: "c"})
	g.AddResource(&resource.Resource{Type: "qux", Name: "d"})

	// a -> b
	g.AddDependency("b", graph.Dependency{
		Field: cty.GetAttrPath("input"),
		Expression: graph.Expression{
			graph.ExprReference{
				Path: cty.GetAttrPath("a").GetAttr("output"),
			},
		},
	})

	// b -> c
	g.AddDependency("c", graph.Dependency{
		Field: cty.GetAttrPath("input"),
		Expression: graph.Expression{
			graph.ExprReference{
				Path: cty.GetAttrPath("b").GetAttr("output"),
			},
		},
	})

	got := g.LeafResources()
	want := []string{"c", "d"}

	opts := []cmp.Option{
		cmpopts.SortSlices(func(a, b string) bool { return a < b }),
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("LeafResources() (-got +want)\n%s", diff)
	}
}

func checkPanic(t *testing.T, wantPanic bool) {
	t.Helper()
	err := recover()
	if err == nil && wantPanic {
		t.Errorf("Did not panic, want panic")
	}
	if err != nil && !wantPanic {
		t.Errorf("Unexpected panic: %v", err)
	}
}

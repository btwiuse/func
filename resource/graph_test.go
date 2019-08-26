package resource

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestGraph_AddResource(t *testing.T) {
	g := &Graph{}

	a := &Resource{Type: "aaa", Name: "a"}

	if err := g.AddResource(a); err != nil {
		t.Fatalf("AddResource() err = %v", err)
	}

	want := &Graph{
		Resources: []*Resource{a},
	}
	opts := []cmp.Option{
		cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
	}
	if diff := cmp.Diff(g, want, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func TestGraph_AddResource_existing(t *testing.T) {
	g := &Graph{
		Resources: []*Resource{
			{Type: "foo", Name: "a"},
		},
	}

	err := g.AddResource(&Resource{Type: "bar", Name: "a"}) // same name
	if err == nil {
		t.Fatalf("Want error")
	}
}

func TestGraph_AddResource_ErrNoName(t *testing.T) {
	g := &Graph{}
	err := g.AddResource(&Resource{Name: "", Type: "foo"})
	if err == nil {
		t.Fatalf("Want error")
	}
}

func TestGraph_AddResource_ErrNoType(t *testing.T) {
	g := &Graph{}
	err := g.AddResource(&Resource{Name: "foo", Type: ""})
	if err == nil {
		t.Fatalf("Want error")
	}
}

func TestGraph_AddDependency(t *testing.T) {
	g := &Graph{
		Resources: []*Resource{
			{Type: "foo", Name: "a"},
			{Type: "bar", Name: "b"},
		},
	}

	dep := &Dependency{
		Child: "b",
		Field: cty.GetAttrPath("input"),
		Expression: Expression{
			ExprReference{
				Path: cty.GetAttrPath("a").GetAttr("output").Index(cty.NumberIntVal(2)),
			},
		},
	}

	if err := g.AddDependency(dep); err != nil {
		t.Fatalf("AddDependency() err = %v", err)
	}

	want := &Graph{
		Resources: []*Resource{
			{Type: "foo", Name: "a"},
			{Type: "bar", Name: "b"},
		},
		Dependencies: []*Dependency{dep},
	}
	opts := []cmp.Option{
		cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
		cmp.Comparer(func(a, b cty.Path) bool { return a.Equals(b) }),
	}
	if diff := cmp.Diff(g, want, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func TestGraph_AddDependency_ErrMissingChild(t *testing.T) {
	g := &Graph{
		Resources: []*Resource{
			{Type: "foo", Name: "a"},
			{Type: "bar", Name: "b"},
		},
	}

	dep := &Dependency{
		Child: "nonexisting",
		Field: cty.GetAttrPath("input"),
		Expression: Expression{
			ExprReference{
				Path: cty.GetAttrPath("a").GetAttr("output"),
			},
		},
	}

	err := g.AddDependency(dep)
	if err == nil {
		t.Fatalf("Want error, got %v", err)
	}
}

func TestGraph_AddDependency_ErrMissingParent(t *testing.T) {
	g := &Graph{
		Resources: []*Resource{
			{Type: "foo", Name: "a"},
			{Type: "bar", Name: "b"},
		},
	}

	dep := &Dependency{
		Child: "a",
		Field: cty.GetAttrPath("input"),
		Expression: Expression{
			ExprReference{
				Path: cty.GetAttrPath("nonexisting").GetAttr("output"),
			},
		},
	}

	err := g.AddDependency(dep)
	if err == nil {
		t.Fatalf("Want error, got %v", err)
	}
}

func TestGraph_AddDependency_ErrInvalidPath(t *testing.T) {
	g := &Graph{
		Resources: []*Resource{
			{Type: "foo", Name: "a"},
			{Type: "bar", Name: "b"},
		},
	}

	dep := &Dependency{
		Child: "a",
		Field: cty.GetAttrPath("input"),
		Expression: Expression{
			ExprReference{
				Path: cty.IndexPath(cty.NumberIntVal(1)),
			},
		},
	}

	err := g.AddDependency(dep)
	if err == nil {
		t.Fatalf("Want error, got %v", err)
	}
}

func TestGraph_ParentResources(t *testing.T) {
	a := &Resource{Type: "foo", Name: "a"}
	b := &Resource{Type: "foo", Name: "b"}

	g := &Graph{
		Resources: []*Resource{a, b},
		Dependencies: []*Dependency{
			{
				Child: "b",
				Field: cty.GetAttrPath("input"),
				Expression: Expression{
					ExprReference{
						Path: cty.GetAttrPath("a").GetAttr("output_a"), // Reference 1 to a
					},
				},
			},
			{
				Child: "b",
				Field: cty.GetAttrPath("input"),
				Expression: Expression{
					ExprReference{
						Path: cty.GetAttrPath("a").GetAttr("output_b"), // Reference 2 to a
					},
				},
			},
		},
	}

	opts := []cmp.Option{
		cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
	}

	got := g.ParentResources("a")
	want := ([]*Resource)(nil)
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Parent resources a (-got +want)\n%s", diff)
	}

	got = g.ParentResources("b")
	want = []*Resource{a}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Parent resources b (-got +want)\n%s", diff)
	}
}

func TestGraph_DependenciesOf(t *testing.T) {
	a := &Resource{Type: "foo", Name: "a"}
	b := &Resource{Type: "foo", Name: "b"}

	dep1 := &Dependency{
		Child: "b",
		Field: cty.GetAttrPath("input"),
		Expression: Expression{
			ExprReference{
				Path: cty.GetAttrPath("a").GetAttr("output_a"),
			},
		},
	}
	dep2 := &Dependency{
		Child: "b",
		Field: cty.GetAttrPath("input"),
		Expression: Expression{
			ExprReference{
				Path: cty.GetAttrPath("a").GetAttr("output_b"),
			},
		},
	}

	g := &Graph{
		Resources:    []*Resource{a, b},
		Dependencies: []*Dependency{dep1, dep2},
	}

	opts := []cmp.Option{
		cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
		cmp.Comparer(func(a, b cty.Path) bool { return a.Equals(b) }),
	}

	got := g.DependenciesOf("a")
	want := ([]*Dependency)(nil)
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Dependencies of a (-got +want)\n%s", diff)
	}

	got = g.DependenciesOf("b")
	want = []*Dependency{dep1, dep2}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Dependencies of b (-got +want)\n%s", diff)
	}
}

func TestGraph_LeafResources(t *testing.T) {
	a := &Resource{Type: "foo", Name: "a"}
	b := &Resource{Type: "bar", Name: "b"}
	c := &Resource{Type: "baz", Name: "c"}
	d := &Resource{Type: "qux", Name: "d"}
	g := &Graph{
		Resources: []*Resource{
			a, b, c, d,
		},
		Dependencies: []*Dependency{
			{
				// b has a dependency on a
				Child: "b",
				Field: cty.GetAttrPath("input"),
				Expression: Expression{
					ExprReference{
						Path: cty.GetAttrPath(a.Name).GetAttr("output"),
					},
				},
			},
			{
				// c has a dependency on b
				Child: "c",
				Field: cty.GetAttrPath("input"),
				Expression: Expression{
					ExprReference{
						Path: cty.GetAttrPath(b.Name).GetAttr("output"),
					},
				},
			},
		},
	}

	got := g.LeafResources()
	want := []*Resource{c, d} // c and d have no children

	opts := []cmp.Option{
		cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Dependencies to a (-got +want)\n%s", diff)
	}
}

package teststore_test

import (
	"context"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/storage/teststore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"
)

func TestStore_Resources(t *testing.T) {
	s := &teststore.Store{}

	project := "testproject"
	ctx := context.Background()

	resA := &resource.Deployed{
		Desired: &resource.Desired{
			Type:  "foo",
			Name:  "a",
			Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("abc")}),
		},
		ID:     "a",
		Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("def")}),
	}
	resB := &resource.Deployed{
		Desired: &resource.Desired{
			Type:    "foo",
			Name:    "b",
			Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("123")}),
			Sources: []string{"x", "y", "z"},
		},
		ID:     "b",
		Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("456")}),
		Deps:   []string{"foo", "bar"},
	}

	// Create
	if err := s.PutResource(ctx, project, resA); err != nil {
		t.Fatal(err)
	}
	if err := s.PutResource(ctx, project, resB); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListResources(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	want := []*resource.Deployed{
		resA,
		resB,
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}

	// Update
	update := &resource.Deployed{
		Desired: &resource.Desired{
			Type:  "foo",
			Name:  "abcdef", // Different name
			Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("ABC")}),
		},
		ID:     "a", // Same id
		Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("DEF")}),
	}
	if err := s.PutResource(ctx, project, update); err != nil {
		t.Fatal(err)
	}

	// Delete
	if err := s.DeleteResource(ctx, project, resB); err != nil {
		t.Fatal(err)
	}

	got, err = s.ListResources(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	want = []*resource.Deployed{
		update, // a is updated
		// b is deleted
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func TestStore_Graphs(t *testing.T) {
	s := &teststore.Store{}

	project := "testproject"
	ctx := context.Background()

	g := &resource.Graph{
		Resources: []*resource.Desired{
			{
				Name:    "alice",
				Type:    "person",
				Sources: []string{"abc"},
				Input: cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("alice"),
					"age":  cty.NumberIntVal(20),
				}),
			},
			{
				Name:    "bob",
				Type:    "person",
				Sources: []string{"abc"},
				Input: cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("bob"),
					"age":  cty.NumberIntVal(30),
				}),
			},
		},
		Dependencies: []*resource.Dependency{
			{
				Child: "bob",
				Field: cty.GetAttrPath("friends"),
				Expression: resource.Expression{
					resource.ExprReference{
						Path: cty.
							GetAttrPath("alice").
							GetAttr("friends").
							Index(cty.NumberIntVal(0)),
					},
				},
			},
		},
	}

	// Create
	if err := s.PutGraph(ctx, project, g); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetGraph(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, g, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

var opts = []cmp.Option{
	cmpopts.EquateEmpty(),
	cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
	cmp.Comparer(func(a, b cty.Path) bool { return a.Equals(b) }),
	cmp.FilterPath(func(p cmp.Path) bool {
		return p.String() == "Deps" || p.String() == "Sources" // String sets are not sorted
	}, cmpopts.SortSlices(func(a, b string) bool {
		return a < b
	})),
}

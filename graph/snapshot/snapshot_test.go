package snapshot_test

import (
	"context"
	"testing"

	"github.com/func/func/graph"
	"github.com/func/func/graph/snapshot"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

func TestSnapshot_roundtrip(t *testing.T) {
	start := snapshot.Snap{
		// Nodes
		Resources: []resource.Resource{
			{Name: "foo", Def: &mockDef{Input: "foo"}},
			{Name: "bar", Def: &mockDef{}},
			{Name: "baz", Def: &mockDef{}, Sources: []string{"123456789"}},
		},

		// Edges
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${bar.in}": "${foo.out}",
			"${baz.in}": "${foo.out}-${mock.bar.out}",
		},
	}

	g, err := start.Graph()
	if err != nil {
		t.Fatalf("FromSnapshot() err = %v", err)
	}

	dot, err := dot.MarshalMulti(g, "Graph", "", "\t")
	if err != nil {
		panic(err)
	}
	t.Logf("Generated graph\n%s", string(dot))

	end := snapshot.Take(g)

	opts := []cmp.Option{
		cmp.Transformer("GoString", func(v cty.Value) string { return v.GoString() }),
	}
	if diff := cmp.Diff(start, end, opts...); diff != "" {
		t.Errorf("Diff() (-start, +end)\n%s", diff)
	}
}

func TestFromSnapshot_errors(t *testing.T) {
	tests := []struct {
		name string
		snap snapshot.Snap
	}{
		{
			"NoDependencyParentType",
			snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &mockDef{Input: "foo"}},
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${mock.bar.in}": "${notfound.foo.out}",
				},
			},
		},
		{
			"NoDependencyParentName",
			snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &mockDef{Input: "foo"}},
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{"${mock.bar.in}": "${mock.notfound.out}"},
			},
		},
		{
			"NoDependencyParentField",
			snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &mockDef{Input: "foo"}},
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${mock.bar.in}": "${mock.foo.notfound}",
				},
			},
		},
		{
			"NoDependencyChildType",
			snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &mockDef{Input: "foo"}},
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${notfound.bar.in}": "${mock.foo.out}",
				},
			},
		},
		{
			"NoDependencyChildName",
			snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &mockDef{Input: "foo"}},
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${mock.notfound.in}": "${mock.foo.out}",
				},
			},
		},
		{
			"NoDependencyChildField",
			snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &mockDef{Input: "foo"}},
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${mock.bar.notfound}": "${mock.foo.out}",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.snap.Graph()
			t.Log(err)
			if err == nil {
				t.Errorf("Want error")
			}
		})
	}
}

type ExampleExpression struct{}

func (ExampleExpression) Eval(data map[graph.Field]interface{}, target interface{}) error { return nil }

type mockDef struct {
	Input  string
	Output string
}

func (mockDef) Type() string                                          { return "mock" }
func (mockDef) Create(context.Context, *resource.CreateRequest) error { return nil }
func (mockDef) Update(context.Context, *resource.UpdateRequest) error { return nil }
func (mockDef) Delete(context.Context, *resource.DeleteRequest) error { return nil }

package reconciler_test

import (
	"context"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
	"github.com/func/func/resource/reconciler"
	"github.com/func/func/storage/mock"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/zap/zaptest"
)

// Everything in same namespace & project
func TestReconciler_Reconcile_events(t *testing.T) {
	tests := []struct {
		name       string
		defs       map[string]resource.Definition
		existing   []resource.Resource
		graph      *graph.Graph
		wantEvents []mock.Event
	}{
		{
			name:     "Empty",
			existing: nil,
			graph: &graph.Graph{
				Resources: nil,
			},
			wantEvents: nil,
		},
		{
			name: "Nop",
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: []resource.Resource{
				{
					Name:    "foo",
					Type:    "nop",
					Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					Output:  cty.EmptyObjectVal,
					Sources: []string{"abc"},
				},
			},
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"foo": { // Identical
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
						Sources: []string{"abc"},
					},
				},
			},
			wantEvents: nil,
		},
		{
			name: "Create",
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: nil, // Nothing exists
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
						Sources: []string{"abc"},
					},
				},
			},
			wantEvents: []mock.Event{
				{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:    "foo",
					Type:    "nop",
					Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
					Output:  cty.EmptyObjectVal,
					Sources: []string{"abc"},
				}},
			},
		},
		{
			name: "CreateDependency",
			defs: map[string]resource.Definition{"passthrough": &passthrough{}},
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Name:  "foo",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
					},
					"bar": {
						Name: "bar",
						Type: "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: map[string][]graph.Dependency{
					"bar": {{
						Field: cty.GetAttrPath("input"),
						Expression: graph.Expression{
							graph.ExprReference{
								Path: cty.GetAttrPath("foo").GetAttr("output"),
							},
						},
					}},
				},
			},
			wantEvents: []mock.Event{
				{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:   "foo",
					Type:   "passthrough",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("bar")}),
				}},
				{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:   "bar",
					Type:   "passthrough",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("bar")}),
				}},
			},
		},
		{
			name: "NopDependency",
			defs: map[string]resource.Definition{"passthrough": &passthrough{}},
			existing: []resource.Resource{
				{
					Name:   "foo",
					Type:   "passthrough",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello")}),
				},
				{
					Name:   "bar",
					Type:   "passthrough",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello")}),
				},
			},
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Name:  "foo",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					"bar": {
						Name: "bar",
						Type: "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: map[string][]graph.Dependency{
					"bar": {{
						Field: cty.GetAttrPath("input"),
						Expression: graph.Expression{
							graph.ExprReference{
								Path: cty.GetAttrPath("foo").GetAttr("output"),
							},
						},
					}},
				},
			},
			wantEvents: nil,
		},
		{
			name: "UpdateConfig",
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: []resource.Resource{{
				Name:   "foo",
				Type:   "nop",
				Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("before")}),
				Output: cty.EmptyObjectVal,
			}},
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Name:  "foo",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("after")}), // Updated
					},
				},
			},
			wantEvents: []mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:   "foo",
					Type:   "nop",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("after")}), // Updated
					Output: cty.EmptyObjectVal,
				}},
			},
		},
		{
			name: "UpdateSource",
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: []resource.Resource{{
				Name:    "foo",
				Type:    "nop",
				Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
				Output:  cty.EmptyObjectVal,
				Sources: []string{"abc"},
			}},
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}), // Same
						Sources: []string{"xyz"},                                                      // Updated
					},
				},
			},
			wantEvents: []mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:    "foo",
					Type:    "nop",
					Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}), // Same
					Output:  cty.EmptyObjectVal,
					Sources: []string{"xyz"}, // Updated
				}},
			},
		},
		{
			name: "UpdateChild",
			defs: map[string]resource.Definition{"passthrough": &passthrough{}},
			existing: []resource.Resource{{
				Name:   "parent",
				Type:   "passthrough",
				Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
				Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello")}),
			}, {
				Name:   "child",
				Type:   "passthrough",
				Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello world")}),
				Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello world")}),
			}},
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"parent": {
						Name:  "parent",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					"child": {
						Name: "child",
						Type: "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: map[string][]graph.Dependency{
					"child": {{
						Field: cty.GetAttrPath("input"),
						Expression: graph.Expression{
							graph.ExprReference{Path: cty.GetAttrPath("parent").GetAttr("output")},
							graph.ExprLiteral{Value: cty.StringVal(" there")},
						},
					}},
				},
			},
			wantEvents: []mock.Event{
				// Parent not updated
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:   "child",
					Type:   "passthrough",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello there")}),
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello there")}),
				}},
			},
		},
		{
			name: "UpdateParent",
			defs: map[string]resource.Definition{"passthrough": &passthrough{}},
			existing: []resource.Resource{{
				Name:   "parent",
				Type:   "passthrough",
				Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
				Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello")}),
			}, {
				Name:   "child",
				Type:   "passthrough",
				Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello world")}),
				Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello world")}),
			}},
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"parent": {
						Name:  "parent",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hi")}),
					},
					"child": {
						Name: "child",
						Type: "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: map[string][]graph.Dependency{
					"child": {{
						Field: cty.GetAttrPath("input"),
						Expression: graph.Expression{
							graph.ExprReference{Path: cty.GetAttrPath("parent").GetAttr("output")},
							graph.ExprLiteral{Value: cty.StringVal(" world")},
						},
					}},
				},
			},
			wantEvents: []mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:   "parent",
					Type:   "passthrough",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hi")}),
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hi")}),
				}},
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:   "child",
					Type:   "passthrough",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hi world")}),
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hi world")}),
				}},
			},
		},
		{
			name: "CreateDelete", // Always create before Delete
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: []resource.Resource{
				{
					Name:  "foo",
					Type:  "nop",
					Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
				},
			},
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"bar": {
						Name:  "bar",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
				},
			},
			wantEvents: []mock.Event{
				{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name:   "bar",
					Type:   "nop",
					Input:  cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					Output: cty.EmptyObjectVal,
				}},
				{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name: "foo",
				}},
			},
		},
		{
			name: "CreateOptionalNotSet",
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input *struct {
					Val string
				} `func:"input"`
			}{}},
			existing: nil,
			graph: &graph.Graph{
				Resources: map[string]*resource.Resource{
					"bar": {
						Name: "bar",
						Type: "nop",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.NullVal(cty.Object(map[string]cty.Type{
								"val": cty.String,
							})),
						}),
					},
				},
			},
			wantEvents: []mock.Event{
				{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
					Name: "bar",
					Type: "nop",
					Input: cty.ObjectVal(map[string]cty.Value{
						"input": cty.NullVal(cty.Object(map[string]cty.Type{
							"val": cty.String,
						})),
					}),
					Output: cty.EmptyObjectVal,
				}},
			},
		},
		{
			name: "DeleteOrder",
			defs: map[string]resource.Definition{"nop": nop{}},
			existing: []resource.Resource{
				{Name: "foo", Type: "nop"},
				{Name: "bar", Type: "nop", Deps: []string{"foo"}},
				{Name: "baz", Type: "nop", Deps: []string{"foo", "bar"}},
				{Name: "qux", Type: "nop", Deps: []string{"baz"}},
			},
			graph: &graph.Graph{},
			wantEvents: []mock.Event{
				{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "qux"}},
				{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "baz"}},
				{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "bar"}},
				{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "foo"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mock.Storage{}
			store.Seed("ns", "proj", tt.existing)

			rec := &reconciler.Reconciler{
				Resources: store,
				Registry:  resource.RegistryFromDefinitions(tt.defs),
				Logger:    zaptest.NewLogger(t),
			}

			ctx := context.Background()
			err := rec.Reconcile(ctx, tt.name, "ns", "proj", tt.graph)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			assertEvents(t, store, tt.wantEvents)
		})
	}
}

func assertEvents(t *testing.T, store *mock.Storage, want []mock.Event) {
	t.Helper()
	opts := []cmp.Option{
		cmp.Transformer("GoString", func(v cty.Value) string { return v.GoString() }),
	}
	if diff := cmp.Diff(store.Events, want, opts...); diff != "" {
		t.Errorf("Events do not match (-got %d +want %d)\n%s", len(store.Events), len(want), diff)
	}
}

// Test resource definitions

type nop struct{}

func (nop) Create(ctx context.Context, req *resource.CreateRequest) error { return nil }
func (nop) Update(ctx context.Context, req *resource.UpdateRequest) error { return nil }
func (nop) Delete(ctx context.Context, req *resource.DeleteRequest) error { return nil }

type passthrough struct {
	Input  *string `func:"input"`
	Output string  `func:"output"`
}

func (p *passthrough) Create(ctx context.Context, req *resource.CreateRequest) error {
	p.Output = *p.Input
	return nil
}
func (p *passthrough) Update(ctx context.Context, req *resource.UpdateRequest) error {
	p.Output = *p.Input
	return nil
}
func (p *passthrough) Delete(ctx context.Context, req *resource.DeleteRequest) error {
	return nil
}

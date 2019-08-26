package reconciler_test

import (
	"context"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/resource/reconciler"
	"github.com/func/func/storage/teststore"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/zap/zaptest"
)

// Everything in same project
func TestReconciler_Reconcile_events(t *testing.T) {
	tests := []struct {
		name       string
		defs       map[string]resource.Definition
		existing   []*resource.Deployed
		graph      *resource.Graph
		wantEvents teststore.Events
	}{
		{
			name:     "Empty",
			existing: nil,
			graph: &resource.Graph{
				Resources: nil,
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
			},
		},
		{
			name: "Nop",
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: []*resource.Deployed{
				{
					Desired: &resource.Desired{
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
						Sources: []string{"abc"},
					},
					Output: cty.EmptyObjectVal,
				},
			},
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						// Identical
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
						Sources: []string{"abc"},
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
			},
		},
		{
			name: "Create",
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: nil, // Nothing exists
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
						Sources: []string{"abc"},
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
						Sources: []string{"abc"},
					},
					Output: cty.EmptyObjectVal,
				}},
			},
		},
		{
			name: "CreateDependency",
			defs: map[string]resource.Definition{"passthrough": &passthrough{}},
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Name:  "foo",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
					},
					{
						Name: "bar",
						Type: "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: []*resource.Dependency{
					{
						Child: "bar",
						Field: cty.GetAttrPath("input"),
						Expression: resource.Expression{
							resource.ExprReference{
								Path: cty.GetAttrPath("foo").GetAttr("output"),
							},
						},
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:  "foo",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("bar")}),
				}},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:  "bar",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("bar")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("bar")}),
					Deps:   []string{"foo"},
				}},
			},
		},
		{
			name: "NopWithDependency",
			defs: map[string]resource.Definition{"passthrough": &passthrough{}},
			existing: []*resource.Deployed{
				{
					Desired: &resource.Desired{
						Name:  "foo",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello")}),
				},
				{
					Desired: &resource.Desired{
						Name:  "bar",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello")}),
				},
			},
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Name:  "foo",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					{
						Name: "bar",
						Type: "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: []*resource.Dependency{
					{
						Child: "bar",
						Field: cty.GetAttrPath("input"),
						Expression: resource.Expression{
							resource.ExprReference{
								Path: cty.GetAttrPath("foo").GetAttr("output"),
							},
						},
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
			},
		},
		{
			name: "UpdateConfig",
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: []*resource.Deployed{
				{
					Desired: &resource.Desired{
						Name:  "foo",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("before")}),
					},
					Output: cty.EmptyObjectVal,
				},
			},
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Name:  "foo",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("after")}), // Updated
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:  "foo",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("after")}), // Updated
					},
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
			existing: []*resource.Deployed{
				{
					Desired: &resource.Desired{
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
						Sources: []string{"abc"},
					},
					Output: cty.EmptyObjectVal,
				},
			},
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}), // Same
						Sources: []string{"xyz"},                                                      // Updated
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:    "foo",
						Type:    "nop",
						Input:   cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}), // Same
						Sources: []string{"xyz"},                                                      // Updated
					},
					Output: cty.EmptyObjectVal,
				}},
			},
		},
		{
			name: "UpdateChild",
			defs: map[string]resource.Definition{"passthrough": &passthrough{}},
			existing: []*resource.Deployed{
				{
					Desired: &resource.Desired{
						Name:  "parent",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello")}),
				}, {
					Desired: &resource.Desired{
						Name:  "child",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello world")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello world")}),
				},
			},
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Name:  "parent",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					{
						Name: "child",
						Type: "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: []*resource.Dependency{
					{
						Child: "child",
						Field: cty.GetAttrPath("input"),
						Expression: resource.Expression{
							resource.ExprReference{Path: cty.GetAttrPath("parent").GetAttr("output")},
							resource.ExprLiteral{Value: cty.StringVal(" there")},
						},
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				// Parent not updated
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Type:  "passthrough",
						Name:  "child",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello there")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello there")}),
					Deps:   []string{"parent"},
				}},
			},
		},
		{
			name: "UpdateParent",
			defs: map[string]resource.Definition{"passthrough": &passthrough{}},
			existing: []*resource.Deployed{
				{
					Desired: &resource.Desired{
						Name:  "parent",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello")}),
				}, {
					Desired: &resource.Desired{
						Name:  "child",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello world")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hello world")}),
				},
			},
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Name:  "parent",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hi")}),
					},
					{
						Name: "child",
						Type: "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: []*resource.Dependency{
					{
						Child: "child",
						Field: cty.GetAttrPath("input"),
						Expression: resource.Expression{
							resource.ExprReference{Path: cty.GetAttrPath("parent").GetAttr("output")},
							resource.ExprLiteral{Value: cty.StringVal(" world")},
						},
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:  "parent",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hi")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hi")}),
				}},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:  "child",
						Type:  "passthrough",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hi world")}),
					},
					Output: cty.ObjectVal(map[string]cty.Value{"output": cty.StringVal("hi world")}),
					Deps:   []string{"parent"},
				}},
			},
		},
		{
			name: "CreateDelete", // Always create before Delete
			defs: map[string]resource.Definition{"nop": struct {
				nop
				Input string `func:"input"`
			}{}},
			existing: []*resource.Deployed{
				{
					Desired: &resource.Desired{
						Name:  "foo",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					Output: cty.EmptyObjectVal,
				},
			},
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Name:  "bar",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
				},
			},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:  "bar",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					Output: cty.EmptyObjectVal,
				}},
				{Method: "DeleteResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name:  "foo",
						Type:  "nop",
						Input: cty.ObjectVal(map[string]cty.Value{"input": cty.StringVal("hello")}),
					},
					Output: cty.EmptyObjectVal,
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
			graph: &resource.Graph{
				Resources: []*resource.Desired{
					{
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
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				{Method: "PutResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{
						Name: "bar",
						Type: "nop",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.NullVal(cty.Object(map[string]cty.Type{
								"val": cty.String,
							})),
						}),
					},
					Output: cty.EmptyObjectVal,
				}},
			},
		},
		{
			name: "DeleteOrder",
			defs: map[string]resource.Definition{"nop": nop{}},
			existing: []*resource.Deployed{
				{Desired: &resource.Desired{Name: "foo", Type: "nop"}},
				{Desired: &resource.Desired{Name: "bar", Type: "nop"}, Deps: []string{"foo"}},
				{Desired: &resource.Desired{Name: "baz", Type: "nop"}, Deps: []string{"foo", "bar"}},
				{Desired: &resource.Desired{Name: "qux", Type: "nop"}, Deps: []string{"baz"}},
			},
			graph: &resource.Graph{},
			wantEvents: teststore.Events{
				{Method: "ListResources", Project: "proj"},
				{Method: "DeleteResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{Type: "nop", Name: "qux"}, Deps: []string{"baz"},
				}},
				{Method: "DeleteResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{Type: "nop", Name: "baz"}, Deps: []string{"foo", "bar"},
				}},
				{Method: "DeleteResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{Type: "nop", Name: "bar"}, Deps: []string{"foo"},
				}},
				{Method: "DeleteResource", Project: "proj", Data: &resource.Deployed{
					Desired: &resource.Desired{Type: "nop", Name: "foo"},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &teststore.Store{}
			store.SeedResources("proj", tt.existing)
			rec := &teststore.Recorder{Store: store}

			reco := &reconciler.Reconciler{
				Resources: rec,
				Registry:  resource.RegistryFromDefinitions(tt.defs),
				Logger:    zaptest.NewLogger(t),
			}

			ctx := context.Background()
			err := reco.Reconcile(ctx, tt.name, "proj", tt.graph)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Value) bool {
					return a.Equals(b).True()
				}),
			}
			if diff := cmp.Diff(rec.Events, tt.wantEvents, opts...); diff != "" {
				t.Errorf("Events (-got +want)\n%s", diff)
			}
		})
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

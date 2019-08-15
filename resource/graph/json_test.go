package graph

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"
)

func TestGraph_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		types map[string]reflect.Type
		graph *Graph
	}{
		{
			"Resource",
			map[string]reflect.Type{
				"footype": reflect.TypeOf(struct {
					Input int `func:"input"`
				}{}),
				"bartype": reflect.TypeOf(struct {
					Nested struct {
						Value string
					} `func:"input"`
				}{}),
			},
			&Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Type:    "footype",
						Name:    "foo",
						Sources: []string{"abc123"},
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.NumberIntVal(123),
						}),
					},
					"bar": {
						Type: "bartype",
						Name: "bar",
						Input: cty.ObjectVal(map[string]cty.Value{
							"nested": cty.ObjectVal(map[string]cty.Value{
								"value": cty.StringVal("hello"),
							}),
						}),
						Deps: []string{"foo", "baz"},
					},
				},
			},
		},
		{
			"Dependency",
			map[string]reflect.Type{
				"footype": reflect.TypeOf(struct {
				}{}),
				"bartype": reflect.TypeOf(struct {
				}{}),
			},
			&Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Type:    "footype",
						Name:    "foo",
						Sources: []string{"abc123"},
						Input:   cty.EmptyObjectVal,
					},
					"bar": {
						Type:  "bartype",
						Name:  "bar",
						Input: cty.EmptyObjectVal,
					},
				},
				Dependencies: map[string][]Dependency{
					"bar": {
						{
							Field: cty.GetAttrPath("ref"),
							Expression: Expression{
								ExprLiteral{Value: cty.StringVal("^^^")},
								ExprReference{Path: cty.GetAttrPath("foo").GetAttr("output")},
								ExprLiteral{Value: cty.StringVal("vvv")},
							},
						},
					},
				},
			},
		},
		{
			"Slice",
			map[string]reflect.Type{
				"footype": reflect.TypeOf(struct {
					Input struct {
						Nested []struct {
							Num int
						}
					} `func:"input"`
				}{}),
			},
			&Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Type:    "footype",
						Name:    "foo",
						Sources: []string{"abc123"},
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.ObjectVal(map[string]cty.Value{
								"nested": cty.ListVal([]cty.Value{
									cty.ObjectVal(map[string]cty.Value{
										"num": cty.NumberIntVal(123),
									}),
									cty.ObjectVal(map[string]cty.Value{
										"num": cty.NumberIntVal(456),
									}),
									cty.ObjectVal(map[string]cty.Value{
										"num": cty.NumberIntVal(789),
									}),
								}),
							}),
						}),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j, err := json.Marshal(tt.graph)
			if err != nil {
				t.Fatalf("MarshalJSON() err = %v", err)
			}

			// Check that resource bytes were not double encoded to base64
			if !bytes.Contains(j, []byte("footype")) {
				t.Errorf("Resource is not json-encoded")
			}

			var buf bytes.Buffer
			if err = json.Indent(&buf, j, "", "\t"); err != nil {
				t.Fatalf("Indent() err = %v", err)
			}
			t.Logf("Encoded graph json %d bytes:\n%s", len(j), buf.String())

			got := New()
			dec := JSONDecoder{
				Target:   got,
				Registry: &resource.Registry{Types: tt.types},
			}

			if err := dec.UnmarshalJSON(j); err != nil {
				t.Fatalf("Unmarshal() err = %+v", err)
			}

			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Value) bool { return a.Equals(b).True() }),
				cmp.Comparer(func(a, b cty.Path) bool { return a.Equals(b) }),
				cmpopts.EquateEmpty(),
			}
			if diff := cmp.Diff(got, tt.graph, opts...); diff != "" {
				t.Errorf("Roundtrip (-got +want)\n%s", diff)
			}
		})
	}
}

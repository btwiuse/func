package graph

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestGraph_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		file  string
		check func(t *testing.T, g *Graph)
	}{
		{"testdata/graph.json", func(t *testing.T, g *Graph) {
			if len(g.Resources) != 20 {
				t.Errorf("Resources = %d, want = %d", len(g.Resources), 20)
			}
			if len(g.Dependencies) != 14 {
				t.Errorf("Dependencies = %d, want = %d", len(g.Dependencies), 14)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := ioutil.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Load file: %v", err)
			}
			g := New()
			err = json.Unmarshal(data, g)
			if err != nil {
				t.Fatalf("UnmarshalJSON() err = %v", err)
			}
			tt.check(t, g)
		})
	}
}

func TestJSON_roundtrip(t *testing.T) {
	tests := []struct {
		name  string
		graph *Graph
	}{
		{
			"Resource",
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
			&Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Type:    "footype",
						Name:    "foo",
						Sources: []string{"abc123"},
					},
					"bar": {
						Type: "bartype",
						Name: "bar",
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
			&Graph{
				Resources: map[string]*resource.Resource{
					"foo": {
						Type:    "footype",
						Name:    "foo",
						Sources: []string{"abc123"},
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.ObjectVal(map[string]cty.Value{
								"nested": cty.TupleVal([]cty.Value{
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
				t.Fatalf("Marshal() err = %v", err)
			}

			var buf bytes.Buffer
			_ = json.Indent(&buf, j, "", "\t")
			t.Logf("%d bytes: %s\n%s", len(j), string(j), buf.String())

			got := &Graph{}
			if err := json.Unmarshal(j, got); err != nil {
				t.Fatalf("Unmarshal() err = %+v", err)
			}

			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Value) bool {
					return a.Equals(b).True()
				}),
				cmp.Transformer("Name", func(v cty.GetAttrStep) string {
					return v.Name
				}),
				cmp.Transformer("GoString", func(v cty.IndexStep) string {
					return v.GoString()
				}),
			}
			if diff := cmp.Diff(got, tt.graph, opts...); diff != "" {
				t.Errorf("Roundtrip (-got, +want)\n%s", diff)
			}
		})
	}
}

package graph

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
	resjson "github.com/func/func/resource/encoding/json"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"
)

func TestGraph_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		file  string
		codec func() ResourceCodec
		check func(t *testing.T, g *Graph)
	}{
		{
			"testdata/graph.json",
			func() ResourceCodec {
				reg := &resource.Registry{}
				aws.Register(reg)
				return &resjson.Encoder{Registry: reg}
			},
			func(t *testing.T, g *Graph) {
				if len(g.Resources) != 20 {
					t.Errorf("Resources = %d, want = %d", len(g.Resources), 20)
				}
				if len(g.Dependencies) != 14 {
					t.Errorf("Dependencies = %d, want = %d", len(g.Dependencies), 14)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := ioutil.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("Load file: %v", err)
			}

			enc := JSONEncoder{Codec: tt.codec()}

			g := New()
			if err = enc.Unmarshal(data, g); err != nil {
				t.Fatalf("UnmarshalJSON() err = %v", err)
			}
			tt.check(t, g)
		})
	}
}

func TestJSON_roundtrip(t *testing.T) {
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
			enc := JSONEncoder{
				Codec: &resjson.Encoder{
					Registry: &resource.Registry{
						Types: tt.types,
					},
				},
			}

			j, err := enc.Marshal(tt.graph)
			if err != nil {
				t.Fatalf("Marshal() err = %v", err)
			}

			// Check that resource bytes were not double encoded to base64
			if !bytes.Contains(j, []byte("footype")) {
				t.Errorf("Resource is not json-encoded")
			}

			var buf bytes.Buffer
			_ = json.Indent(&buf, j, "", "\t")
			t.Logf("Encoded graph json %d bytes:\n%s", len(j), buf.String())

			var got Graph
			if err := enc.Unmarshal(j, &got); err != nil {
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
				cmpopts.EquateEmpty(),
			}
			if diff := cmp.Diff(&got, tt.graph, opts...); diff != "" {
				t.Errorf("Roundtrip (-got, +want)\n%s", diff)
			}
		})
	}
}

func TestMarshalPanic(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("Did not panic on json.Marshal")
		}
	}()
	g := New()
	_, _ = json.Marshal(g)
}

func TestUnmarshalPanic(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("Did not panic on json.Unarshal")
		}
	}()
	g := New()
	_ = json.Unmarshal([]byte("{}"), &g)
}

package hcldecoder_test

import (
	"bytes"
	"flag"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/func/func/config"
	"github.com/func/func/resource"
	"github.com/func/func/resource/hcldecoder"
	"github.com/go-stack/stack"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestDecodeBody(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		types       map[string]reflect.Type
		want        *resource.Graph
		wantSources []*config.SourceInfo
	}{
		{
			name: "StaticInput",
			config: `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
			`,
			types: map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "a",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"),
						}),
					},
				},
			},
		},
		{
			name: "ConvertInputs",
			config: `
				resource "foo" {
					type    = "a"
					string  = 123 
					strings = [123, 4.5, -6.789]
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					String  string   `func:"input"`
					Strings []string `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "a",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							// Values converted to target types.
							"string": cty.StringVal("123"),
							"strings": cty.ListVal([]cty.Value{
								cty.StringVal("123"),
								cty.StringVal("4.5"),
								cty.StringVal("-6.789"),
							}),
						}),
					},
				},
			},
		},
		{
			name: "Source",
			config: `
				resource "foo" {
					type   = "a"
					source = "ff:abc:def"
				}
			`,
			types: map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type:    "a",
						Name:    "foo",
						Sources: []string{"def"},
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.NullVal(cty.String),
						}),
					},
				},
			},
			wantSources: []*config.SourceInfo{
				{Key: "def", MD5: "abc", Len: 0xFF},
			},
		},
		{
			name: "DependencyToInput",
			config: `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = foo.input
				}
			`,
			types: map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "a",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"),
						}),
					},
					{
						Type: "a",
						Name: "bar",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"), // Can be statically resolved.
						}),
					},
				},
			},
		},
		{
			name: "DependencyToTransitiveInput",
			config: `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = foo.input
				}
				resource "baz" {
					type  = "a"
					input = bar.input
				}
			`,
			types: map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "a",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"),
						}),
					},
					{
						Type: "a",
						Name: "bar",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"), // Can be statically resolved.
						}),
					},
					{
						Type: "a",
						Name: "baz",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"), // Can be transitively resolved.
						}),
					},
				},
			},
		},
		{
			name: "DependencyToOutput",
			config: `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = foo.output
				}
			`,
			types: map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "a",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"),
						}),
					},
					{
						Type: "a",
						Name: "bar",
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
		},
		{
			name: "DependencyExpression",
			config: `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = "world"
				}
				resource "baz" {
					type  = "a"
					input = "Oh, ${foo.input} ${bar.input} ${foo.output}!"
				}
			`,
			types: map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "a",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"),
						}),
					},
					{
						Type: "a",
						Name: "bar",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("world"),
						}),
					},
					{
						Type: "a",
						Name: "baz",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
				},
				Dependencies: []*resource.Dependency{
					{
						Child: "baz",
						Field: cty.GetAttrPath("input"),
						Expression: resource.Expression{
							resource.ExprLiteral{Value: cty.StringVal("Oh, hello world ")}, // merged
							resource.ExprReference{Path: cty.GetAttrPath("foo").GetAttr("output")},
							resource.ExprLiteral{Value: cty.StringVal("!")},
						},
					},
				},
			},
		},
		{
			name: "PointerInput",
			config: `
				resource "foo" {
					type  = "ptr"
					input = "hello"
				}
			`,
			types: map[string]reflect.Type{
				"ptr": reflect.TypeOf(struct {
					Input *string `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "ptr",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.StringVal("hello"),
						}),
					},
				},
			},
		},
		{
			name: "Map",
			config: `
				resource "foo" {
					type = "mapdef"
					map = {
						foo = "bar"
					}
				}
			`,
			types: map[string]reflect.Type{
				"mapdef": reflect.TypeOf(struct {
					Map map[string]string `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "mapdef",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"map": cty.MapVal(map[string]cty.Value{
								"foo": cty.StringVal("bar"),
							}),
						}),
					},
				},
			},
		},
		{
			name: "Slice",
			config: `
				resource "foo" {
					type    = "slicedef"
					strings = ["hello", "world"]
				}
			`,
			types: map[string]reflect.Type{
				"slicedef": reflect.TypeOf(struct {
					Strings []string `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "slicedef",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"strings": cty.ListVal([]cty.Value{
								cty.StringVal("hello"),
								cty.StringVal("world"),
							}),
						}),
					},
				},
			},
		},
		{
			name: "StructBlock",
			config: `
				resource "foo" {
					type = "structdef"
					deep {
						nested {
							val = 123
						}
					}
				}
			`,
			types: map[string]reflect.Type{
				"structdef": reflect.TypeOf(struct {
					Deep struct {
						Nested struct {
							Val int // no input tag required here
						} // or here
					} `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "structdef",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"deep": cty.ObjectVal(map[string]cty.Value{
								"nested": cty.ObjectVal(map[string]cty.Value{
									"val": cty.NumberIntVal(123),
								}),
							}),
						}),
					},
				},
			},
		},
		{
			name: "MissingOptionalBlock",
			config: `
				resource "foo" {
					type = "structdef"
				}
			`,
			types: map[string]reflect.Type{
				"structdef": reflect.TypeOf(struct {
					Sub *struct {
						Val string
					} `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "structdef",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"sub": cty.NullVal(cty.Object(map[string]cty.Type{
								"val": cty.String,
							})),
						}),
					},
				},
			},
		},
		{
			name: "BlockSliceEmpty",
			config: `
				resource "foo" {
					type = "bar"
				}
			`,
			types: map[string]reflect.Type{
				"bar": reflect.TypeOf(struct {
					Sub []struct {
						Val string
					} `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "bar",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"sub": cty.ListValEmpty(cty.Object(map[string]cty.Type{
								"val": cty.String,
							})),
						}),
					},
				},
			},
		},
		{
			name: "StructPointer",
			config: `
				resource "foo" {
					type = "pie"
					value {
						val = 31415
					}
				}
			`,
			types: map[string]reflect.Type{
				"pie": reflect.TypeOf(struct {
					Value *struct {
						Val uint32
					} `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "pie",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"value": cty.ObjectVal(map[string]cty.Value{
								"val": cty.NumberUIntVal(31415),
							}),
						}),
					},
				},
			},
		},
		{
			name: "MultipleBlocks",
			config: `
				resource "foo" {
					type = "multi"
					multi {
						name = "alice"
						age  = 20
					}
					multi {
						name = "bob"
						age  = 30
					}
				}
			`,
			types: map[string]reflect.Type{
				"multi": reflect.TypeOf(struct {
					Multi []struct {
						Name string
						Age  int64
					} `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "multi",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"multi": cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									"name": cty.StringVal("alice"),
									"age":  cty.NumberIntVal(20),
								}),
								cty.ObjectVal(map[string]cty.Value{
									"name": cty.StringVal("bob"),
									"age":  cty.NumberIntVal(30),
								}),
							}),
						}),
					},
				},
			},
		},
		{
			name: "MultipleBlockPtrs",
			config: `
				resource "foo" {
					type = "multi"
					multi {
						name = "carol"
					}
					multi {
						name = "dan"
					}
				}
			`,
			types: map[string]reflect.Type{
				"multi": reflect.TypeOf(struct {
					Multi []*struct {
						Name string
					} `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "multi",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"multi": cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									"name": cty.StringVal("carol"),
								}),
								cty.ObjectVal(map[string]cty.Value{
									"name": cty.StringVal("dan"),
								}),
							}),
						}),
					},
				},
			},
		},
		{
			name: "OutputMapStruct",
			config: `
				resource "foo" {
					type = "complex"
				}
				resource "bar" {
					type  = "simple"
					input = foo.nested["foo"].output
				}
			`,
			types: map[string]reflect.Type{
				"complex": reflect.TypeOf(struct {
					Nested map[string]simpleDef `func:"output"`
				}{}),
				"simple": reflect.TypeOf(simpleDef{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type:  "complex",
						Name:  "foo",
						Input: cty.EmptyObjectVal,
					},
					{
						Type: "simple",
						Name: "bar",
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
								Path: cty.GetAttrPath("foo").GetAttr("nested").Index(cty.StringVal("foo")).GetAttr("output"),
							},
						},
					},
				},
			},
		},
		{
			name: "OutputComplex",
			config: `
				resource "foo" {
					type = "complex"
				}
				resource "bar" {
					type  = "simple"
					input = foo.nested["foo"][0]["bar"].output
				}
			`,
			types: map[string]reflect.Type{
				"complex": reflect.TypeOf(struct {
					Nested map[string][]map[string]simpleDef `func:"output"`
				}{}),
				"simple": reflect.TypeOf(simpleDef{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type:  "complex",
						Name:  "foo",
						Input: cty.EmptyObjectVal,
					},
					{
						Type: "simple",
						Name: "bar",
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
								Path: cty.
									GetAttrPath("foo").
									GetAttr("nested").
									Index(cty.StringVal("foo")).
									Index(cty.NumberIntVal(0)).
									Index(cty.StringVal("bar")).
									GetAttr("output"),
							},
						},
					},
				},
			},
		},
		{
			name: "OutputSliceStruct",
			config: `
				resource "foo" {
					type = "complex"
				}
				resource "bar" {
					type  = "simple"
					input = foo.nested[0].output
				}
			`,
			types: map[string]reflect.Type{
				"complex": reflect.TypeOf(struct {
					Nested []simpleDef `func:"output"`
				}{}),
				"simple": reflect.TypeOf(simpleDef{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type:  "complex",
						Name:  "foo",
						Input: cty.EmptyObjectVal,
					},
					{
						Type: "simple",
						Name: "bar",
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
								Path: cty.GetAttrPath("foo").GetAttr("nested").Index(cty.NumberIntVal(0)).GetAttr("output"),
							},
						},
					},
				},
			},
		},
		{
			name: "NestedDependencies",
			config: `
				resource "foo" {
					type = "a"
					in   = "hello"
				}
				resource "bar" {
					type = "b"
					input {
						string = foo.out
						int    = 123
					}
				}
				resource "baz" {
					type    = "c"
					num     = bar.output.number
					strings = bar.output.names
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					In  string `func:"input"`
					Out string `func:"output"`
				}{}),
				"b": reflect.TypeOf(struct {
					Input struct {
						String string
						Int    int32
					} `func:"input"`
					Output struct {
						Number uint64
						Names  []string
					} `func:"output"`
				}{}),
				"c": reflect.TypeOf(struct {
					Num     float64  `func:"input"`
					Strings []string `func:"input"`
				}{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type: "a",
						Name: "foo",
						Input: cty.ObjectVal(map[string]cty.Value{
							"in": cty.StringVal("hello"),
						}),
					},
					{
						Type: "b",
						Name: "bar",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.ObjectVal(map[string]cty.Value{
								"string": cty.UnknownVal(cty.String),
								"int":    cty.NumberIntVal(123),
							}),
						}),
					},
					{
						Type: "c",
						Name: "baz",
						Input: cty.ObjectVal(map[string]cty.Value{
							"num":     cty.UnknownVal(cty.Number),
							"strings": cty.UnknownVal(cty.List(cty.String)),
						}),
					},
				},
				Dependencies: []*resource.Dependency{
					{
						Child: "bar",
						Field: cty.GetAttrPath("input").GetAttr("string"),
						Expression: resource.Expression{
							resource.ExprReference{Path: cty.GetAttrPath("foo").GetAttr("out")},
						},
					},
					{
						Child: "baz",
						Field: cty.GetAttrPath("num"),
						Expression: resource.Expression{
							resource.ExprReference{Path: cty.GetAttrPath("bar").GetAttr("output").GetAttr("number")},
						},
					},
					{
						Child: "baz",
						Field: cty.GetAttrPath("strings"),
						Expression: resource.Expression{
							resource.ExprReference{Path: cty.GetAttrPath("bar").GetAttr("output").GetAttr("names")},
						},
					},
				},
			},
		},
		{
			name: "NestedSliceIndex",
			config: `
				resource "foo" {
					type = "output"
				}
				resource "bar" {
					type = "simple"
					input = foo.out[0]
				}
				resource "baz" {
					type = "simple"
					input = foo.out[1]
				}
			`,
			types: map[string]reflect.Type{
				"output": reflect.TypeOf(struct {
					Out []string `func:"output"`
				}{}),
				"simple": reflect.TypeOf(simpleDef{}),
			},
			want: &resource.Graph{
				Resources: []*resource.Desired{
					{
						Type:  "output",
						Name:  "foo",
						Input: cty.EmptyObjectVal,
					},
					{
						Type: "simple",
						Name: "bar",
						Input: cty.ObjectVal(map[string]cty.Value{
							"input": cty.UnknownVal(cty.String),
						}),
					},
					{
						Type: "simple",
						Name: "baz",
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
							resource.ExprReference{Path: cty.GetAttrPath("foo").GetAttr("out").Index(cty.NumberIntVal(0))},
						},
					},
					{
						Child: "baz",
						Field: cty.GetAttrPath("input"),
						Expression: resource.Expression{
							resource.ExprReference{Path: cty.GetAttrPath("foo").GetAttr("out").Index(cty.NumberIntVal(1))},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer checkPanic(t)
			g := &resource.Graph{}

			parser := &testParser{}
			body := parser.Parse(t, tt.config)

			dec := &hcldecoder.Decoder{
				Resources: &resource.Registry{Types: tt.types},
				Validator: ValidateFunc(func(interface{}, string) error { return nil }),
			}
			srcs, diags := dec.DecodeBody(body, g)
			parser.CheckDiags(t, diags)

			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Path) bool { return a.Equals(b) }),
				cmp.Comparer(func(a, b cty.Value) bool {
					if a.IsWhollyKnown() && b.IsWhollyKnown() {
						return a.Equals(b).True()
					}
					return a.GoString() == b.GoString()
				}),
				// Order of resource or dependencies do not matter
				cmpopts.SortSlices(func(a, b *resource.Desired) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b *resource.Dependency) bool {
					astr := fmt.Sprintf("%+v", a)
					bstr := fmt.Sprintf("%+v", b)
					return astr < bstr
				}),
			}
			if diff := cmp.Diff(g, tt.want, opts...); diff != "" {
				t.Errorf("Graph does not match (-got +want)\n%s", diff)
			}
			if diff := cmp.Diff(srcs, tt.wantSources, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Sources do not match (-got +want)\n%s", diff)
			}
		})
	}
}

func TestDecodeBody_Diagnostics(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		types     map[string]reflect.Type
		validator hcldecoder.Validator
		diags     hcl.Diagnostics // filename is always file.hcl
	}{
		{
			name: "ExtraLabel",
			config: `
				resource "foo" "bar" {}
			`,
			types:     map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Extraneous label for resource",
				Detail:   "Only 1 labels (name) are expected for resource blocks.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 16, Byte: 15},
					End:      hcl.Pos{Line: 1, Column: 21, Byte: 20},
				},
				Context: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 23, Byte: 22},
				},
			}},
		},
		{
			name: "MissingType",
			config: `
				resource "foo" {
					input = "a"
				}
			`,
			types:     map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   `The argument "type" is required, but no definition was found.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 16, Byte: 15},
					End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
				},
			}},
		},
		{
			name: "UnsupportedArgument",
			config: `
				resource "foo" {
					type         = "a"
					input        = "hello"
					notsupported = 123
				}
			`,
			types:     map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported argument",
				Detail:   `An argument named "notsupported" is not expected here.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 4, Column: 2, Byte: 62},
					End:      hcl.Pos{Line: 4, Column: 14, Byte: 74},
				},
			}},
		},
		{
			name: "RefWithUnsupportedArgument",
			config: `
				resource "foo" {
					type         = "a"
					notsupported = 123
				}
				resource "bar" {
					type         = "a"
					input        = foo.input
				}
			`,
			types: map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported argument",
				Detail:   `An argument named "notsupported" is not expected here.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 2, Byte: 38},
					End:      hcl.Pos{Line: 3, Column: 14, Byte: 50},
				},
			}},
		},
		{
			name: "InvalidSource",
			config: `
				resource "foo" {
					type   = "a"
					input  = "hello"
					source = "xxx"
				}
			`,
			types:     map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Could not decode source information",
				Detail:   "Error: string must contain 3 parts separated by ':'. This is always a bug.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 4, Column: 11, Byte: 59},
					End:      hcl.Pos{Line: 4, Column: 16, Byte: 64},
				},
			}},
		},
		{
			name: "NonexistingDependency",
			config: `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = nonexisting.output
				}
			`,
			types:     map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Referenced value not found",
					Detail:   "An object named \"nonexisting\" is not defined.",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 7, Column: 2, Byte: 80},
						End:      hcl.Pos{Line: 7, Column: 28, Byte: 106},
					},
				},
			},
		},
		{
			name: "NonexistingDependencySuggest",
			config: `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = zoo.output
				}
			`,
			types:     map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Referenced value not found",
					Detail:   "An object named \"zoo\" is not defined. Did you mean \"foo\"?",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 7, Column: 2, Byte: 80},
						End:      hcl.Pos{Line: 7, Column: 20, Byte: 98},
					},
				},
			},
		},
		{
			name: "NonexistingDependencyField",
			config: `
				resource "foo" {
					type  = "first_type"
					input = "hello"
				}
				resource "bar" {
					type  = "second_type"
					input = foo.nonexisting
				}
			`,
			types: map[string]reflect.Type{
				"first_type":  reflect.TypeOf(simpleDef{}),
				"second_type": reflect.TypeOf(simpleDef{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "No such field",
					Detail:   "Object foo (first_type) does not have a field \"nonexisting\".",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 7, Column: 2, Byte: 99},
						End:      hcl.Pos{Line: 7, Column: 25, Byte: 122},
					},
				},
			},
		},
		{
			name: "NonexistingDependencyFieldSuggest",
			config: `
				resource "foo" {
					type  = "first_type"
					input = "hello"
				}
				resource "bar" {
					type  = "second_type"
					input = foo.putput # typo
				}
			`,
			types: map[string]reflect.Type{
				"first_type":  reflect.TypeOf(simpleDef{}),
				"second_type": reflect.TypeOf(simpleDef{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "No such field",
					Detail:   "Object foo (first_type) does not have a field \"putput\". Did you mean \"output\"?",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 7, Column: 2, Byte: 99},
						End:      hcl.Pos{Line: 7, Column: 20, Byte: 117},
					},
				},
			},
		},
		{
			name: "InvalidReference",
			config: `
				resource "foo" {
					type  = "test_type"
					input = "hello"
				}
				resource "bar" {
					type  = "test_type"
					input = foo.output.value # nested value in string
				}
			`,
			types:     map[string]reflect.Type{"test_type": reflect.TypeOf(simpleDef{})},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference",
					Detail:   "Object foo (test_type): cannot access nested type \"value\" in string.",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 7, Column: 2, Byte: 96},
						End:      hcl.Pos{Line: 7, Column: 26, Byte: 120},
					},
				},
			},
		},
		{
			name: "StructAssignInvalid",
			config: `
				resource "foo" {
					type = "a"
					nested {
						sub {
							value = ["hello", "world"]
						}
					}
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					Nested struct {
						Sub struct {
							Value []int `func:"input"`
						} `func:"input"`
					} `func:"input"`
				}{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "The value must be a list of number, conversion from tuple is not possible.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 5, Column: 4, Byte: 50},
					End:      hcl.Pos{Line: 5, Column: 30, Byte: 76},
				},
			}},
		},
		{
			name: "MultipleBlocksNotAllowed",
			config: `
				resource "foo" {
					type = "a"
					nested {
						value = "hello"
					}
					nested {
						value = "world"
					}
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					resource.Definition
					Nested struct {
						Value string `func:"input"`
					} `func:"input"`
				}{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Duplicate block",
				Detail:   "Only one \"nested\" block is allowed. Another was defined on line 3.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 6, Column: 2, Byte: 61},
					End:      hcl.Pos{Line: 6, Column: 8, Byte: 67},
				},
				Context: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 2, Byte: 30},
					End:      hcl.Pos{Line: 6, Column: 8, Byte: 67},
				},
			}},
		},
		{
			name: "MissingBlock",
			config: `
				resource "foo" {
					type  = "a"
					# required block not set
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					RequiredChild struct{} `func:"input"`
				}{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required block",
				Detail:   "A required_child block is required.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 16, Byte: 15},
					End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
				},
			}},
		},
		{
			name: "MissingNestedBlock",
			config: `
				resource "foo" {
					type = "a"
					nested {
					}
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					resource.Definition
					Nested struct {
						Sub struct {
							Val string
						}
					} `func:"input"`
				}{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required block",
				Detail:   "A sub block is required.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 9, Byte: 37},
					End:      hcl.Pos{Line: 3, Column: 9, Byte: 37},
				},
			}},
		},
		{
			name: "NoResourceName",
			config: `
				resource "" {
					type = "test"
				}
			`,
			types: map[string]reflect.Type{"test": reflect.TypeOf(simpleDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource name not set",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
				Context: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
			}},
		},
		{
			name: "DuplicateResource",
			config: `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "foo" {
					type  = "a"
					input = "world"
				}
			`,
			types:     map[string]reflect.Type{"a": reflect.TypeOf(simpleDef{})},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Duplicate resource",
				Detail:   `Another resource "foo" was defined in file.hcl on line 1.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 5, Column: 1, Byte: 49},
					End:      hcl.Pos{Line: 5, Column: 15, Byte: 63},
				},
			}},
		},
		{
			name: "InvalidType",
			config: `
				resource "foo" {
					type = "a"
					int = "this cannot be an int"
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					Int int `func:"input"`
				}{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "The value must be a number, conversion from string is not possible.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 2, Byte: 30},
					End:      hcl.Pos{Line: 3, Column: 31, Byte: 59},
				},
			}},
		},
		{
			name: "ConvertType",
			config: `
				resource "foo" {
					type = "a"
					string = 123
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					String string `func:"input"`
				}{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagWarning,
				Summary:  "Value is converted from number to string",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 2, Byte: 30},
					End:      hcl.Pos{Line: 3, Column: 14, Byte: 42},
				},
			}},
		},
		{
			name: "ResourceNotFound",
			config: `
				resource "bar" {
					type = "not_found"
				}
			`,
			types:     map[string]reflect.Type{},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource not supported",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 9, Byte: 25},
					End:      hcl.Pos{Line: 2, Column: 20, Byte: 36},
				},
			}},
		},
		{
			name: "TypeExpression",
			config: `
				resource "foo" {
					type = foo.bar
				}
			`,
			types:     map[string]reflect.Type{},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Variables not allowed",
				Detail:   "Variables may not be used here.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 9, Byte: 25},
					End:      hcl.Pos{Line: 2, Column: 12, Byte: 28},
				},
				Expression: &hclsyntax.ScopeTraversalExpr{
					SrcRange: hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 2, Column: 9, Byte: 25},
						End:      hcl.Pos{Line: 2, Column: 16, Byte: 32},
					},
					Traversal: hcl.Traversal{
						hcl.TraverseRoot{
							Name: "foo",
							SrcRange: hcl.Range{
								Filename: "file.hcl",
								Start:    hcl.Pos{Line: 2, Column: 9, Byte: 25},
								End:      hcl.Pos{Line: 2, Column: 12, Byte: 28},
							},
						},
						hcl.TraverseAttr{
							Name: "bar",
							SrcRange: hcl.Range{
								Filename: "file.hcl",
								Start:    hcl.Pos{Line: 2, Column: 12, Byte: 28},
								End:      hcl.Pos{Line: 2, Column: 16, Byte: 32},
							},
						},
					},
				},
			}},
		},
		{
			name: "SuggestResource",
			config: `
				resource "bar" {
					type = "sample"
				}
			`,
			types: map[string]reflect.Type{
				"simple": reflect.TypeOf(simpleDef{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource not supported",
				Detail:   "Did you mean \"simple\"?",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 9, Byte: 25},
					End:      hcl.Pos{Line: 2, Column: 17, Byte: 33},
				},
			}},
		},
		{
			name: "MissingRequiredArg",
			config: `
				resource "a" {
					type  = "a"
					# input not set
				}
			`,
			types: map[string]reflect.Type{
				"a": reflect.TypeOf(struct {
					Input string `func:"input"`
				}{}),
			},
			validator: ValidateFunc(func(interface{}, string) error { return nil }),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   `The argument "input" is required, but no definition was found.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 14, Byte: 13},
					End:      hcl.Pos{Line: 1, Column: 14, Byte: 13},
				},
			}},
		},
		{
			name: "ValidationError",
			config: `
				resource "a" {
					type  = "validation"
					input = "foo"
				}
			`,
			types: map[string]reflect.Type{
				"validation": reflect.TypeOf(struct {
					Input string `func:"input" validate:"bar"`
				}{}),
			},
			validator: ValidateFunc(func(v interface{}, param string) error {
				if fmt.Sprintf("%v", v) != param {
					return fmt.Errorf(`value must be %q`, param)
				}
				return nil
			}),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Validation error",
				Detail:   `Value must be "bar"`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 10, Byte: 46},
					End:      hcl.Pos{Line: 3, Column: 15, Byte: 51},
				},
			}},
		},
		{
			name: "ValidationErrorRef",
			config: `
				resource "a" {
					type  = "simple"
					input = "foo"
				}
				resource "b" {
					type  = "validation"
					input = a.input
				}
			`,
			types: map[string]reflect.Type{
				"simple": reflect.TypeOf(simpleDef{}),
				"validation": reflect.TypeOf(struct {
					Input string `func:"input" validate:"bar"`
				}{}),
			},
			validator: ValidateFunc(func(v interface{}, param string) error {
				if fmt.Sprintf("%v", v) != param {
					return fmt.Errorf(`value must be %q`, param)
				}
				return nil
			}),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Validation error",
				Detail:   `Value must be "bar"`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 7, Column: 2, Byte: 88},
					End:      hcl.Pos{Line: 7, Column: 17, Byte: 103},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer checkPanic(t)
			g := &resource.Graph{}

			parser := &testParser{
				filename: "file.hcl",
			}
			body := parser.Parse(t, tt.config)

			dec := &hcldecoder.Decoder{
				Resources: &resource.Registry{Types: tt.types},
				Validator: tt.validator,
			}
			_, diags := dec.DecodeBody(body, g)

			opts := []cmp.Option{
				cmpopts.SortSlices(func(a, b hcl.Diagnostic) bool { return a.Error() < b.Error() }),
				cmpopts.IgnoreUnexported(hcl.TraverseRoot{}),
				cmpopts.IgnoreUnexported(hcl.TraverseAttr{}),
			}
			if diff := cmp.Diff(diags, tt.diags, opts...); diff != "" {
				got := parser.DiagString(diags)
				want := parser.DiagString(tt.diags)
				t.Errorf(`DecodeBody()
Diff (-got +want):
%s
Got:
%s
Want:
%s`, diff, indent(got, "\t"), indent(want, "\t"))
			}
		})
	}
}

// ---

type testParser struct {
	filename string
	files    map[string]*hcl.File
}

func (p *testParser) Parse(t *testing.T, src string) hcl.Body {
	t.Helper()
	if p.filename == "" {
		p.filename = t.Name()
	}
	src = unindent(src)
	src = strings.TrimSpace(src)
	f, diags := hclsyntax.ParseConfig([]byte(src), p.filename, hcl.Pos{Byte: 0, Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Errorf("Parse test body: %v", diags)
	}
	if p.files == nil {
		p.files = make(map[string]*hcl.File)
	}
	p.files[p.filename] = f
	return f.Body
}

func unindent(str string) string {
	str = strings.TrimLeft(str, "\n")
	ind := ""
	for _, c := range str {
		if !unicode.IsSpace(c) {
			break
		}
		ind += string(c)
	}
	lines := strings.Split(str, "\n")
	for i, l := range lines {
		lines[i] = strings.Replace(l, ind, "", 1)
	}
	return strings.Join(lines, "\n")
}

func indent(str string, ind string) string {
	lines := strings.Split(str, "\n")
	for i, s := range lines {
		lines[i] = ind + s
	}
	return strings.Join(lines, "\n")
}

var printWarnings = flag.Bool("warnings", false, "Print warning diagnostics")

func (p *testParser) CheckDiags(t *testing.T, diags hcl.Diagnostics) {
	t.Helper()
	if diags.HasErrors() || *printWarnings {
		t.Log(p.DiagString(diags))
	}
	if diags.HasErrors() {
		t.FailNow()
	}
}

func (p *testParser) DiagString(diags hcl.Diagnostics) string {
	var buf bytes.Buffer
	wr := hcl.NewDiagnosticTextWriter(&buf, p.files, 80, true)
	if err := wr.WriteDiagnostics(diags); err != nil {
		return fmt.Sprintf("Write diagnostics: %v", err)
	}
	return buf.String()
}

func checkPanic(t *testing.T) {
	t.Helper()
	if err := recover(); err != nil {
		c := stack.Caller(2)
		t.Fatalf("Panic: %k/%v: %v", c, c, err)
	}
}

// simpleDef is commonly used simple definition
type simpleDef struct {
	resource.Definition
	Input  *string `func:"input"`
	Output string  `func:"output"`
}

type ValidateFunc func(interface{}, string) error

func (fn ValidateFunc) Validate(val interface{}, rule string) error { return fn(val, rule) }

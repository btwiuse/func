package resource

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestResource_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name     string
		resource *Resource
	}{
		{
			"Simple",
			&Resource{
				Type: "foo",
				Name: "bar",
				Input: cty.ObjectVal(map[string]cty.Value{
					"baz": cty.StringVal("qux"),
				}),
				Output: cty.ObjectVal(map[string]cty.Value{
					"qux": cty.StringVal("baz"),
				}),
				Deps:    []string{"a", "b"},
				Sources: []string{"c", "d"},
			},
		},
		{
			"NoOutput",
			&Resource{
				Type: "foo",
				Name: "bar",
				Input: cty.ObjectVal(map[string]cty.Value{
					"baz": cty.StringVal("qux"),
				}),
				Deps:    nil,
				Sources: nil,
			},
		},
		{
			"Lists",
			&Resource{
				Type: "foo",
				Name: "bar",
				Input: cty.ObjectVal(map[string]cty.Value{
					"in": cty.ListVal([]cty.Value{
						cty.StringVal("a"),
						cty.StringVal("b"),
						cty.StringVal("c"),
					}),
				}),
				Output: cty.ObjectVal(map[string]cty.Value{
					"out": cty.TupleVal([]cty.Value{
						cty.StringVal("a"),
						cty.NumberIntVal(123),
						cty.MapVal(map[string]cty.Value{
							"b": cty.StringVal("c"),
						}),
					}),
				}),
				Deps:    nil,
				Sources: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.resource.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() err = %v", err)
			}

			t.Logf(string(b))

			got := &Resource{
				Input:  cty.UnknownVal(tt.resource.Input.Type()),
				Output: cty.UnknownVal(tt.resource.Output.Type()),
			}
			if err := got.UnmarshalJSON(b); err != nil {
				t.Fatalf("UnmarshalJSON() err = %v", err)
			}

			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Value) bool {
					if a.IsNull() {
						return b.IsNull()
					}
					if b.IsNull() {
						return false
					}
					return a.RawEquals(b)
				}),
			}
			if diff := cmp.Diff(got, tt.resource, opts...); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

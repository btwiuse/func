package graph_test

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/func/func/resource/graph"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestDependency_Parents(t *testing.T) {
	dep := graph.Dependency{
		Field: cty.GetAttrPath("input"),
		Expression: graph.Expression{
			graph.ExprLiteral{Value: cty.StringVal("foo")},
			graph.ExprReference{Path: cty.GetAttrPath("parent1").GetAttr("output")},
			graph.ExprLiteral{Value: cty.StringVal("bar")},
			graph.ExprReference{Path: cty.GetAttrPath("parent2").GetAttr("output")},
		},
	}
	want := []string{"parent1", "parent2"}

	got := dep.Parents()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Parents() (-got +want)\n%s", diff)
	}
}

func TestDependency_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		dep  graph.Dependency
	}{
		{
			"Simple",
			graph.Dependency{
				Field: cty.GetAttrPath("input"),
				Expression: graph.Expression{
					graph.ExprLiteral{Value: cty.StringVal("foo")},
				},
			},
		},
		{
			"Complex",
			graph.Dependency{
				Field: cty.GetAttrPath("input"),
				Expression: graph.Expression{
					graph.ExprLiteral{Value: cty.StringVal("foo")},
					graph.ExprReference{Path: cty.GetAttrPath("bar").Index(cty.NumberIntVal(2))},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.dep)
			if err != nil {
				t.Fatalf("Marshal() err = %v", err)
			}

			t.Log(string(b))

			var got graph.Dependency
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("Unmarshal() err = %v", err)
			}

			opts := []cmp.Option{
				cmp.Comparer(func(a, b graph.Dependency) bool {
					return a.Equals(b)
				}),
			}
			if diff := cmp.Diff(got, tt.dep, opts...); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

func ExampleDependency_MarshalJSON() {
	dep := graph.Dependency{
		Field: cty.GetAttrPath("input"),
		Expression: graph.Expression{
			graph.ExprLiteral{Value: cty.StringVal("foo")},
		},
	}

	b, err := json.MarshalIndent(dep, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))
	// Output:
	// {
	//     "field": "input",
	//     "expr": [
	//         {
	//             "lit": "foo"
	//         }
	//     ]
	// }
}

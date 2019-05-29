package graph_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
	"github.com/zclconf/go-cty/cty"
)

func ExampleGraph_MarshalJSON() {
	g := graph.Graph{
		Resources: map[string]*resource.Resource{
			"alice": {
				Type:    "person",
				Sources: []string{"abc"},
				Input: cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("alice"),
					"age":  cty.NumberIntVal(20),
				}),
			},
			"bob": {
				Type:    "person",
				Sources: []string{"abc"},
				Input: cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("bob"),
					"age":     cty.NumberIntVal(30),
					"friends": cty.ListValEmpty(cty.String),
				}),
				Deps: []string{"alice", "carol"},
			},
		},
		Dependencies: map[string][]graph.Dependency{
			"bob": {{
				Field: cty.GetAttrPath("friends"),
				Expression: graph.Expression{
					graph.ExprReference{
						Path: cty.
							GetAttrPath("alice").
							GetAttr("friends").
							Index(cty.NumberIntVal(0)),
					},
				},
			}},
		},
	}

	out, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		log.Fatalf("Marshal graph: %v", err)
	}
	fmt.Println(string(out))
	// Output:
	// {
	//     "res": {
	//         "alice": {
	//             "type": "person",
	//             "srcs": [
	//                 "abc"
	//             ],
	//             "input": {
	//                 "age": 20,
	//                 "name": "alice"
	//             }
	//         },
	//         "bob": {
	//             "type": "person",
	//             "srcs": [
	//                 "abc"
	//             ],
	//             "input": {
	//                 "age": 30,
	//                 "friends": [],
	//                 "name": "bob"
	//             },
	//             "deps": [
	//                 "alice",
	//                 "carol"
	//             ],
	//             "edges": [
	//                 {
	//                     "field": [
	//                         "friends"
	//                     ],
	//                     "expr": [
	//                         {
	//                             "ref": [
	//                                 "alice",
	//                                 "friends",
	//                                 0
	//                             ]
	//                         }
	//                     ]
	//                 }
	//             ]
	//         }
	//     }
	// }
}

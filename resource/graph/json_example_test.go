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
	g := &graph.Graph{
		Resources: map[string]*resource.Resource{
			"alice": {
				Name:    "alice",
				Type:    "person",
				Sources: []string{"abc"},
				Input: cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("alice"),
					"age":  cty.NumberIntVal(20),
				}),
			},
			"bob": {
				Name:    "bob",
				Type:    "person",
				Sources: []string{"abc"},
				Input: cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("bob"),
					"age":  cty.NumberIntVal(30),
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

	j, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(j))
	// Output:
	// {
	//     "res": [
	//         {
	//             "name": "alice",
	//             "type": "person",
	//             "input": {
	//                 "age": 20,
	//                 "name": "alice"
	//             },
	//             "src": [
	//                 "abc"
	//             ]
	//         },
	//         {
	//             "name": "bob",
	//             "type": "person",
	//             "input": {
	//                 "age": 30,
	//                 "name": "bob"
	//             },
	//             "deps": [
	//                 "alice",
	//                 "carol"
	//             ],
	//             "src": [
	//                 "abc"
	//             ]
	//         }
	//     ],
	//     "deps": {
	//         "bob": [
	//             {
	//                 "field": "friends",
	//                 "expr": [
	//                     {
	//                         "ref": "alice.friends[0]"
	//                     }
	//                 ]
	//             }
	//         ]
	//     }
	// }
}

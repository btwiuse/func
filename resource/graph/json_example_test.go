package graph_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/func/func/resource"
	resjson "github.com/func/func/resource/encoding/json"
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

	// Note: Do NOT use json.Marshal on graph.

	types := map[string]reflect.Type{
		"person": reflect.TypeOf(struct {
			Name string `func:"input"`
			Age  int    `func:"input"`
		}{}),
	}

	enc := graph.JSONEncoder{
		Codec: &resjson.Encoder{
			Registry: &resource.Registry{
				Types: types,
			},
		},
	}

	j, err := enc.Marshal(g)
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	if err = json.Indent(&buf, j, "", "    "); err != nil {
		log.Fatal(err)
	}
	fmt.Println(buf.String())
	// Output:
	// {
	//     "res": [
	//         {
	//             "name": "alice",
	//             "type": "person",
	//             "srcs": [
	//                 "abc"
	//             ],
	//             "input": {
	//                 "age": 20,
	//                 "name": "alice"
	//             }
	//         },
	//         {
	//             "name": "bob",
	//             "type": "person",
	//             "deps": [
	//                 "alice",
	//                 "carol"
	//             ],
	//             "srcs": [
	//                 "abc"
	//             ],
	//             "input": {
	//                 "age": 30,
	//                 "name": "bob"
	//             }
	//         }
	//     ],
	//     "deps": {
	//         "bob": [
	//             {
	//                 "field": [
	//                     "friends"
	//                 ],
	//                 "expr": [
	//                     {
	//                         "ref": [
	//                             "alice",
	//                             "friends",
	//                             0
	//                         ]
	//                     }
	//                 ]
	//             }
	//         ]
	//     }
	// }
}

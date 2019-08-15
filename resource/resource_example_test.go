package resource_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/func/func/resource"
	"github.com/zclconf/go-cty/cty"
)

func ExampleResource_MarshalJSON() {
	res := resource.Resource{
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
	}

	b, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))
	// Output:
	// {
	//     "name": "bar",
	//     "type": "foo",
	//     "input": {
	//         "baz": "qux"
	//     },
	//     "output": {
	//         "qux": "baz"
	//     },
	//     "deps": [
	//         "a",
	//         "b"
	//     ],
	//     "src": [
	//         "c",
	//         "d"
	//     ]
	// }
}

func ExampleResource_UnmarshalJSON() {
	res := &resource.Resource{
		Input: cty.UnknownVal(cty.Object(map[string]cty.Type{
			"baz": cty.String,
		})),
		Output: cty.UnknownVal(cty.Object(map[string]cty.Type{
			"qux": cty.String,
		})),
	}

	data := []byte(`{
	    "name": "bar",
	    "type": "foo",
	    "input": {
	        "baz": "qux"
	    },
	    "output": {
	        "qux": "baz"
	    },
	    "deps": ["a", "b"],
	    "src": ["c", "d"]
	}`)
	if err := res.UnmarshalJSON(data); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Type:    %q\n", res.Type)
	fmt.Printf("Name:    %q\n", res.Name)
	fmt.Printf("Deps:    %v\n", res.Deps)
	fmt.Printf("Sources: %v\n", res.Sources)
	fmt.Printf("Input:   %#v\n", res.Input)
	fmt.Printf("Output:  %#v\n", res.Output)
	// Output:
	// Type:    "foo"
	// Name:    "bar"
	// Deps:    [a b]
	// Sources: [c d]
	// Input:   cty.ObjectVal(map[string]cty.Value{"baz":cty.StringVal("qux")})
	// Output:  cty.ObjectVal(map[string]cty.Value{"qux":cty.StringVal("baz")})
}

func ExampleResource_UnmarshalJSON_noOutput() {
	res := &resource.Resource{
		Input: cty.UnknownVal(cty.Object(map[string]cty.Type{
			"numbers": cty.List(cty.Number),
		})),
	}

	data := []byte(`{
	    "name": "bar",
	    "type": "foo",
	    "input": {
	        "numbers": [3.14]
	    }
	}`)
	if err := res.UnmarshalJSON(data); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Type:    %q\n", res.Type)
	fmt.Printf("Name:    %q\n", res.Name)
	fmt.Printf("Deps:    %v\n", res.Deps)
	fmt.Printf("Sources: %v\n", res.Sources)
	fmt.Printf("Input:   %#v\n", res.Input)
	fmt.Printf("Output:  %#v\n", res.Output)
	// Output:
	// Type:    "foo"
	// Name:    "bar"
	// Deps:    []
	// Sources: []
	// Input:   cty.ObjectVal(map[string]cty.Value{"numbers":cty.ListVal([]cty.Value{cty.MustParseNumberVal("3.14")})})
	// Output:  cty.NilVal
}

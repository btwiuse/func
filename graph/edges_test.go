package graph_test

import (
	"fmt"
	"testing"

	"github.com/func/func/graph"
)

func TestNoop(t *testing.T) {
	// Disable full file example
}

func ExampleField_Value() {
	res := &graph.Resource{Definition: SomeDefinition{
		Str: "hello",
		Int: 123,
		Nested: Nested{
			Str: "world",
		},
	}}

	f1 := graph.Field{Resource: res, Index: []int{0}}
	f2 := graph.Field{Resource: res, Index: []int{1}}
	f3 := graph.Field{Resource: res, Index: []int{2, 0}}

	fmt.Println(f1.Value())
	fmt.Println(f2.Value())
	fmt.Println(f3.Value())
	// Output:
	// hello
	// 123
	// world
}

type SomeDefinition struct {
	Str    string
	Int    int
	Nested Nested
}

type Nested struct {
	Str string
}

func (SomeDefinition) Type() string { return "" }

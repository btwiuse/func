package resource_test

import (
	"fmt"
	"reflect"

	"github.com/func/func/resource"
)

type Resource struct {
	// Inputs
	RoleName string `input:"role_name"`

	// Outputs
	ARN string `output:"arn"`
}

func ExampleFields() {
	t := reflect.TypeOf(Resource{})

	fields := resource.Fields(t, resource.IO)

	for i, f := range fields {
		fmt.Printf("%d/%d\n", i+1, len(fields))
		fmt.Printf("  Name:      %s\n", f.Name)
		fmt.Printf("  Index:     %v\n", f.Index)
		fmt.Printf("  Direction: %s\n", f.Dir)
	}

	// Output:
	// 1/2
	//   Name:      role_name
	//   Index:     0
	//   Direction: input
	// 2/2
	//   Name:      arn
	//   Index:     1
	//   Direction: output
}

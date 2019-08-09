package ctyext_test

import (
	"fmt"

	"github.com/func/func/ctyext"
	"github.com/zclconf/go-cty/cty"
)

func ExamplePathString() {
	path := cty.
		GetAttrPath("foo").
		GetAttr("bar").
		Index(cty.NumberIntVal(1)).
		Index(cty.StringVal("baz"))

	fmt.Println(ctyext.PathString(path))
	// Output: foo.bar[1]["baz"]
}

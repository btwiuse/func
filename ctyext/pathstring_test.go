package ctyext_test

import (
	"fmt"
	"testing"

	"github.com/func/func/ctyext"
	"github.com/google/go-cmp/cmp"
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

func TestParsePathString(t *testing.T) {
	tests := []struct {
		str  string
		want cty.Path
	}{
		{
			``,
			cty.Path{},
		},
		{
			`a`,
			cty.GetAttrPath("a"),
		},
		{
			`a.b.c`,
			cty.GetAttrPath("a").GetAttr("b").GetAttr("c"),
		},
		{
			`a[1]`,
			cty.GetAttrPath("a").Index(cty.NumberIntVal(1)),
		},
		{
			`a["b"]`,
			cty.GetAttrPath("a").Index(cty.StringVal("b")),
		},
		{
			`a.b["cde"][3].f`,
			cty.GetAttrPath("a").GetAttr("b").Index(cty.StringVal("cde")).Index(cty.NumberIntVal(3)).GetAttr("f"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			got, err := ctyext.ParsePathString(tt.str)
			if err != nil {
				t.Fatalf("ParsePathString() err = %v", err)
			}
			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Path) bool {
					return a.Equals(b)
				}),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

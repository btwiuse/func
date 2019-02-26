package structdoc

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	code := `
package example

// TestStruct is a test
//
// This is a test
type TestStruct struct {
	// Field doc
	Str string ` + "`input:\"in,info\"`" + ` // field comment
	Ptr *int64 ` + "`output:\"out\"`" + `
} // comment
`

	got, err := Parse(strings.NewReader(code), "TestStruct")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := &Struct{
		Doc:     "TestStruct is a test\n\nThis is a test",
		Comment: "comment",
		Fields: []Field{
			{
				Doc:     "Field doc",
				Comment: "field comment",
				Name:    "Str",
				Pointer: false,
				Type:    "string",
				Tags: []Tag{
					{Key: "input", Name: "in", Options: []string{"info"}},
				},
			},
			{
				Doc:     "",
				Comment: "",
				Name:    "Ptr",
				Pointer: true,
				Type:    "int64",
				Tags: []Tag{
					{Key: "output", Name: "out", Options: nil},
				},
			},
		},
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got, +want)\n%s", diff)
	}

}

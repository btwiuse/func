package suggest_test

import (
	"fmt"
	"testing"

	"github.com/func/func/suggest"
)

func ExampleString() {
	userProvided := "aws_lambdafunction"
	candidates := []string{"aws:lambda_function", "aws:iam_role"}

	suggestion := suggest.String(userProvided, candidates)
	fmt.Printf("Did you mean %q?", suggestion)
	// Output: Did you mean "aws:lambda_function"?
}

func TestString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		options []string
		want    string
	}{
		{"Exact", "foo", []string{"bar", "foo"}, "foo"},
		{"Almost", "boo", []string{"bar", "foo"}, "foo"},
		{"NoMatch", "go", []string{"bar", "foo"}, ""},
		{"Long", "Lorem lipsam", []string{"Lorem ipsum", "Lorem dolor"}, "Lorem ipsum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggest.String(tt.input, tt.options)
			if got != tt.want {
				t.Errorf("Suggest(%s, %v) got = %q, want = %q", tt.input, tt.options, got, tt.want)
			}
		})
	}
}

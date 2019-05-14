package resource_test

import (
	"strings"
	"testing"

	"github.com/func/func/resource"
)

func TestRegistry_New(t *testing.T) {
	r := &resource.Registry{}

	_, err := r.New("test")
	if _, ok := err.(resource.NotSupportedError); !ok {
		t.Fatalf("Get unregistered resource; got %v, want %T", err, resource.NotSupportedError{})
	}
	if !strings.Contains(err.Error(), "test") {
		t.Errorf("Not supported error does not contain name of requested type\nGot %v", err)
	}

	r.Register("test", &mockDef{})

	_, err = r.New("test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
}

func TestRegistry_Register_notStrPtr(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("Expected panic")
		}
	}()

	r := &resource.Registry{}
	r.Register("notptr", mockDef{})
}

func TestRegistry_SuggestType(t *testing.T) {
	r := &resource.Registry{}
	r.Register("aws:lambda_function", &mockDef{})
	r.Register("aws:iam_role", &mockDef{})
	r.Register("aws:iam_policy", &mockDef{})

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Exact", "aws:lambda_function", "aws:lambda_function"},
		{"Close", "aws_lambda:function", "aws:lambda_function"},
		{"Ambiguous", "aws:iam", "aws:iam_role"}, // Return closer match
		{"NoMatch", "aws:api_gateway", ""},       // No match
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.SuggestType(tt.input)
			if got != tt.want {
				t.Errorf("SuggestType() got = %q, want = %q", got, tt.want)
			}
		})
	}
}

type mockDef struct {
	resource.Definition
}

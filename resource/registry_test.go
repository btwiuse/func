package resource_test

import (
	"strings"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
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

	r.Register(&mockDef{Typename: "test"})

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
	r.Register(notptr{})
}

func TestRegistry_SuggestType(t *testing.T) {
	r := &resource.Registry{}
	r.Register(&mockDef{Typename: "aws_lambda_function"})
	r.Register(&mockDef{Typename: "aws_iam_role"})
	r.Register(&mockDef{Typename: "aws_iam_policy"})

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Exact", "aws_lambda_function", "aws_lambda_function"},
		{"Close", "aws:lambda:function", "aws_lambda_function"},
		{"Ambiguous", "aws_iam", "aws_iam_role"}, // Return closer match
		{"NoMatch", "aws_api_gateway", ""},
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

func TestRegistry_Marshal(t *testing.T) {
	r := &resource.Registry{}
	foo := &mockDef{Typename: "foo"}
	r.Register(foo)

	b, err := r.Marshal(foo)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	t.Log(b)
	t.Log(string(b))

	got, err := r.Unmarshal(b)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if diff := cmp.Diff(foo, got); diff != "" {
		t.Errorf("Roundtrip (-before, +after)\n%s", diff)
	}
}

type mockDef struct {
	resource.Definition
	Typename string
}

func (r *mockDef) Type() string { return r.Typename }

type notptr struct {
	resource.Definition
}

func (r notptr) Type() string { return "" }

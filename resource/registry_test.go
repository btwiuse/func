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

func TestRegistry_Types(t *testing.T) {
	r := &resource.Registry{}
	r.Register("aws:lambda_function", &mockDef{})
	r.Register("aws:iam_role", &mockDef{})
	r.Register("aws:iam_policy", &mockDef{})

	got := r.Types()
	want := []string{
		"aws:iam_policy",
		"aws:iam_role",
		"aws:lambda_function",
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Types() (-got +want)\n%s", diff)
	}
}

type mockDef struct {
	resource.Definition
}

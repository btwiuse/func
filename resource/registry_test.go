package resource_test

import (
	"reflect"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
)

func TestRegistry_Type(t *testing.T) {
	r := &resource.Registry{}

	got := r.Type("nonexisting")
	if got != nil {
		t.Errorf("Nonexisting type should return nil")
	}

	r.Register("test", &mockDef{})

	got = r.Type("test")
	gotStr := got.String()
	wantStr := reflect.TypeOf(&mockDef{}).String()

	if gotStr != wantStr {
		t.Errorf("Got = %s, want = %s", gotStr, wantStr)
	}
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

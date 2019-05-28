package json

import (
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
)

func TestEncoder_rountrip(t *testing.T) {
	type mockDef struct {
		resource.Definition
		Input string
	}

	reg := &resource.Registry{}
	reg.Register("testtype", &mockDef{})

	before := resource.Resource{
		Name: "name",
		Type: "testtype",
		Def: &mockDef{
			Input: "foo",
		},
		Deps:    []string{"a", "b"},
		Sources: []string{"abc", "def"},
	}

	enc := &Encoder{Registry: reg}

	b, err := enc.MarshalResource(before)
	if err != nil {
		t.Fatalf("MarshalResource() err = %v", err)
	}

	t.Log(string(b))

	after, err := enc.UnmarshalResource(b)
	if err != nil {
		t.Fatalf("UnmarshalResource() err = %v", err)
	}

	if diff := cmp.Diff(after, before); diff != "" {
		t.Errorf("Roundtrip (-got +want)\n%s", diff)
	}
}

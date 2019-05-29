package json

import (
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestEncoder_rountrip(t *testing.T) {
	type mockDef struct {
		resource.Definition
		Input  string `func:"input"`
		Output string `func:"output"`
	}

	reg := &resource.Registry{}
	reg.Register("testtype", &mockDef{})

	before := resource.Resource{
		Name: "test",
		Type: "testtype",
		Input: cty.ObjectVal(map[string]cty.Value{
			"input": cty.StringVal("foo"),
		}),
		Output: cty.ObjectVal(map[string]cty.Value{
			"output": cty.StringVal("bar"),
		}),
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

	opts := []cmp.Option{
		cmp.Transformer("GoString", func(v cty.Value) string {
			return v.GoString()
		}),
	}
	if diff := cmp.Diff(after, before, opts...); diff != "" {
		t.Errorf("Roundtrip (-got +want)\n%s", diff)
	}
}

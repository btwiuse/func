package testsuite

import (
	"context"
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/func/func/resource"
	"github.com/go-stack/stack"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

// The Target interface is implemented by stores that persist data.
type Target interface {
	PutResource(ctx context.Context, ns, project string, resource resource.Resource) error
	DeleteResource(ctx context.Context, ns, project, name string) error
	ListResources(ctx context.Context, ns, project string) (map[string]resource.Resource, error)
}

// Config provides configuration options for the test suite.
type Config struct {
	// New is used to instantiate a new store.
	//
	// The returned done function is called on test completion, allowing
	// cleanup to be performed.
	New func(t *testing.T, types map[string]reflect.Type) (target Target, done func())
}

// Run executes the test suite for the given configuration.
func Run(t *testing.T, cfg Config) {
	run(t, "ResourceIO", cfg, resourceIO)
	run(t, "ResourceList/OtherNS", cfg, listResourcesOtherNS)
	run(t, "ResourceList/OtherProject", cfg, listResourcesOtherProject)
}

func run(t *testing.T, name string, cfg Config, testFunc func(*testing.T, Config)) {
	t.Run(name, func(t *testing.T) {
		defer checkPanic(t)
		testFunc(t, cfg)
	})
}

func resourceIO(t *testing.T, cfg Config) {
	ctx := context.Background()
	ns, proj := "ns", "proj"

	a := resource.Resource{
		Name: "a",
		Type: "t",
		Input: cty.ObjectVal(map[string]cty.Value{
			"input": cty.StringVal("foo"),
		}),
		Output: cty.ObjectVal(map[string]cty.Value{
			"output": cty.StringVal("bar"),
		}),
		Sources: []string{"abc"},
	}
	b := resource.Resource{
		Name: "b",
		Type: "t",
		Input: cty.ObjectVal(map[string]cty.Value{
			"input": cty.StringVal("bar"),
		}),
		Output: cty.ObjectVal(map[string]cty.Value{
			"output": cty.StringVal("baz"),
		}),
		Deps: []string{"a"},
	}
	c := resource.Resource{
		Name: "c",
		Type: "t",
		Input: cty.ObjectVal(map[string]cty.Value{
			"input": cty.StringVal("baz"),
		}),
		Output: cty.ObjectVal(map[string]cty.Value{
			"output": cty.StringVal("qux"),
		}),
	}

	types := map[string]reflect.Type{
		"t": reflect.TypeOf(struct {
			Input  string `func:"input"`
			Output string `func:"output"`
		}{}),
	}

	s, done := cfg.New(t, types)
	defer done()

	// Add some resources
	if err := s.PutResource(ctx, ns, proj, a); err != nil {
		t.Fatalf("PutResource() err = %+v", err)
	}
	if err := s.PutResource(ctx, ns, proj, b); err != nil {
		t.Fatalf("PutResource() err = %+v", err)
	}
	if err := s.PutResource(ctx, ns, proj, c); err != nil {
		t.Fatalf("PutResource() err = %+v", err)
	}

	opts := []cmp.Option{
		cmp.Transformer("GoString", func(v cty.Value) string { return v.GoString() }),
	}

	// List resources
	got, err := s.ListResources(ctx, ns, proj)
	if err != nil {
		t.Fatalf("ListResources() err = %+v", err)
	}
	want := map[string]resource.Resource{"a": a, "b": b, "c": c}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("(-got +want)\n%s", diff)
	}

	// Delete a resource
	if err := s.DeleteResource(ctx, "ns", proj, "b"); err != nil {
		t.Fatalf("DeleteResource() err = %+v", err)
	}

	// Update a resource
	updateA := resource.Resource{
		Name: "a",
		Type: "t",
		Input: cty.ObjectVal(map[string]cty.Value{
			"input": cty.StringVal("FOO"),
		}),
		Output: cty.ObjectVal(map[string]cty.Value{
			"output": cty.StringVal("QUX"),
		}),
	}
	if err := s.PutResource(ctx, ns, proj, updateA); err != nil {
		t.Fatalf("PutResource() err = %+v", err)
	}

	got, err = s.ListResources(ctx, ns, proj)
	if err != nil {
		t.Fatalf("ListResources() err = %+v", err)
	}
	want = map[string]resource.Resource{"a": updateA, "c": c}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("(-got +want)\n%s", diff)
	}
}

func listResourcesOtherNS(t *testing.T, cfg Config) {
	ctx := context.Background()

	types := map[string]reflect.Type{
		"t": reflect.TypeOf(struct{}{}),
	}

	s, done := cfg.New(t, types)
	defer done()

	a := resource.Resource{Name: "a", Type: "t", Input: cty.EmptyObjectVal, Output: cty.EmptyObjectVal}
	if err := s.PutResource(ctx, "ns", "proj", a); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListResources(ctx, "other", "proj")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Errorf("Got %d resources, want 0", len(got))
	}
}

func listResourcesOtherProject(t *testing.T, cfg Config) {
	ctx := context.Background()

	types := map[string]reflect.Type{
		"t": reflect.TypeOf(struct{}{}),
	}

	s, done := cfg.New(t, types)
	defer done()

	a := resource.Resource{Name: "a", Type: "atype", Input: cty.EmptyObjectVal, Output: cty.EmptyObjectVal}
	if err := s.PutResource(ctx, "ns", "proj", a); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListResources(ctx, "ns", "other")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Errorf("Got %d resources, want 0", len(got))
	}
}

func checkPanic(t *testing.T) {
	t.Helper()
	if err := recover(); err != nil {
		c := stack.Caller(2)
		debug.PrintStack()
		t.Fatalf("Panic: %k/%v: %v", c, c, err)
	}
}

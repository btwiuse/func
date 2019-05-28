package testsuite

import (
	"context"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
)

// The Target interface is implemented by stores that persist data.
type Target interface {
	Put(ctx context.Context, ns, project string, resource resource.Resource) error
	Delete(ctx context.Context, ns, project, name string) error
	List(ctx context.Context, ns, project string) (map[string]resource.Resource, error)
}

// Config provides configuration options for the test suite.
type Config struct {
	// New is used to instantiate a new store.
	// The returned done function is called on test completion, allowing
	// cleanup to be performed.
	New func(t *testing.T) (target Target, done func())
}

// Run executes the test suite for the given configuration.
func Run(t *testing.T, cfg Config) {
	run(t, "IO", cfg, io)
	run(t, "List/OtherNS", cfg, listResourcesOtherNS)
	run(t, "List/OtherProject", cfg, listResourcesOtherProject)
}

func run(t *testing.T, name string, cfg Config, testFunc func(*testing.T, Target)) {
	t.Run(name, func(t *testing.T) {
		store, done := cfg.New(t)
		defer done()
		testFunc(t, store)
	})
}

func io(t *testing.T, s Target) {
	ctx := context.Background()
	ns, proj := "ns", "proj"

	a := resource.Resource{
		Name: "a",
		Type: "atype",
		Def: mockDef{
			Input: "foo",
		},
		Sources: []string{"abc"},
	}
	b := resource.Resource{
		Name: "b",
		Type: "btype",
		Def: mockDef{
			Input: "bar",
		},
		Deps: []string{"a"},
	}
	c := resource.Resource{
		Name: "c",
		Type: "ctype",
		Def: mockDef{
			Input: "baz",
		},
	}

	// Add some resources
	if err := s.Put(ctx, ns, proj, a); err != nil {
		t.Fatalf("Put() err = %+v", err)
	}
	if err := s.Put(ctx, ns, proj, b); err != nil {
		t.Fatalf("Put() err = %+v", err)
	}
	if err := s.Put(ctx, ns, proj, c); err != nil {
		t.Fatalf("Put() err = %+v", err)
	}

	// List
	got, err := s.List(ctx, ns, proj)
	if err != nil {
		t.Fatalf("List() err = %+v", err)
	}
	want := map[string]resource.Resource{"a": a, "b": b, "c": c}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got +want)\n%s", diff)
	}

	// Delete a resource
	if err := s.Delete(ctx, "ns", proj, "b"); err != nil {
		t.Fatalf("DeleteResource() err = %+v", err)
	}

	// Update a resource
	updateA := resource.Resource{
		Name: "a",
		Type: "atype",
		Def: mockDef{
			Input: "qux",
		},
	}
	if err := s.Put(ctx, ns, proj, updateA); err != nil {
		t.Fatalf("Put() err = %+v", err)
	}

	got, err = s.List(ctx, ns, proj)
	if err != nil {
		t.Fatalf("List() err = %+v", err)
	}
	want = map[string]resource.Resource{"a": updateA, "c": c}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got +want)\n%s", diff)
	}
}

func listResourcesOtherNS(t *testing.T, s Target) {
	ctx := context.Background()

	a := resource.Resource{Name: "a", Type: "atype"}
	if err := s.Put(ctx, "ns", "proj", a); err != nil {
		t.Fatal(err)
	}

	got, err := s.List(ctx, "other", "proj")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Errorf("Got %d resources, want 0", len(got))
	}
}

func listResourcesOtherProject(t *testing.T, s Target) {
	ctx := context.Background()

	a := resource.Resource{Name: "a", Type: "atype"}
	if err := s.Put(ctx, "ns", "proj", a); err != nil {
		t.Fatal(err)
	}

	got, err := s.List(ctx, "ns", "other")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Errorf("Got %d resources, want 0", len(got))
	}
}

type mockDef struct {
	Input string
}

func (mockDef) Create(context.Context, *resource.CreateRequest) error { return nil }
func (mockDef) Update(context.Context, *resource.UpdateRequest) error { return nil }
func (mockDef) Delete(context.Context, *resource.DeleteRequest) error { return nil }

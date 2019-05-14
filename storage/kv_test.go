package storage_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/storage"
	"github.com/func/func/storage/kvbackend"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestKV(t *testing.T) {
	s := &storage.KV{
		Backend: &kvbackend.Memory{},
		Registry: resource.RegistryFromResources(map[string]resource.Definition{
			"mock": &mockDef{},
		}),
	}

	ctx := context.Background()
	ns := "ns"
	proj := "proj"

	// Empty
	got, err := s.List(ctx, ns, proj)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List returned %d items, want zero\n%v", len(got), got)
	}

	res1 := resource.Resource{Type: "mock", Name: "a", Def: &mockDef{Value: "foo"}, Sources: []string{"abc"}}
	if err := s.Put(ctx, ns, proj, res1); err != nil {
		t.Fatalf("Put() res1 error = %v", err)
	}

	res2 := resource.Resource{Type: "mock", Name: "b", Def: &mockDef{Value: "bar"}, Deps: []string{"a"}}
	if err := s.Put(ctx, ns, proj, res2); err != nil {
		t.Fatalf("Put() res2 error = %v", err)
	}

	opts := []cmp.Option{
		cmpopts.SortSlices(func(a, b resource.Resource) bool {
			return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b)
		}),
		cmpopts.EquateEmpty(),
	}

	got, err = s.List(ctx, ns, proj)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := []resource.Resource{res1, res2}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("List() (-got, +want)\n%s", diff)
	}

	err = s.Delete(ctx, ns, proj, "mock", "nonexisting")
	if err == nil {
		t.Errorf("Delete() non-existing returned nil error")
	}

	err = s.Delete(ctx, ns, proj, "mock", "a")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	got, err = s.List(ctx, ns, proj)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want = []resource.Resource{res2}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("List() (-got, +want)\n%s", diff)
	}
}

type mockDef struct {
	resource.Definition
	Value string
}

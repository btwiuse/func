package storage_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/func/func/resource/hash"
	"github.com/func/func/storage"
	"github.com/func/func/storage/kvbackend"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestKV(t *testing.T) {
	s := &storage.KV{
		Backend:       &kvbackend.Memory{},
		ResourceCodec: mockCodec{},
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

	res1 := resource.Resource{Name: "a", Def: &mockDef{Value: "foo"}, Sources: []string{"abc"}}
	if err := s.Put(ctx, ns, proj, res1); err != nil {
		t.Fatalf("Put() res1 error = %v", err)
	}

	res2 := resource.Resource{Name: "b", Def: &mockDef{Value: "bar"}, Deps: []resource.Dependency{
		{Type: "mockDef", Name: "a"},
	}}
	if err := s.Put(ctx, ns, proj, res2); err != nil {
		t.Fatalf("Put() res2 error = %v", err)
	}

	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(graph.Resource{}),
		cmpopts.SortSlices(func(a, b resource.Resource) bool {
			return hash.Compute(a.Def) < hash.Compute(b.Def)
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

type mockCodec struct{}

func (mockCodec) Marshal(def resource.Definition) ([]byte, error) {
	return json.Marshal(def)
}

func (mockCodec) Unmarshal(data []byte) (resource.Definition, error) {
	var m mockDef
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

type mockDef struct {
	resource.Definition
	Value string `input:"value"`
}

func (mockDef) Type() string { return "mock" }

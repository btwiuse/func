package reconciler_test

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/graph/reconciler"
	"github.com/func/func/graph/reconciler/mock"
	"github.com/func/func/graph/snapshot"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
)

func TestReconciler_Reconcile_empty(t *testing.T) {
	r := &reconciler.Reconciler{State: &mock.Store{}}

	err := r.Reconcile(context.Background(), "ns", config.Project{Name: "empty"}, graph.New())
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
}

func TestReconciler_Reconcile_noop(t *testing.T) {
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "foo", Def: &noopDef{Input: "bar"}}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "noop", Name: "foo", Def: &noopDef{Input: "bar"}}, // exact match to existing resource
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, nil)
}

func TestReconciler_Reconcile_noopWithSource(t *testing.T) {
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{
			Type:    "noop",
			Name:    "foo",
			Def:     &noopDef{Input: "bar"},
			Sources: []string{"abc", "xyz"},
		}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{{ // exact match to existing resource
			Type:    "noop",
			Name:    "foo",
			Def:     &noopDef{Input: "bar"},
			Sources: []string{"abc", "xyz"},
		}},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, nil)
}

func TestReconciler_Reconcile_create(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "noop", Name: "foo", Def: &noopDef{Input: "bar"}},
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "foo",
			Def:  &noopDef{Input: "bar"},
		}},
	})
}

func TestReconciler_Reconcile_noUpdateOther(t *testing.T) {
	tests := []struct {
		name         string
		ns1, ns2     string
		proj1, proj2 string
	}{
		{"DiffNS", "ns1", "ns2", "proj", "proj"},
		{"DiffProj", "ns", "ns", "a", "b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resource.Resource{Type: "noop", Name: "foo", Def: &noopDef{Input: "bar"}}

			// Existing in other namespace or project but otherwise identical resource.
			existing := []mock.Resource{{NS: tt.ns1, Proj: tt.proj1, Res: res}}

			store := &mock.Store{Resources: existing}
			r := &reconciler.Reconciler{State: store}

			desired := fromSnapshot(t, snapshot.Snap{Resources: []resource.Resource{res}})

			if err := r.Reconcile(context.Background(), tt.ns2, config.Project{Name: tt.proj2}, desired); err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			assertEvents(t, store, []mock.Event{
				// Not update of other ns/project
				{Op: "create", NS: tt.ns2, Proj: tt.proj2, Res: resource.Resource{
					Type: "noop",
					Name: "foo",
					Def:  &noopDef{Input: "bar"},
				}},
			})
		})
	}
}

func TestReconciler_Reconcile_createWithDependencies(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			// Deliberately out of order to ensure dependency order is followed.
			{Type: "concat", Name: "b", Def: &concatDef{Add: "b"}},
			{Type: "concat", Name: "c", Def: &concatDef{Add: "c"}},
			{Type: "concat", Name: "a", Def: &concatDef{Add: "a"}},
		},
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${b.in}": "${a.out}",
			"${c.in}": "${b.out}",
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "concat",
			Name: "a",
			Def:  &concatDef{In: "", Add: "a", Out: "a"},
			Deps: nil,
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "concat",
			Name: "b",
			Def:  &concatDef{In: "a", Add: "b", Out: "ab"},
			Deps: []string{"a"},
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "concat",
			Name: "c",
			Def:  &concatDef{In: "ab", Add: "c", Out: "abc"},
			Deps: []string{"b"},
		}},
	})
}

func TestReconciler_Reconcile_create_sourceCode(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	var got []string

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "mock", Name: "src", Def: &mockDef{
				onCreate: func(ctx context.Context, r *resource.CreateRequest) error {
					got = make([]string, len(r.Source))
					for i, s := range r.Source {
						got[i] = s.Key()
					}
					return nil
				},
			}},
		},
		Sources: []config.SourceInfo{
			{Key: "abc"},
		},
		ResourceSources: map[int][]int{
			0: {0},
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	want := []string{"abc"}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Source code (-got, +want)\n%s", diff)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type:    "mock",
			Name:    "src",
			Def:     &mockDef{},
			Sources: []string{"abc"},
		}},
	})
}

func TestReconciler_Reconcile_sourcePointer(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	strval := "hello"
	strptr := &strval

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "noop", Name: "a", Def: &noopDef{OutputPtr: strptr}},
			{Type: "noop", Name: "b", Def: &noopDef{}},
		},
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${b.in}": "${a.outptr}", // *string -> string
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "a",
			Def:  &noopDef{OutputPtr: strptr},
			Deps: nil,
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "b",
			Def:  &noopDef{Input: strval},
			Deps: []string{"a"},
		}},
	})
}

func TestReconciler_Reconcile_targetPointer(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	strval := "hello"
	strptr := &strval

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "noop", Name: "a", Def: &noopDef{Output: strval}},
			{Type: "noop", Name: "b", Def: &noopDef{}},
		},
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${b.inptr}": "${a.out}", // string -> *string
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "a",
			Def:  &noopDef{Output: strval},
			Deps: nil,
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "b",
			Def:  &noopDef{InputPtr: strptr},
			Deps: []string{"a"},
		}},
	})
}

func TestReconciler_Reconcile_update(t *testing.T) {
	tests := []struct {
		name     string
		existing []mock.Resource
		snapshot snapshot.Snap
		events   []mock.Event
	}{
		{
			"NoSource",
			[]mock.Resource{
				{NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "foo", Def: &noopDef{Input: "before"}}},
			},
			snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "noop", Name: "foo", Def: &noopDef{Input: "after"}},
				},
			},
			[]mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Type:    "noop",
					Name:    "foo",
					Def:     &noopDef{Input: "after"},
					Sources: nil,
				}},
			},
		},
		{
			// Resource has source that did not change
			"UpdateConfig",
			[]mock.Resource{
				{NS: "ns", Proj: "proj", Res: resource.Resource{
					Type:    "noop",
					Name:    "foo",
					Def:     &noopDef{Input: "foo"},
					Sources: []string{"abc"},
				}},
			},
			snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "noop", Name: "foo", Def: &noopDef{Input: "bar"}}, // updated
				},
				Sources: []config.SourceInfo{
					{Key: "abc"}, // no change
				},
				ResourceSources: map[int][]int{0: {0}},
			},
			[]mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Type:    "noop",
					Name:    "foo",
					Def:     &noopDef{Input: "bar"},
					Sources: []string{"abc"},
				}},
			},
		},
		{
			// Resource has source that did change, config did not
			"UpdateSource",
			[]mock.Resource{
				{NS: "ns", Proj: "proj", Res: resource.Resource{
					Type:    "noop",
					Name:    "foo",
					Def:     &noopDef{Input: "foo"},
					Sources: []string{"abc"},
				}},
			},
			snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "noop", Name: "foo", Def: &noopDef{Input: "foo"}}, // no change
				},
				Sources: []config.SourceInfo{
					{Key: "xyz"}, // updated
				},
				ResourceSources: map[int][]int{0: {0}},
			},
			[]mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Type:    "noop",
					Name:    "foo",
					Def:     &noopDef{Input: "foo"},
					Sources: []string{"xyz"},
				}},
			},
		},
		{
			// Resource has source, both source and config changed
			"UpdateSourceAndConfig",
			[]mock.Resource{
				{NS: "ns", Proj: "proj", Res: resource.Resource{
					Type:    "noop",
					Name:    "foo",
					Def:     &noopDef{Input: "foo"},
					Sources: []string{"abc"},
				}},
			},
			snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "noop", Name: "foo", Def: &noopDef{Input: "bar"}}, // updated
				},
				Sources: []config.SourceInfo{
					{Key: "xyz"}, // updated
				},
				ResourceSources: map[int][]int{0: {0}},
			},
			[]mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
					Type:    "noop",
					Name:    "foo",
					Def:     &noopDef{Input: "bar"},
					Sources: []string{"xyz"},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mock.Store{Resources: tt.existing}
			r := &reconciler.Reconciler{State: store}

			desired := fromSnapshot(t, tt.snapshot)

			if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			assertEvents(t, store, tt.events)
		})
	}
}

func TestReconciler_Reconcile_update_with_previous(t *testing.T) {
	prev := &mockDef{
		Value: "before",
	}

	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{Type: "mock", Name: "foo", Def: prev}},
	}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "mock", Name: "foo", Def: &mockDef{
				onUpdate: func(ctx context.Context, r *resource.UpdateRequest) error {
					prev, ok := r.Previous.(*mockDef)
					if !ok {
						return errors.Errorf("previous does not match type, got %T, want %T", r.Previous, &mockDef{})
					}
					if prev.Value != "before" {
						return errors.Errorf("Previous value does not match, got %s, want %s", prev.Value, "before")
					}
					return nil
				},
				Value: "after",
			}},
		},
	})

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{
		State:   store,
		Backoff: withoutRetry,
	}

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
}

func TestReconciler_Reconcile_keepPrevOutput(t *testing.T) {
	ptr := "old-value-to-remove"
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "a",
			Def:  &noopDef{Input: "foo", InputPtr: &ptr, Output: "existing-output"}, // InputPtr and Output were set
		}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "noop", Name: "a", Def: &noopDef{Input: "bar"}}, // Not output in input
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "a",
			Def: &noopDef{
				Input:    "bar",
				InputPtr: nil,               // should be cleared
				Output:   "existing-output", // previous output is kept
			},
		}},
	})
}

func TestReconciler_Reconcile_updateChild(t *testing.T) {
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "concat",
			Name: "a",
			Def:  &concatDef{In: "", Add: "a", Out: "a"},
		}},
		{NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "concat",
			Name: "b",
			Def:  &concatDef{In: "a", Add: "b", Out: "ab"},
		}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "concat", Name: "a", Def: &concatDef{Add: "a", Out: "a"}}, // Out is resolved to same value
			{Type: "concat", Name: "b", Def: &concatDef{Add: "x"}},           // Add changed to x
		},
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${b.in}": "${a.out}",
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{
			Op:   "update",
			NS:   "ns",
			Proj: "proj",
			Res: resource.Resource{
				Type: "concat",
				Name: "b", Def: &concatDef{In: "a", Add: "x", Out: "ax"},
				Deps: []string{"a"},
			},
		},
	})
}

func TestReconciler_Reconcile_updateParent(t *testing.T) {
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "concat",
			Name: "a",
			Def:  &concatDef{In: "", Add: "a", Out: "a"},
		}},
		{NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "concat",
			Name: "b",
			Def:  &concatDef{In: "a", Add: "b", Out: "ab"},
		}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "concat", Name: "a", Def: &concatDef{Add: "x"}}, // Add changed to x
			{Type: "concat", Name: "b", Def: &concatDef{Add: "b"}}, // Did not change, but will receive new input from a
		},
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${b.in}": "${a.out}",
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	// Parent changed so both resources will get updated.
	assertEvents(t, store, []mock.Event{
		{
			Op:   "update",
			NS:   "ns",
			Proj: "proj",
			Res: resource.Resource{
				Type: "concat",
				Name: "a",
				Def:  &concatDef{In: "", Add: "x", Out: "x"},
			},
		},
		{
			Op:   "update",
			NS:   "ns",
			Proj: "proj",
			Res: resource.Resource{
				Type: "concat",
				Name: "b",
				Def:  &concatDef{In: "x", Add: "b", Out: "xb"},
				Deps: []string{"a"},
			},
		},
	})
}

func TestReconciler_Reconcile_delete(t *testing.T) {
	existing := []mock.Resource{
		// Deliberately out of order to ensure resources are deleted in deleted
		// reverse order from dependencies (a->b->c => delete c->b->a).
		{NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "a", Def: &noopDef{}, Deps: nil}},
		{NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "c", Def: &noopDef{}, Deps: []string{"b"}}},
		{NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "b", Def: &noopDef{}, Deps: []string{"a"}}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := graph.New() // empty

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	// Parent changed so both resources will get updated.
	assertEvents(t, store, []mock.Event{
		{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "c"}},
		{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "b"}},
		{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "a"}},
	})
}

func TestReconciler_Reconcile_deleteAfterCreate(t *testing.T) {
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "foo", Def: &noopDef{Input: "old"}}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "noop", Name: "bar", Def: &noopDef{Input: "new"}},
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		// Create first
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "bar", Def: &noopDef{Input: "new"}}}, // nolint: lll
		{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "foo"}},
	})
}

func TestReconciler_Reconcile_concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	concurrencies := []int{1, 4, 8, 16}

	n := 16                          // Number of steps, all possible to execute concurrently.
	wait := 50 * time.Millisecond    // Time to wait per step.
	total := time.Duration(n) * wait // Total time, without concurrency.

	for _, c := range concurrencies {
		t.Run(strconv.Itoa(c), func(t *testing.T) {
			var snap snapshot.Snap
			for i := 0; i < n; i++ {
				res := resource.Resource{Type: "mock", Name: fmt.Sprintf("res%v", i), Def: &mockDef{
					onCreate: func(context.Context, *resource.CreateRequest) error {
						time.Sleep(wait)
						return nil
					},
				}}
				snap.Resources = append(snap.Resources, res)
			}

			r := &reconciler.Reconciler{
				State:       &mock.Store{},
				Concurrency: c,
			}

			start := time.Now()
			if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, fromSnapshot(t, snap)); err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			got := time.Since(start)         // Perceived time.
			want := total / time.Duration(c) // Total / concurrency.
			margin := want / 2               // Allow ±50% margin for comparison.

			// Print some debug info, in case this tests starts failing because of flakiness.
			conc := float64(total) / float64(got)
			concPct := 100 * conc / float64(c)
			t.Logf("Executed at concurrency %.2f/%d (%.1f%%). Difference %s/%s", conc, c, concPct, got-want, margin)

			if got < want-margin || got > want+margin {
				t.Errorf("Completed in %s, want %s ±%s", got, want, margin)
			}
		})
	}
}

func TestReconciler_Reconcile_fanIn(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "noop", Name: "a", Def: &noopDef{Output: "A"}},
			{Type: "noop", Name: "b", Def: &noopDef{Output: "B"}},
			{Type: "noop", Name: "c", Def: &noopDef{Output: "C"}},
			{Type: "noop", Name: "x", Def: &noopDef{}},
		},
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${x.in}": "${a.out}-${b.out}-${c.out}",
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	// a/b/c execute in arbitrary order.
	sort.Slice(store.Events, func(i, j int) bool {
		return store.Events[i].Res.Name < store.Events[j].Res.Name
	})
	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "a", Def: &noopDef{Output: "A"}}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "b", Def: &noopDef{Output: "B"}}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Type: "noop", Name: "c", Def: &noopDef{Output: "C"}}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "x",
			Def:  &noopDef{Input: "A-B-C"},
			Deps: []string{"a", "b", "c"},
		}},
	})
}

func TestReconciler_Reconcile_fanOut(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Type: "noop", Name: "a", Def: &noopDef{Output: "hello"}},
			{Type: "noop", Name: "x", Def: &noopDef{}},
			{Type: "noop", Name: "y", Def: &noopDef{}},
			{Type: "noop", Name: "z", Def: &noopDef{}},
		},
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${x.in}": "${a.out}",
			"${y.in}": "${a.out}",
			"${z.in}": "${a.out}",
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	// x/y/z execute in arbitrary order.
	sort.Slice(store.Events, func(i, j int) bool {
		return store.Events[i].Res.Name < store.Events[j].Res.Name
	})
	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "a",
			Def:  &noopDef{Output: "hello"},
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "x",
			Def:  &noopDef{Input: "hello"},
			Deps: []string{"a"},
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "y",
			Def:  &noopDef{Input: "hello"},
			Deps: []string{"a"},
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Type: "noop",
			Name: "z",
			Def:  &noopDef{Input: "hello"},
			Deps: []string{"a"},
		}},
	})
}

func TestReconciler_Reconcile_errParent(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{
		State:   store,
		Backoff: withoutRetry,
	}

	wantErr := errors.New("parent err")
	desired := fromSnapshot(t, snapshot.Snap{
		Resources: []resource.Resource{
			{Name: "parent", Def: &noopDef{Err: wantErr}},
			{Name: "child", Def: &noopDef{}},
		},
		Dependencies: map[snapshot.Expr]snapshot.Expr{
			"${child.in}": "${parent.out}",
		},
	})

	err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired)
	if errors.Cause(err) != wantErr {
		t.Errorf("Reconcile() error = %v, want = %v", err, wantErr)
	}

	// Child should not be executed
	assertEvents(t, store, nil)
}

// Resource Definitions

// noopDef is a no-op definition that does nothing when executed.
type noopDef struct {
	Input  string `func:"input" name:"in"`
	Output string `func:"output" name:"out"`

	InputPtr  *string `func:"input" name:"inptr"`
	OutputPtr *string `func:"output" name:"outptr"`

	Err error
}

func (n *noopDef) Create(context.Context, *resource.CreateRequest) error { return n.Err }
func (n *noopDef) Update(context.Context, *resource.UpdateRequest) error { return n.Err }
func (n *noopDef) Delete(context.Context, *resource.DeleteRequest) error { return n.Err }

// concatDef concatenates a value to the input and sets it as the output.
// Only supports Create().
type concatDef struct {
	In  string `func:"input"`
	Add string `func:"input"` // Value to add to input
	Out string `func:"output"`

	resource.Definition
}

func (c *concatDef) Create(context.Context, *resource.CreateRequest) error {
	c.Out = c.In + c.Add
	return nil
}
func (c *concatDef) Update(context.Context, *resource.UpdateRequest) error {
	c.Out = c.In + c.Add
	return nil
}

type mockDef struct {
	onCreate func(context.Context, *resource.CreateRequest) error
	onUpdate func(context.Context, *resource.UpdateRequest) error
	onDelete func(context.Context, *resource.DeleteRequest) error

	Value string `func:"input"`
}

func (s *mockDef) Create(ctx context.Context, req *resource.CreateRequest) error {
	return s.onCreate(ctx, req)
}
func (s *mockDef) Update(ctx context.Context, req *resource.UpdateRequest) error {
	return s.onUpdate(ctx, req)
}
func (s *mockDef) Delete(ctx context.Context, req *resource.DeleteRequest) error {
	return s.onDelete(ctx, req)
}

// Test helpers

func fromSnapshot(t *testing.T, snap snapshot.Snap) *graph.Graph {
	t.Helper()
	g, err := snap.Graph()
	if err != nil {
		t.Fatalf("Make graph from snapshot: %v", err)
	}
	return g
}

func assertEvents(t *testing.T, store *mock.Store, want []mock.Event) {
	t.Helper()
	opts := []cmp.Option{
		cmpopts.SortSlices(func(a, b string) bool { return a < b }),
		cmpopts.IgnoreUnexported(mockDef{}),
	}
	if diff := cmp.Diff(store.Events, want, opts...); diff != "" {
		t.Errorf("Events do not match (-got %d, +want %d)\n%s", len(store.Events), len(want), diff)
	}
}

func withoutRetry() backoff.BackOff {
	return noretry{}
}

type noretry struct{}

func (noretry) NextBackOff() time.Duration { return backoff.Stop }
func (noretry) Reset()                     {}

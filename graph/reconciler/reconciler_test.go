package reconciler_test

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/graph/reconciler"
	"github.com/func/func/graph/reconciler/mock"
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
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "foo", Def: &noopDef{Input: "bar"}}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "foo", Def: &noopDef{Input: "bar"}}, // exact match to existing resource
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
			Name:    "foo",
			Def:     &noopDef{Input: "bar"},
			Sources: []string{"abc", "xyz"},
		}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "foo", Def: &noopDef{Input: "bar"}, Sources: []string{"abc", "xyz"}}, // exact match to existing resource
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, nil)
}

func TestReconciler_Reconcile_create(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "foo", Def: &noopDef{Input: "bar"}},
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "foo", Def: &noopDef{Input: "bar"}}},
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
			res := resource.Resource{Name: "foo", Def: &noopDef{Input: "bar"}}

			// Existing in other namespace or project but otherwise identical resource.
			existing := []mock.Resource{{NS: tt.ns1, Proj: tt.proj1, Res: res}}

			store := &mock.Store{Resources: existing}
			r := &reconciler.Reconciler{State: store}

			desired := fromSnapshot(t, graph.Snapshot{Resources: []resource.Resource{res}})

			if err := r.Reconcile(context.Background(), tt.ns2, config.Project{Name: tt.proj2}, desired); err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			assertEvents(t, store, []mock.Event{
				// Not update of other ns/project
				{Op: "create", NS: tt.ns2, Proj: tt.proj2, Res: resource.Resource{Name: "foo", Def: &noopDef{Input: "bar"}}},
			})
		})
	}
}

func TestReconciler_Reconcile_createWithDependencies(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			// Deliberately out of order to ensure dependency order is followed.
			{Name: "b", Def: &concatDef{Add: "b"}},
			{Name: "c", Def: &concatDef{Add: "c"}},
			{Name: "a", Def: &concatDef{Add: "a"}},
		},
		References: []graph.SnapshotRef{
			{Source: 2, Target: 0, SourceIndex: []int{2}, TargetIndex: []int{0}}, // a -> b
			{Source: 0, Target: 1, SourceIndex: []int{2}, TargetIndex: []int{0}}, // b -> c
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "a",
			Def:  &concatDef{In: "", Add: "a", Out: "a"},
			Deps: nil,
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "b",
			Def:  &concatDef{In: "a", Add: "b", Out: "ab"},
			Deps: []resource.Dependency{
				{Type: "concat", Name: "a"},
			},
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "c",
			Def:  &concatDef{In: "ab", Add: "c", Out: "abc"},
			Deps: []resource.Dependency{
				{Type: "concat", Name: "b"},
			},
		}},
	})
}

func TestReconciler_Reconcile_create_sourceCode(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	var got []string

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "src", Def: &mockDef{
				onCreate: func(ctx context.Context, r *resource.CreateRequest) error {
					got = make([]string, len(r.Source))
					for i, s := range r.Source {
						got[i] = s.Digest()
					}
					return nil
				},
			}},
		},
		Sources: []config.SourceInfo{
			{SHA: "abc"},
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

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "a", Def: &noopDef{OutputPtr: strptr}},
			{Name: "b", Def: &noopDef{}},
		},
		References: []graph.SnapshotRef{
			// Output is a *string, input is a string
			{Source: 0, Target: 1, SourceIndex: []int{3}, TargetIndex: []int{0}}, // OutputPtr -> Input
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "a",
			Def:  &noopDef{OutputPtr: strptr},
			Deps: nil,
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "b",
			Def:  &noopDef{Input: strval},
			Deps: []resource.Dependency{
				{Type: "noop", Name: "a"},
			},
		}},
	})
}

func TestReconciler_Reconcile_targetPointer(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	strval := "hello"
	strptr := &strval

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "a", Def: &noopDef{Output: strval}},
			{Name: "b", Def: &noopDef{}},
		},
		References: []graph.SnapshotRef{
			// Output is a *string, input is a string
			{Source: 0, Target: 1, SourceIndex: []int{1}, TargetIndex: []int{2}}, // Output -> InputPtr
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "a",
			Def:  &noopDef{Output: strval},
			Deps: nil,
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "b",
			Def:  &noopDef{InputPtr: strptr},
			Deps: []resource.Dependency{
				{Type: "noop", Name: "a"},
			},
		}},
	})
}

func TestReconciler_Reconcile_update(t *testing.T) {
	tests := []struct {
		name     string
		existing []mock.Resource
		snapshot graph.Snapshot
		events   []mock.Event
	}{
		{
			"NoSource",
			[]mock.Resource{
				{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "foo", Def: &noopDef{Input: "before"}}},
			},
			graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "foo", Def: &noopDef{Input: "after"}},
				},
			},
			[]mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
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
					Name:    "foo",
					Def:     &noopDef{Input: "foo"},
					Sources: []string{"abc"},
				}},
			},
			graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "foo", Def: &noopDef{Input: "bar"}}, // updated
				},
				Sources: []config.SourceInfo{
					{SHA: "abc"}, // no change
				},
				ResourceSources: map[int][]int{0: {0}},
			},
			[]mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
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
					Name:    "foo",
					Def:     &noopDef{Input: "foo"},
					Sources: []string{"abc"},
				}},
			},
			graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "foo", Def: &noopDef{Input: "foo"}}, // no change
				},
				Sources: []config.SourceInfo{
					{SHA: "xyz"}, // updated
				},
				ResourceSources: map[int][]int{0: {0}},
			},
			[]mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
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
					Name:    "foo",
					Def:     &noopDef{Input: "foo"},
					Sources: []string{"abc"},
				}},
			},
			graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "foo", Def: &noopDef{Input: "bar"}}, // updated
				},
				Sources: []config.SourceInfo{
					{SHA: "xyz"}, // updated
				},
				ResourceSources: map[int][]int{0: {0}},
			},
			[]mock.Event{
				{Op: "update", NS: "ns", Proj: "proj", Res: resource.Resource{
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

func TestReconciler_Reconcile_updateChild(t *testing.T) {
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "a", Def: &concatDef{In: "", Add: "a", Out: "a"}}},
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "b", Def: &concatDef{In: "a", Add: "b", Out: "ab"}}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "a", Def: &concatDef{Add: "a"}}, // Out is resolved to same value
			{Name: "b", Def: &concatDef{Add: "x"}}, // Add changed to x
		},
		References: []graph.SnapshotRef{
			{Source: 0, Target: 1, SourceIndex: []int{1}, TargetIndex: []int{0}},
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
				Name: "b", Def: &concatDef{In: "a", Add: "x", Out: "ax"},
				Deps: []resource.Dependency{{Type: "concat", Name: "a"}},
			},
		},
	})
}

func TestReconciler_Reconcile_updateParent(t *testing.T) {
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "a", Def: &concatDef{In: "", Add: "a", Out: "a"}}},
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "b", Def: &concatDef{In: "a", Add: "b", Out: "ab"}}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "a", Def: &concatDef{Add: "x"}}, // Add changed to x
			{Name: "b", Def: &concatDef{Add: "b"}}, // Did not change, but will receive new input from a
		},
		References: []graph.SnapshotRef{
			{Source: 0, Target: 1, SourceIndex: []int{1}, TargetIndex: []int{0}},
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
				Name: "a", Def: &concatDef{In: "", Add: "x", Out: "x"},
			},
		},
		{
			Op:   "update",
			NS:   "ns",
			Proj: "proj",
			Res: resource.Resource{
				Name: "b", Def: &concatDef{In: "x", Add: "b", Out: "xb"},
				Deps: []resource.Dependency{{Type: "concat", Name: "a"}},
			},
		},
	})
}

func TestReconciler_Reconcile_delete(t *testing.T) {
	existing := []mock.Resource{
		// Deliberately out of order to ensure resources are deleted in deleted
		// reverse order from dependencies (a->b->c => delete c->b->a).
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "a", Def: &noopDef{}, Deps: nil}},
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "c", Def: &noopDef{}, Deps: []resource.Dependency{
			{Type: "noop", Name: "b"},
		}}},
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "b", Def: &noopDef{}, Deps: []resource.Dependency{
			{Type: "noop", Name: "a"},
		}}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := graph.New() // empty

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	// Parent changed so both resources will get updated.
	assertEvents(t, store, []mock.Event{
		{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "c"}},
		{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "b"}},
		{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "a"}},
	})
}

func TestReconciler_Reconcile_deleteAfterCreate(t *testing.T) {
	existing := []mock.Resource{
		{NS: "ns", Proj: "proj", Res: resource.Resource{Name: "foo", Def: &noopDef{Input: "old"}}},
	}

	store := &mock.Store{Resources: existing}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "bar", Def: &noopDef{Input: "new"}},
		},
	})

	if err := r.Reconcile(context.Background(), "ns", config.Project{Name: "proj"}, desired); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	assertEvents(t, store, []mock.Event{
		// Create first
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "bar", Def: &noopDef{Input: "new"}}},
		{Op: "delete", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "foo"}},
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
			var snap graph.Snapshot
			for i := 0; i < n; i++ {
				res := resource.Resource{Name: fmt.Sprintf("res%v", i), Def: &mockDef{
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

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "a", Def: &noopDef{Output: "a"}},
			{Name: "b", Def: &noopDef{Output: "b"}},
			{Name: "c", Def: &noopDef{Output: "c"}},
			{Name: "x", Def: &joinDef{}},
		},
		References: []graph.SnapshotRef{
			{Source: 0, Target: 3, SourceIndex: []int{1}, TargetIndex: []int{0}}, // a Out -> x A
			{Source: 1, Target: 3, SourceIndex: []int{1}, TargetIndex: []int{1}}, // b Out -> x B
			{Source: 2, Target: 3, SourceIndex: []int{1}, TargetIndex: []int{2}}, // c Out -> x C
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
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "a", Def: &noopDef{Output: "a"}}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "b", Def: &noopDef{Output: "b"}}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "c", Def: &noopDef{Output: "c"}}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "x",
			Def:  &joinDef{A: "a", B: "b", C: "c", Out: "a-b-c"},
			Deps: []resource.Dependency{
				{Type: "noop", Name: "a"},
				{Type: "noop", Name: "b"},
				{Type: "noop", Name: "c"},
			},
		}},
	})
}

func TestReconciler_Reconcile_fanOut(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "a", Def: &noopDef{Output: "hello"}},
			{Name: "x", Def: &noopDef{}},
			{Name: "y", Def: &noopDef{}},
			{Name: "z", Def: &noopDef{}},
		},
		References: []graph.SnapshotRef{
			{Source: 0, Target: 1, SourceIndex: []int{1}, TargetIndex: []int{0}}, // a -> x
			{Source: 0, Target: 2, SourceIndex: []int{1}, TargetIndex: []int{0}}, // a -> y
			{Source: 0, Target: 3, SourceIndex: []int{1}, TargetIndex: []int{0}}, // a -> z
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
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{Name: "a", Def: &noopDef{Output: "hello"}}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "x",
			Def:  &noopDef{Input: "hello"},
			Deps: []resource.Dependency{{Type: "noop", Name: "a"}},
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "y",
			Def:  &noopDef{Input: "hello"},
			Deps: []resource.Dependency{{Type: "noop", Name: "a"}},
		}},
		{Op: "create", NS: "ns", Proj: "proj", Res: resource.Resource{
			Name: "z",
			Def:  &noopDef{Input: "hello"},
			Deps: []resource.Dependency{{Type: "noop", Name: "a"}},
		}},
	})
}

func TestReconciler_Reconcile_errParent(t *testing.T) {
	store := &mock.Store{Resources: nil}
	r := &reconciler.Reconciler{State: store}

	wantErr := errors.New("parent err")
	desired := fromSnapshot(t, graph.Snapshot{
		Resources: []resource.Resource{
			{Name: "parent", Def: &noopDef{Err: wantErr}},
			{Name: "child", Def: &noopDef{}},
		},
		References: []graph.SnapshotRef{
			{Source: 0, Target: 1, SourceIndex: []int{1}, TargetIndex: []int{0}},
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
	Input  string `input:"in"`
	Output string `output:"out"`

	InputPtr  *string `input:"inptr"`
	OutputPtr *string `input:"outptr"`

	Err error
}

func (n *noopDef) Type() string                                          { return "noop" }
func (n *noopDef) Create(context.Context, *resource.CreateRequest) error { return n.Err }
func (n *noopDef) Update(context.Context, *resource.UpdateRequest) error { return n.Err }
func (n *noopDef) Delete(context.Context, *resource.DeleteRequest) error { return n.Err }

// concatDef concatenates a value to the input and sets it as the output.
// Only supports Create().
type concatDef struct {
	In  string `input:"in"`
	Add string `input:"add"` // Value to add to input
	Out string `output:"out"`

	resource.Definition
}

func (c *concatDef) Type() string { return "concat" }
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
}

func (s *mockDef) Type() string { return "mock" }
func (s *mockDef) Create(ctx context.Context, req *resource.CreateRequest) error {
	return s.onCreate(ctx, req)
}
func (s *mockDef) Update(ctx context.Context, req *resource.UpdateRequest) error {
	return s.onUpdate(ctx, req)
}
func (s *mockDef) Delete(ctx context.Context, req *resource.DeleteRequest) error {
	return s.onDelete(ctx, req)
}

// joinDef is a no-op definition that does nothing when executed.
type joinDef struct {
	A   string `input:"a"`
	B   string `input:"b"`
	C   string `input:"c"`
	Out string `output:"out"`
}

func (j *joinDef) Type() string                                          { return "noop" }
func (j *joinDef) Create(context.Context, *resource.CreateRequest) error { j.run(); return nil }
func (j *joinDef) Update(context.Context, *resource.UpdateRequest) error { j.run(); return nil }
func (j *joinDef) Delete(context.Context, *resource.DeleteRequest) error { return nil }
func (j *joinDef) run() {
	j.Out = fmt.Sprintf("%s-%s-%s", j.A, j.B, j.C)
}

// Test helpers

func fromSnapshot(t *testing.T, snap graph.Snapshot) *graph.Graph {
	t.Helper()
	g, err := graph.FromSnapshot(snap)
	if err != nil {
		t.Fatalf("Make graph from snapshot: %v", err)
	}
	return g
}

func assertEvents(t *testing.T, store *mock.Store, want []mock.Event) {
	t.Helper()
	opts := []cmp.Option{
		cmpopts.SortSlices(func(a, b resource.Dependency) bool {
			return a.String() < b.String()
		}),
		cmpopts.IgnoreUnexported(mockDef{}),
	}
	if diff := cmp.Diff(store.Events, want, opts...); diff != "" {
		t.Errorf("Events do not match (-got, +want)\n%s", diff)
	}
}

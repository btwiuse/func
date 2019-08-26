package teststore

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
)

type store interface {
	PutResource(ctx context.Context, project string, res *resource.Resource) error
	DeleteResource(ctx context.Context, project string, res *resource.Resource) error
	ListResources(ctx context.Context, project string) ([]*resource.Resource, error)
	PutGraph(ctx context.Context, project string, g *resource.Graph) error
	GetGraph(ctx context.Context, project string) (*resource.Graph, error)
}

// A Recorder acts as a wrapper to a store. It records all transactions with
// the store for test or debugging purposes.
type Recorder struct {
	Store store

	mu     sync.Mutex
	Events Events
}

// Events is a collection of events.
type Events []Event

// Equals returns true if the events match other events.
func (ee Events) Equals(other Events) bool {
	if len(ee) != len(other) {
		return false
	}
	for i, a := range ee {
		if !a.Equals(other[i]) {
			return false
		}
	}
	return true
}

// Diff returns a diff of events. Returns an empty string if the events are equal.
func (ee Events) Diff(other Events) string {
	opts := []cmp.Option{
		cmp.Comparer(func(a, b Event) bool {
			return a.Equals(b)
		}),
	}
	return cmp.Diff(ee, other, opts...)
}

// String returns a string of all events that have occurred.
//
// If no events have been recorded, returns
//  <no events>
func (ee Events) String() string {
	n := len(ee)
	if n == 0 {
		return "<no events>"
	}
	ss := make([]string, len(ee))
	for i, e := range ee {
		ss[i] = e.String()
	}
	return fmt.Sprintf("%v", ss)
}

// An Event is a recorded event.
type Event struct {
	Method  string      // Called method.
	Project string      // Project that was passed in.
	Data    interface{} // Arbitrary data. Content depends on the method.
	Err     error       // Error that was returned from call.
}

// Equals returns true if the two events are equal.
func (ev Event) Equals(other Event) bool {
	if ev.Method != other.Method {
		return false
	}
	if ev.Project != other.Project {
		return false
	}
	if cmp.Equal(ev.Data, other.Data) {
		return false
	}
	if ev.Err != other.Err {
		return false
	}
	return true
}

func (ev Event) String() string {
	var buf bytes.Buffer
	buf.WriteString(ev.Method)
	buf.WriteString("(project: ")
	buf.WriteString(ev.Project)
	buf.WriteString(") ")
	if ev.Data != "" {
		fmt.Fprintf(&buf, "data: %v", ev.Data)
	}
	if ev.Err != nil {
		buf.WriteString(" -> ")
		buf.WriteString(ev.Err.Error())
	}
	return buf.String()
}

// PutResource calls the corresponding method on the underlying store and records the event.
//
// Resource is set as event data.
func (r *Recorder) PutResource(ctx context.Context, project string, res *resource.Resource) error {
	ev := Event{
		Method:  "PutResource",
		Project: project,
		Data:    res,
	}
	err := r.Store.PutResource(ctx, project, res)
	if err != nil {
		ev.Err = err
	}
	r.mu.Lock()
	r.Events = append(r.Events, ev)
	r.mu.Unlock()
	return err
}

// DeleteResource calls the corresponding method on the underlying store and records the event.
//
// Resource is set as event data.
func (r *Recorder) DeleteResource(ctx context.Context, project string, res *resource.Resource) error {
	ev := Event{
		Method:  "DeleteResource",
		Project: project,
		Data:    res,
	}
	err := r.Store.DeleteResource(ctx, project, res)
	if err != nil {
		ev.Err = err
	}
	r.mu.Lock()
	r.Events = append(r.Events, ev)
	r.mu.Unlock()
	return err
}

// ListResources calls the corresponding method on the underlying store and records the event.
func (r *Recorder) ListResources(ctx context.Context, project string) ([]*resource.Resource, error) {
	ev := Event{
		Method:  "ListResources",
		Project: project,
	}
	out, err := r.Store.ListResources(ctx, project)
	if err != nil {
		ev.Err = err
	}
	r.mu.Lock()
	r.Events = append(r.Events, ev)
	r.mu.Unlock()
	return out, err
}

// PutGraph calls the corresponding method on the underlying store and records the event.
func (r *Recorder) PutGraph(ctx context.Context, project string, g *resource.Graph) error {
	ev := Event{
		Method:  "PutGraph",
		Project: project,
		Data:    g,
	}
	err := r.Store.PutGraph(ctx, project, g)
	if err != nil {
		ev.Err = err
	}
	r.mu.Lock()
	r.Events = append(r.Events, ev)
	r.mu.Unlock()
	return err
}

// GetGraph calls the corresponding method on the underlying store and records the event.
func (r *Recorder) GetGraph(ctx context.Context, project string) (*resource.Graph, error) {
	ev := Event{
		Method:  "GetGraph",
		Project: project,
	}
	out, err := r.Store.GetGraph(ctx, project)
	if err != nil {
		ev.Err = err
	}
	r.mu.Lock()
	r.Events = append(r.Events, ev)
	r.mu.Unlock()
	return out, err
}

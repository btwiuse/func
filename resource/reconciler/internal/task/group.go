package task

import "sync"

// Group executes keyed tasks exactly once. It behaves very similarly to
// sync.Once, except different tasks can be invoked with different keys.
type Group struct {
	mu    sync.Mutex
	tasks map[string]*task
	wg    sync.WaitGroup
}

type task struct {
	once sync.Once
	err  error
}

// NewGroup creates a new task group.
func NewGroup() *Group {
	return &Group{
		tasks: make(map[string]*task),
	}
}

// Do invokes the given function exactly once. Concurrent calls to Do will
// block until the first one has finished. Calls to Do with another key will
// not block.
//
// Calls after the first call will immediately return without invoking the
// function and return the error (if any) that was returned from the call.
func (g *Group) Do(key string, fn func() error) error {
	g.wg.Add(1)

	g.mu.Lock()
	t, ok := g.tasks[key]
	if !ok {
		t = &task{}
		g.tasks[key] = t
	}
	g.mu.Unlock()

	// Do will block if concurrent calls are in-flight.
	t.once.Do(func() { t.err = fn() })

	g.wg.Done()
	return t.err
}

// Wait blocks until all in-flight tasks are completed.
func (g *Group) Wait() {
	g.wg.Wait()
}

package task_test

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/func/func/resource/reconciler/internal/task"
)

func TestGroup_Do_sameKey(t *testing.T) {
	var got string

	g := task.NewGroup()
	_ = g.Do("a", func() error {
		got = "first"
		return nil
	})
	_ = g.Do("a", func() error {
		got = "second"
		return nil
	})

	want := "first"
	if got != want {
		t.Errorf("Got = %q, want = %q", got, want)
	}
}

func TestGroup_Do_diffKey(t *testing.T) {
	var got string

	g := task.NewGroup()
	_ = g.Do("a", func() error {
		got = "initial"
		return nil
	})
	_ = g.Do("b", func() error {
		got = "update"
		return nil
	})

	want := "update"
	if got != want {
		t.Errorf("Got = %q, want = %q", got, want)
	}
}

func TestGroup_Do_err(t *testing.T) {
	g := task.NewGroup()
	err := g.Do("a", func() error {
		return fmt.Errorf("err")
	})
	g.Wait()

	if err == nil {
		log.Fatal("nil error was returned")
	}
}

func TestGroup_Do_concurrent(t *testing.T) {
	g := task.NewGroup()

	var wg sync.WaitGroup

	var got int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = g.Do("a", func() error {
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&got, 1)
				return nil
			})
		}()
	}

	// Ensure all goroutines have started
	wg.Wait()

	// Wait for all tasks to complete
	g.Wait()

	want := int32(1)
	if got != want {
		t.Errorf("Got = %d, want = %d", got, want)
	}
}

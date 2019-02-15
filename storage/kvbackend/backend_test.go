package kvbackend

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/func/func/storage"
	"github.com/pkg/errors"
)

func TestBackend_io(t *testing.T) {
	tests := []struct {
		name   string
		create func(t *testing.T) (store storage.KVBackend, done func())
	}{
		{
			"Memory",
			func(*testing.T) (storage.KVBackend, func()) {
				return &Memory{}, func() {}
			},
		},
		{
			"Bolt",
			func(t *testing.T) (storage.KVBackend, func()) {
				tmp, err := ioutil.TempFile("", "bolt-test")
				if err != nil {
					t.Fatal(err)
				}
				if err = tmp.Close(); err != nil {
					t.Fatal(err)
				}
				bolt, err := NewBoltWithFile(tmp.Name())
				if err != nil {
					t.Fatal(err)
				}
				return bolt, func() {
					if err := bolt.Close(); err != nil {
						t.Errorf("close db: %v", err)
					}
					if err := os.Remove(tmp.Name()); err != nil {
						t.Errorf("remove db file: %v", err)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be, done := tt.create(t)
			defer done()

			ctx := context.Background()

			// Get non-existing
			_, err := be.Get(ctx, "foo/bar")
			if errors.Cause(err) != storage.ErrNotFound {
				t.Errorf("Get non-expecting key; want error = %v, got = %v", storage.ErrNotFound, err)
			}

			// Create
			err = be.Put(ctx, "foo/bar", []byte("baz"))
			if err != nil {
				t.Fatalf("Create error = %v", err)
			}

			// Get existing
			assertValue(t, be, "foo/bar", []byte("baz"))

			// Update
			err = be.Put(ctx, "foo/bar", []byte("qux"))
			if err != nil {
				t.Fatalf("Update error = %v", err)
			}
			assertValue(t, be, "foo/bar", []byte("qux"))

			// Create another
			err = be.Put(ctx, "foo/baz", []byte("123"))
			if err != nil {
				t.Fatalf("Create another error = %v", err)
			}

			// Scan non-existing
			assertScan(t, be, "nonexisting", nil)

			// Scan existing
			assertScan(t, be, "foo", map[string][]byte{
				"foo/bar": []byte("qux"),
				"foo/baz": []byte("123"),
			})

			// Delete non-existing key
			err = be.Delete(ctx, "foo/nonexisting")
			if err == nil {
				t.Error("Delete() nonexisting returned nil error")
			}

			err = be.Delete(ctx, "foo/bar")
			if err != nil {
				t.Errorf("Delete() error = %v", err)
			}

			_, err = be.Get(ctx, "foo/bar")
			if errors.Cause(err) != storage.ErrNotFound {
				t.Errorf("Get deleted key; want error = %v, got = %v", storage.ErrNotFound, err)
			}
		})
	}
}

func assertValue(t *testing.T, be storage.KVBackend, key string, want []byte) {
	t.Helper()
	got, err := be.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get updated error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Get updated\nGot:  %q\nWant: %q", got, want)
	}
}

func assertScan(t *testing.T, be storage.KVBackend, prefix string, want map[string][]byte) {
	t.Helper()
	got, err := be.Scan(context.Background(), prefix)
	if err != nil {
		t.Fatalf("Scan error = %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("Scan() got %d, want %d", len(got), len(want))
	}
	if want == nil && len(got) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan results\nGot:  %#v\nWant: %#v", got, want)
	}
}

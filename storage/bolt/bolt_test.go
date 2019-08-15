package bolt

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/storage/testsuite"
)

func TestBolt(t *testing.T) {
	testsuite.Run(t, testsuite.Config{
		New: func(t *testing.T, types map[string]reflect.Type) (testsuite.Target, func()) {
			file, err := ioutil.TempFile("", "bolt-test")
			if err != nil {
				t.Fatal(err)
			}
			if err = file.Close(); err != nil {
				t.Fatal(err)
			}
			bolt, err := New(file.Name(), &resource.Registry{
				Types: types,
			})
			if err != nil {
				t.Fatalf("Create db: %v", err)
			}

			done := func() {
				if err := bolt.Close(); err != nil {
					t.Errorf("Close bolt: %v", err)
				}
				if err := os.Remove(file.Name()); err != nil {
					t.Errorf("Delete bolt file: %v", err)
				}
			}

			return bolt, done
		},
	})
}

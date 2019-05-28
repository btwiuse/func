package bolt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/storage/testsuite"
	"github.com/pkg/errors"
)

func TestBolt(t *testing.T) {
	testsuite.Run(t, testsuite.Config{
		New: func(t *testing.T) (testsuite.Target, func()) {
			file, err := ioutil.TempFile("", "bolt-test")
			if err != nil {
				t.Fatal(err)
			}
			if err = file.Close(); err != nil {
				t.Fatal(err)
			}
			bolt, err := New(file.Name(), &testCodec{})
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

type testCodec struct {
	types map[string]reflect.Type
}

type jsonResource struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Deps    []string        `json:"dep,omitempty"`
	Sources []string        `json:"srcs,omitempty"`
	Def     json.RawMessage `json:"def"`
}

func (c *testCodec) MarshalResource(res resource.Resource) ([]byte, error) {
	if c.types == nil {
		c.types = make(map[string]reflect.Type)
	}
	typ := reflect.TypeOf(res.Def)
	c.types[res.Type] = typ
	def, err := json.Marshal(res.Def)
	if err != nil {
		return nil, errors.Wrap(err, "marshal definition")
	}
	r := jsonResource{
		Name:    res.Name,
		Type:    res.Type,
		Deps:    res.Deps,
		Sources: res.Sources,
		Def:     def,
	}
	return json.Marshal(r)
}

func (c *testCodec) UnmarshalResource(b []byte) (resource.Resource, error) {
	var res jsonResource
	if err := json.Unmarshal(b, &res); err != nil {
		return resource.Resource{}, errors.Wrap(err, "unmarshal")
	}
	t, ok := c.types[res.Type]
	if !ok {
		return resource.Resource{}, fmt.Errorf("type not registered: %q", res.Type)
	}
	var def resource.Definition
	if t != nil {
		v := reflect.New(t)
		if err := json.Unmarshal(res.Def, v.Interface()); err != nil {
			return resource.Resource{}, errors.Wrap(err, "unmarshal")
		}
		def = v.Elem().Interface().(resource.Definition)
	}
	return resource.Resource{
		Name:    res.Name,
		Type:    res.Type,
		Deps:    res.Deps,
		Sources: res.Sources,
		Def:     def,
	}, nil
}

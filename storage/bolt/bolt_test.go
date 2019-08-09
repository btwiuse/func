package bolt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/storage/testsuite"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
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

// testCodec encodes data as json. Rather than pre-registering types, types are
// registered on-demand as they are encoded. This allows tests to be simpler as
// types do not have to be registered in advance.
type testCodec struct {
	types map[string]io
}

type io struct {
	input, output cty.Type
}

type jsonResource struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Deps    []string        `json:"deps,omitempty"`
	Sources []string        `json:"srcs,omitempty"`
	Input   json.RawMessage `json:"input"`
	Output  json.RawMessage `json:"output"`
}

func (c *testCodec) MarshalResource(res resource.Resource) ([]byte, error) {
	if c.types == nil {
		c.types = make(map[string]io)
	}

	t, ok := c.types[res.Type]
	if !ok {
		i, err := gocty.ImpliedType(res.Input)
		if err != nil {
			return nil, err
		}
		o, err := gocty.ImpliedType(res.Output)
		if err != nil {
			return nil, err
		}
		t = io{i, o}
		c.types[res.Type] = t
	}

	input, err := ctyjson.Marshal(res.Input, t.input)
	if err != nil {
		return nil, errors.Wrap(err, "marshal input")
	}
	output, err := ctyjson.Marshal(res.Output, t.output)
	if err != nil {
		return nil, errors.Wrap(err, "marshal output")
	}

	return json.Marshal(jsonResource{
		Name:    res.Name,
		Type:    res.Type,
		Deps:    res.Deps,
		Sources: res.Sources,
		Input:   input,
		Output:  output,
	})
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
	input, err := ctyjson.Unmarshal(res.Input, t.input)
	if err != nil {
		return resource.Resource{}, errors.Wrap(err, "unmarshal input")
	}
	output, err := ctyjson.Unmarshal(res.Output, t.output)
	if err != nil {
		return resource.Resource{}, errors.Wrap(err, "unmarshal output")
	}

	return resource.Resource{
		Name:    res.Name,
		Type:    res.Type,
		Deps:    res.Deps,
		Sources: res.Sources,
		Input:   input,
		Output:  output,
	}, nil
}

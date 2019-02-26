package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// The KVBackend is used for persisting key-value data.
type KVBackend interface {
	// Put creates or updates a key.
	Put(ctx context.Context, key string, value []byte) error

	// Get returns the given key. Returns ErrNotFound if the given key does not
	// exist.
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete deletes a key. Returns ErrNotFound if the given key does not exist.
	Delete(ctx context.Context, key string) error

	// Scan returns a key-value map of all keys matching the given prefix.
	Scan(ctx context.Context, prefix string) (map[string][]byte, error)
}

// A ResourceCodec encodes and decodes resource definitions to/from binary
// representations.
type ResourceCodec interface {
	Marshal(def resource.Definition) ([]byte, error)
	Unmarshal(data []byte) (resource.Definition, error)
}

// KV is a Key-Value store.
type KV struct {
	Backend       KVBackend     // Backend to use for persisting data.
	ResourceCodec ResourceCodec // Used for resource encoding/decoding.
}

// an envelope wraps the data and is used when marshalling to json.
type envelope struct {
	Name    string          `json:"name"`
	Def     json.RawMessage `json:"def"`
	Deps    [][2]string     `json:"deps,omitempty"`
	Sources []string        `json:"srcs,omitempty"`
}

// Put stores a resource for a namespace-project.
func (kv *KV) Put(ctx context.Context, ns, project string, res resource.Resource) error {
	def, err := kv.ResourceCodec.Marshal(res.Def)
	if err != nil {
		return errors.Wrap(err, "marshal definition")
	}

	deps := make([][2]string, len(res.Deps))
	for i, d := range res.Deps {
		deps[i] = [2]string{d.Type, d.Name}
	}

	env := envelope{
		Name:    res.Name,
		Def:     def,
		Deps:    deps,
		Sources: res.Sources,
	}
	j, err := json.Marshal(env)
	if err != nil {
		return errors.Wrap(err, "marshal envelope")
	}

	k := fmt.Sprintf("%s/%s/%s:%s", ns, project, res.Def.Type(), res.Name)

	if err := kv.Backend.Put(ctx, k, j); err != nil {
		return errors.Wrap(err, "store")
	}

	return nil
}

// Delete deletes a single resource.
func (kv *KV) Delete(ctx context.Context, ns, project, typename, name string) error {
	k := fmt.Sprintf("%s/%s/%s:%s", ns, project, typename, name)
	if err := kv.Backend.Delete(ctx, k); err != nil {
		return errors.Wrap(err, "delete")
	}
	return nil
}

// List lists all resource for a given namespace-project.
func (kv *KV) List(ctx context.Context, ns, project string) ([]resource.Resource, error) {
	prefix := fmt.Sprintf("%s/%s", ns, project)
	values, err := kv.Backend.Scan(ctx, prefix)
	if err != nil {
		return nil, errors.Wrap(err, "scan")
	}

	ret := make([]resource.Resource, 0, len(values))
	for _, v := range values {
		var env envelope
		if err := json.Unmarshal(v, &env); err != nil {
			return nil, errors.Wrap(err, "unmarshal stored resource")
		}
		def, err := kv.ResourceCodec.Unmarshal(env.Def)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal resource definition")
		}
		deps := make([]resource.Dependency, len(env.Deps))
		for i, d := range env.Deps {
			deps[i] = resource.Dependency{Type: d[0], Name: d[1]}
		}
		res := resource.Resource{Name: env.Name, Def: def, Deps: deps, Sources: env.Sources}
		ret = append(ret, res)
	}
	return ret, nil
}

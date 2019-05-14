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

// A ResourceRegistry can create new resource definitions with a given type.
type ResourceRegistry interface {
	New(typename string) (resource.Definition, error)
}

// KV is a Key-Value store.
type KV struct {
	Backend  KVBackend        // Backend to use for persisting data.
	Registry ResourceRegistry // Used for instantiating new definition from the stored data.
}

type resEnvelope struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Deps    []string        `json:"deps,omitempty"`
	Sources []string        `json:"srcs,omitempty"`
}

// Put stores a resource for a namespace-project.
func (kv *KV) Put(ctx context.Context, ns, project string, res resource.Resource) error {
	if res.Type == "" {
		return errors.New("resource type not set")
	}
	if res.Name == "" {
		return errors.New("resource name not set")
	}
	data, err := json.Marshal(res.Def)
	if err != nil {
		return errors.Wrap(err, "marshal resource definition")
	}
	env := resEnvelope{
		Name:    res.Name,
		Type:    res.Type,
		Data:    data,
		Deps:    res.Deps,
		Sources: res.Sources,
	}
	j, err := json.Marshal(env)
	if err != nil {
		return errors.Wrap(err, "marshal envelope")
	}

	k := fmt.Sprintf("%s/%s/%s-%s", ns, project, res.Type, res.Name)

	if err := kv.Backend.Put(ctx, k, j); err != nil {
		return errors.Wrap(err, "store")
	}

	return nil
}

// Delete deletes a single resource.
func (kv *KV) Delete(ctx context.Context, ns, project, typename, name string) error {
	k := fmt.Sprintf("%s/%s/%s-%s", ns, project, typename, name)
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
		var env resEnvelope
		if err := json.Unmarshal(v, &env); err != nil {
			return nil, errors.Wrap(err, "unmarshal stored resource")
		}
		def, err := kv.Registry.New(env.Type)
		if err != nil {
			return nil, errors.Wrap(err, "create resource definition")
		}
		if err := json.Unmarshal(env.Data, def); err != nil {
			return nil, errors.Wrap(err, "unmarshal resource definition")
		}
		res := resource.Resource{
			Name:    env.Name,
			Type:    env.Type,
			Def:     def,
			Deps:    env.Deps,
			Sources: env.Sources,
		}
		ret = append(ret, res)
	}
	return ret, nil
}

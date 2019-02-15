package kvbackend

import (
	"context"
	"strings"
	"sync"

	"github.com/func/func/storage"
)

// Memory stores key-value pairs in memory.
//
// Because data is not persisted anywhere, Memory store should only be used in
// tests.
type Memory struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// Put creates or updates a value.
func (m *Memory) Put(ctx context.Context, key string, value []byte) error {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = value
	m.mu.Unlock()
	return nil
}

// Get returns a single value.
func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return v, nil
}

// Delete deletes a key.
func (m *Memory) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; !ok {
		return storage.ErrNotFound
	}
	delete(m.data, key)
	return nil
}

// Scan performs a prefix scan and populates the returned map with any values
// matching the prefix.
func (m *Memory) Scan(ctx context.Context, prefix string) (map[string][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string][]byte)
	for k, v := range m.data {
		if strings.HasPrefix(k, prefix) {
			out[k] = v
		}
	}
	return out, nil
}

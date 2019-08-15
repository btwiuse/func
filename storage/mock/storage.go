package mock

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/func/func/resource"
	"github.com/func/func/resource/graph"
)

// Storage stores data in memory for tests.
type Storage struct {
	mu        sync.Mutex
	resources map[string]resource.Resource
	graphs    map[string]graph.Graph
	Events    []Event
}

// An Event describes some operation that was done.
type Event struct {
	Op   string // noop / create / update / delete
	NS   string
	Proj string
	Res  resource.Resource
}

// resourceKey computes the key to use for storage.
func resourceKey(ns, project, name string) string {
	return fmt.Sprintf("%s/%s/%s", ns, project, name)
}

func graphKey(ns, project string) string {
	return fmt.Sprintf("%s/%s", ns, project)
}

// Seed seeds the storage for tests with existing data. Seed can be ran
// multiple times for adding resources to multiple namespaces or projects.
func (s *Storage) Seed(ns, project string, resources []resource.Resource) {
	if s.resources == nil {
		s.resources = make(map[string]resource.Resource)
	}
	for _, res := range resources {
		k := resourceKey(ns, project, res.Name)
		s.resources[k] = res
	}
}

// PutResource creates or updates a resource.
func (s *Storage) PutResource(ctx context.Context, ns, project string, res resource.Resource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resources == nil {
		s.resources = make(map[string]resource.Resource)
	}
	k := resourceKey(ns, project, res.Name)
	op := "create"
	if _, ok := s.resources[k]; ok {
		op = "update"
	}
	s.resources[k] = res
	s.Events = append(s.Events, Event{Op: op, NS: ns, Proj: project, Res: res})
	return nil
}

// DeleteResource deletes a resource. No-op if the resource does not exist.
func (s *Storage) DeleteResource(ctx context.Context, namespace, project, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := resourceKey(namespace, project, name)
	delete(s.resources, k)
	s.Events = append(s.Events, Event{Op: "delete", NS: namespace, Proj: project, Res: resource.Resource{Name: name}})
	return nil
}

// ListResources lists all resources for a project.
func (s *Storage) ListResources(ctx context.Context, namespace, project string) (map[string]resource.Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]resource.Resource)
	prefix := resourceKey(namespace, project, "")
	for k, res := range s.resources {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		out[res.Name] = res
	}
	return out, nil
}

// PutGraph creates or updates a resource.
func (s *Storage) PutGraph(ctx context.Context, ns, project string, g *graph.Graph) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.graphs == nil {
		s.graphs = make(map[string]graph.Graph)
	}
	k := graphKey(ns, project)
	op := "create-graph"
	if _, ok := s.graphs[k]; ok {
		op = "update-graph"
	}
	s.graphs[k] = *g
	s.Events = append(s.Events, Event{Op: op, NS: ns, Proj: project})
	return nil
}

// GetGraph deletes a resource. No-op if the resource does not exist.
func (s *Storage) GetGraph(ctx context.Context, namespace, project string) (*graph.Graph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := graphKey(namespace, project)
	g, ok := s.graphs[k]
	if !ok {
		return nil, nil
	}
	return &g, nil
}

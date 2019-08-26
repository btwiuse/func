package teststore

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/func/func/resource"
)

// Store is a store that's intended to be used in tests. All data is stored in memory.
type Store struct {
	mu        sync.RWMutex
	resources map[string]map[string]*resource.Resource
	graphs    map[string]*resource.Graph
}

// SeedResources seeds the store with resources for a given project. If the
// project already has resources, resources are added to it.
//
// The method may be called multiple times to add resources in parts, or add
// resources to different projects.
func (s *Store) SeedResources(project string, resources []*resource.Resource) {
	if len(resources) == 0 {
		return
	}
	if s.resources == nil {
		s.resources = make(map[string]map[string]*resource.Resource)
	}
	if s.resources[project] == nil {
		s.resources[project] = make(map[string]*resource.Resource)
	}
	for _, res := range resources {
		s.resources[project][res.Name] = res
	}
}

// SeedGraph seeds the store with the graph for a given project. If the project
// already has a graph, it is overwritten.
//
// The method may be called multiple times to set the graph for multiple
// projects.
func (s *Store) SeedGraph(project string, g *resource.Graph) {
	if s.graphs == nil {
		s.graphs = make(map[string]*resource.Graph)
	}
	s.graphs[project] = g
}

// PutResource creates or updates a resource.
func (s *Store) PutResource(ctx context.Context, project string, res *resource.Resource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resources == nil {
		s.resources = make(map[string]map[string]*resource.Resource)
	}
	if s.resources[project] == nil {
		s.resources[project] = make(map[string]*resource.Resource)
	}
	s.resources[project][res.Name] = res
	return nil
}

// DeleteResource deletes a resource. No-op if the resource does not exist.
func (s *Store) DeleteResource(ctx context.Context, project string, res *resource.Resource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.resources[project][res.Name]; !ok {
		return fmt.Errorf("resource %q does not exist in project %q", res.Name, project)
	}
	delete(s.resources[project], res.Name)
	return nil
}

// ListResources lists all resources in a project.
func (s *Store) ListResources(ctx context.Context, project string) ([]*resource.Resource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rr := s.resources[project]
	out := make([]*resource.Resource, 0, len(rr))
	for _, r := range rr {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// PutGraph creates or updates a graph.
func (s *Store) PutGraph(ctx context.Context, project string, g *resource.Graph) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.graphs == nil {
		s.graphs = make(map[string]*resource.Graph)
	}
	s.graphs[project] = g
	return nil
}

// GetGraph returns a graph for a project. Returns nil if the project does not
// have a graph.
func (s *Store) GetGraph(ctx context.Context, project string) (*resource.Graph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.graphs[project]
	if !ok {
		return nil, nil
	}
	return g, nil
}

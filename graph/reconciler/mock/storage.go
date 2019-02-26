package mock

import (
	"context"
	"errors"
	"sync"

	"github.com/func/func/resource"
)

// Resource is a resource that is part of the test store.
type Resource struct {
	NS   string
	Proj string
	Res  resource.Resource
}

// An Event describes some operation that was done.
type Event struct {
	Op   string // create / update / delete
	NS   string
	Proj string
	Res  resource.Resource
}

// Store is a Key-Value store.
type Store struct {
	mu        sync.Mutex
	Resources []Resource
	Events    []Event
}

func (s *Store) resourceIndex(ns, project, typename, id string) int {
	for i, r := range s.Resources {
		if r.NS == ns && r.Proj == project && r.Res.Def.Type() == typename && r.Res.Name == id {
			return i
		}
	}
	return -1
}

// Put stores a resource for a namespace-project.
func (s *Store) Put(ctx context.Context, ns, project string, res resource.Resource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := Resource{NS: ns, Proj: project, Res: res}
	op := "create"
	idx := s.resourceIndex(ns, project, res.Def.Type(), res.Name)
	if idx >= 0 {
		op = "update"
		s.Resources[idx] = r
	} else {
		s.Resources = append(s.Resources, r)
	}

	s.Events = append(s.Events, Event{
		Op:   op,
		NS:   ns,
		Proj: project,
		Res:  res,
	})

	return nil
}

// Delete deletes a single resource.
func (s *Store) Delete(ctx context.Context, ns, project, typename, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.resourceIndex(ns, project, typename, name)
	if idx < 0 {
		return errors.New("not found")
	}
	s.Resources = append(s.Resources[:idx], s.Resources[idx:]...)

	s.Events = append(s.Events, Event{
		Op:   "delete",
		NS:   ns,
		Proj: project,
		Res:  resource.Resource{Name: name},
	})

	return nil
}

// List lists all resource for a given namespace-project.
func (s *Store) List(ctx context.Context, ns, project string) ([]resource.Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ret []resource.Resource
	for _, r := range s.Resources {
		if r.NS == ns && r.Proj == project {
			ret = append(ret, r.Res)
		}
	}
	return ret, nil
}

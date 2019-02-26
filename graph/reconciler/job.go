package reconciler

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// job is a reconciliation job for a single namespace-project.
type job struct {
	sem      chan int
	ns       string
	graph    *graph.Graph
	existing *existingResources
	project  config.Project
	state    StateStorage
	source   SourceStorage

	mu      sync.Mutex
	process map[*graph.Resource]chan error
}

func (j *job) CreateUpdate(ctx context.Context) error {
	var leaves []*graph.Resource
	for _, r := range j.graph.Resources() {
		if len(r.Dependents()) == 0 {
			leaves = append(leaves, r)
		}
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, l := range leaves {
		l := l
		g.Go(func() error {
			return <-j.processResource(ctx, l)
		})
	}

	return g.Wait()
}

func (j *job) waitForDeps(ctx context.Context, res *graph.Resource) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, ref := range res.Dependencies() {
		dep := ref.Source.Resource
		g.Go(func() error {
			return <-j.processResource(ctx, dep)
		})
	}
	return g.Wait()
}

func (j *job) processResource(ctx context.Context, res *graph.Resource) chan error {
	j.sem <- 1
	defer func() {
		<-j.sem
	}()

	if err := j.waitForDeps(ctx, res); err != nil {
		errc := make(chan error, 1)
		errc <- errors.Wrap(err, "process deps")
		return errc
	}

	j.mu.Lock()
	errc := j.process[res]
	if errc != nil {
		j.mu.Unlock()
		return errc
	}
	errc = make(chan error, 1)
	j.process[res] = errc
	j.mu.Unlock()

	defer func() {
		for _, ref := range res.Dependents() {
			updateRef(ref)
		}

		// Note: It is important to close _after_ updating refs as closing the
		// channel allows the flow to continue, which could cause a race
		// condition when the resource is being hashed (read) while refs being
		// updated (written).
		close(errc)
	}()

	hash := resource.Hash(res.Config.Def)

	srcs := res.Sources()
	sourceList := make([]resource.SourceCode, len(srcs))
	for i, src := range res.Sources() {
		sourceList[i] = &source{
			info:    src.Config,
			storage: j.source,
		}
		res.Config.Sources = append(res.Config.Sources, src.Config.SHA)
	}

	ex := j.existing.Find(res.Config.Def.Type(), res.Config.Name)
	updateConfig := false
	updateSource := false
	if ex != nil {
		j.existing.Keep(ex)

		if ex.hash == hash {
			// Resource did not change.
			// Set all dependent inputs from existing resource definition.
			for _, ref := range res.Dependents() {
				// Change ref source to deployed resource.
				ref.Source.Resource.Config.Def = ex.res.Def
			}
		} else {
			updateConfig = true
			// Merge existing outputs into resource
			// Inputs set on the resource are not overwritten.
			if err := mergo.Merge(res.Config.Def, ex.res.Def); err != nil {
				errc <- errors.Wrap(err, "merge existing")
				return errc
			}
		}

		opts := []cmp.Option{
			cmpopts.SortSlices(func(a, b string) bool { return a < b }),
			cmpopts.EquateEmpty(),
		}
		updateSource = !cmp.Equal(ex.res.Sources, res.Config.Sources, opts...)
	}

	if ex != nil && !updateConfig && !updateSource {
		// Nothing to do
		return errc
	}

	if ex == nil {
		req := &resource.CreateRequest{
			Auth:   tempLocalAuthProvider{},
			Source: sourceList,
		}
		if err := res.Config.Def.Create(ctx, req); err != nil {
			errc <- errors.Wrap(err, "create")
			return errc
		}
	} else {
		req := &resource.UpdateRequest{
			Auth:          tempLocalAuthProvider{},
			Source:        sourceList,
			Previous:      ex.res,
			ConfigChanged: updateConfig,
			SourceChanged: updateSource,
		}
		if err := res.Config.Def.Update(ctx, req); err != nil {
			errc <- errors.Wrap(err, "update")
			return errc
		}
	}

	refs := res.Dependencies()
	if len(refs) > 0 {
		res.Config.Deps = make([]resource.Dependency, len(refs))
		for i, ref := range refs {
			res.Config.Deps[i] = resource.Dependency{
				Type: ref.Source.Resource.Config.Def.Type(),
				Name: ref.Source.Resource.Config.Name,
			}
		}
	}

	// Use new context so a cancelled context still stores the result.
	pctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := j.state.Put(pctx, j.ns, j.project.Name, res.Config); err != nil {
		errc <- errors.Wrap(err, "store resource")
		return errc
	}

	return errc
}

func (j *job) Prune(ctx context.Context) error {
	for _, e := range j.existing.Remaining() {
		req := &resource.DeleteRequest{
			Auth: tempLocalAuthProvider{},
		}
		if err := e.res.Def.Delete(ctx, req); err != nil {
			return errors.Wrap(err, "delete")
		}
		// Use new context so a cancelled context still stores the result.
		dctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := j.state.Delete(dctx, j.ns, j.project.Name, e.res.Name); err != nil {
			return errors.Wrap(err, "delete resource")
		}
	}
	return nil
}

func updateRef(ref graph.Reference) {
	src := ref.Source.Value()
	dst := ref.Target.Value()
	srcType := src.Type()
	dstType := dst.Type()

	// Direct (or close enough; int32 -> int64 etc) match
	// string -> string
	if srcType.AssignableTo(dstType) {
		dst.Set(src)
		return
	}

	// Output Pointer to Input Value
	// *string -> string
	if src.Kind() == reflect.Ptr && src.Elem().Type() == dstType {
		dst.Set(src.Elem()) // Set value from pointer's underlying value
		return
	}

	// Output Value to Input Pointer
	// string -> *string
	if dstType.Kind() == reflect.Ptr && dstType.Elem() == srcType {
		ptr := reflect.New(dstType.Elem()) // Create new pointer
		ptr.Elem().Set(src)                // Set pointer value
		dst.Set(ptr)                       // Set destination to pointer
		return
	}

	// If the application ever reached this point, it is likely because input
	// validation was not performed correctly when the configs were parsed.
	panic(fmt.Sprintf("Cannot assign %s to %s", srcType, dstType))
}

package api

import (
	"context"
	"sync"

	"github.com/func/func/config"
	"github.com/func/func/resource/graph"
	"github.com/func/func/resource/graph/hcldecoder"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Apply applies resource changes.
//
// If any resources require source code, the response will contain source
// requests. Once the sources have been uploaded, Apply should be retried.
func (f *Func) Apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	logger := f.Logger.With(zap.String("ns", req.Namespace))
	logger.Info("Apply")

	// Resolve graph and validate resource input
	g := graph.New()
	decCtx := &hcldecoder.DecodeContext{Resources: f.Resources}
	proj, srcs, diags := hcldecoder.DecodeBody(req.Config, decCtx, g)
	if diags.HasErrors() {
		return nil, diags
	}

	if proj == nil {
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "No project set",
				Subject:  req.Config.MissingItemRange().Ptr(),
			},
		}
	}

	logger = logger.With(zap.String("project", proj.Name))
	logger.Debug("Payload decoded", zap.Int("Resources", len(g.Resources)))

	// Check missing source files
	missing, err := f.missingSource(ctx, srcs)
	if err != nil {
		return nil, errors.Wrap(err, "check for source code")
	}
	if len(missing) > 0 {
		// Request source code
		logger.Debug("Source code required", zap.Strings("keys", sources(missing).Keys()))
		sr := make([]SourceRequest, len(missing))
		for i, src := range missing {
			u, err := f.Source.NewUpload(source.UploadConfig{
				Filename:      src.Key,
				ContentMD5:    src.MD5,
				ContentLength: src.Len,
			})
			if err != nil {
				return nil, errors.Wrap(err, "request upload")
			}
			sr[i] = SourceRequest{Key: src.Key, URL: u.URL, Headers: u.Headers}
		}
		return &ApplyResponse{SourcesRequired: sr}, nil
	}

	if f.Reconciler != nil {
		id := ksuid.New().String()
		if err := f.Reconciler.Reconcile(ctx, id, req.Namespace, proj.Name, g); err != nil {
			return nil, errors.Wrap(err, "reconcile graph")
		}
	} else {
		f.Logger.Info("TODO: queue reconciliation")
	}

	return &ApplyResponse{}, nil
}

func (f *Func) missingSource(ctx context.Context, sources []*config.SourceInfo) ([]*config.SourceInfo, error) {
	var mu sync.Mutex
	var missing []*config.SourceInfo
	g, ctx := errgroup.WithContext(ctx)
	for _, src := range sources {
		src := src
		g.Go(func() error {
			ok, err := f.Source.Has(ctx, src.Key)
			if err != nil {
				return errors.Wrapf(err, "check %s", src.Key)
			}
			if !ok {
				mu.Lock()
				missing = append(missing, src)
				mu.Unlock()
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, errors.WithStack(err)
	}
	return missing, nil
}

type sources []*config.SourceInfo

func (ss sources) Keys() []string {
	list := make([]string, len(ss))
	for i, s := range ss {
		list[i] = s.Key
	}
	return list
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/func/func/api/internal/rpc"
	"github.com/func/func/config"
	"github.com/func/func/resource/graph"
	"github.com/func/func/resource/graph/hcldecoder"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// A Reconciler reconciles changes to the graph.
type Reconciler interface {
	Reconcile(ctx context.Context, id, ns, project string, graph *graph.Graph) error
}

// Storage persists resolved graphs.
type Storage interface {
	PutGraph(ctx context.Context, ns, project string, g *graph.Graph) error
}

// A Registry is used for matching resource type names to resource
// implementations.
type Registry interface {
	Type(typename string) reflect.Type
	Typenames() []string
}

// A Validator validates user input.
type Validator interface {
	Validate(input interface{}, rule string) error
}

// Server provides the serverside api implementation.
type Server struct {
	Logger    *zap.Logger
	Registry  Registry
	Source    source.Storage
	Storage   Storage
	Validator Validator

	// If set, reconciliation is done synchronously.
	Reconciler Reconciler
}

// Handler returns a http handler for handling RPC request.
func (s *Server) Handler() http.Handler {
	return rpc.NewRPCServer(s, nil)
}

// Apply applies resource changes.
//
// If any resources require source code, the response will contain source
// requests. Once the sources have been uploaded, Apply should be retried.
func (s *Server) Apply(ctx context.Context, req *rpc.ApplyRequest) (*rpc.ApplyResponse, error) {
	logger := s.Logger
	logger.Info("Apply", zap.String("ns", req.Namespace))

	if req.Namespace == "" {
		logger.Debug("Namespace not set")
		return nil, twirp.NewError(twirp.InvalidArgument, "Namespace not set")
	}

	config := &hclpack.Body{}
	if err := json.Unmarshal(req.GetConfig(), config); err != nil {
		logger.Debug("Could not parse body", zap.Error(err), zap.ByteString("config", req.Config))
		return nil, twirp.NewError(twirp.InvalidArgument, fmt.Sprintf("parse config: %v", err))
	}

	resp := &rpc.ApplyResponse{}

	// Resolve graph and validate resource input
	g := graph.New()
	dec := &hcldecoder.Decoder{
		Resources: s.Registry,
		Validator: s.Validator,
	}

	proj, srcs, diags := dec.DecodeBody(config, g)
	if diags.HasErrors() {
		logger.Info("Config contains diagnostics errors", zap.Error(diags))
		resp.Diagnostics = rpc.DiagsFromHCL(diags)
		return resp, nil
	}

	if proj == nil {
		resp.Diagnostics = append(resp.Diagnostics, &rpc.Diagnostic{
			Error:   true,
			Summary: "No project set",
			Subject: rpc.RangeFromHCL(config.MissingItemRange().Ptr()),
		})
		return resp, nil
	}

	logger = logger.With(zap.String("project", proj.Name))
	logger.Debug("Payload decoded", zap.Int("Resources", len(g.Resources)))

	// Check missing source files
	missing, err := s.missingSource(ctx, srcs)
	if err != nil {
		logger.Error("Could not check source code", zap.Error(err))
		return nil, twirp.NewError(twirp.Unavailable, "Could not check source code")
	}
	if len(missing) > 0 {
		// Request source code
		logger.Debug("Source code required", zap.Strings("keys", sources(missing).Keys()))
		sr := make([]*rpc.SourceRequest, len(missing))
		for i, src := range missing {
			u, err := s.Source.NewUpload(source.UploadConfig{
				Filename:      src.Key,
				ContentMD5:    src.MD5,
				ContentLength: src.Len,
			})
			if err != nil {
				logger.Error("Could not generate upload url", zap.Error(err))
				return nil, twirp.NewError(twirp.Unavailable, "Could not generate upload url")
			}
			sr[i] = &rpc.SourceRequest{Key: src.Key, Url: u.URL, Headers: u.Headers}
		}
		return &rpc.ApplyResponse{SourcesRequired: sr}, nil
	}

	if err := s.Storage.PutGraph(ctx, req.Namespace, proj.Name, g); err != nil {
		logger.Error("Could not store graph", zap.Error(err))
		return nil, twirp.NewError(twirp.Unavailable, "Could not store graph")
	}

	if s.Reconciler != nil {
		id := ksuid.New().String()
		if err := s.Reconciler.Reconcile(ctx, id, req.Namespace, proj.Name, g); err != nil {
			logger.Error("Reconciler error", zap.Error(err))
			return nil, twirp.NewError(twirp.Unavailable, "Reconciling resource graph failed")
		}
		return resp, nil
	}

	s.Logger.Info("TODO: queue reconciliation")

	return resp, nil
}

func (s *Server) missingSource(ctx context.Context, sources []*config.SourceInfo) ([]*config.SourceInfo, error) {
	var mu sync.Mutex
	var missing []*config.SourceInfo
	g, ctx := errgroup.WithContext(ctx)
	for _, src := range sources {
		src := src
		g.Go(func() error {
			ok, err := s.Source.Has(ctx, src.Key)
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

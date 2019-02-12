package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/func/func/api"
	"github.com/func/func/graph"
	"github.com/func/func/graph/decoder"
	"github.com/func/func/resource"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/pkg/errors"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// A GraphDecoder is used for decoding the config body to a graph.
type GraphDecoder interface {
	DecodeBody(body hcl.Body, ctx *decoder.DecodeContext, g *graph.Graph) hcl.Diagnostics
}

// A ResourceRegistry is used for matching resource type names to resource
// implementations.
type ResourceRegistry interface {
	New(typename string) (resource.Definition, error)
	SuggestType(typename string) string
}

// A Server implements the server-side business logic.
type Server struct {
	Logger    *zap.Logger
	Source    source.Storage
	Resources ResourceRegistry
}

// Apply applies resources.
func (s *Server) Apply(ctx context.Context, req *api.ApplyRequest) (*api.ApplyResponse, error) {
	logger := s.Logger.With(zap.String("ns", req.Namespace))
	logger.Info("Apply")

	// Unmarshal JSON config
	var body hclpack.Body
	if err := json.Unmarshal(req.Config, &body); err != nil {
		logger.Error("Could not unmarshal config payload", zap.Error(err))
		return nil, twirp.InvalidArgumentError("config", err.Error())
	}

	// Resolve graph and validate resource input
	g := graph.New()
	decCtx := &decoder.DecodeContext{Resources: s.Resources}
	proj, diags := decoder.DecodeBody(&body, decCtx, g)
	if diags.HasErrors() {
		logger.Info("Could not resolve graph", zap.Errors("diagnostics", diags.Errs()))
		twerr := twirp.NewError(twirp.InvalidArgument, "Could not resolve graph")
		if j, err := json.Marshal(diags); err == nil {
			twerr = twerr.WithMeta("diagnostics", string(j))
		}
		return nil, twerr
	}
	logger = logger.With(zap.String("project", proj.Name))
	logger.Debug("Payload decoded", zap.Int("Resources", len(g.Resources())))

	// Check missing source files
	missing, err := s.missingSource(ctx, g.Sources())
	if err != nil {
		logger.Error("Could not check source code availability", zap.Error(err))
		return nil, twirp.NewError(twirp.Unavailable, "could not check source code")
	}
	if len(missing) > 0 {
		// Request source code
		logger.Debug("Source code required", zap.Strings("hashes", sources(missing).Hashes()))
		sr := &api.SourceRequired{}
		for _, src := range missing {
			u, err := s.Source.NewUpload(source.UploadConfig{
				Filename:      src.SHA + src.Ext,
				ContentMD5:    src.MD5,
				ContentLength: src.Len,
			})
			if err != nil {
				logger.Error("Could not create upload url", zap.Error(err))
				return nil, twirp.NewError(twirp.Unavailable, "request upload")
			}
			sr.Uploads = append(sr.Uploads, &api.UploadRequest{Digest: src.SHA, Url: u.URL, Headers: u.Headers})
		}
		return &api.ApplyResponse{Response: &api.ApplyResponse_SourceRequest{SourceRequest: sr}}, nil
	}

	dot, err := dot.MarshalMulti(g, "Graph", "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(dot))

	return nil, twirp.NewError(twirp.Unimplemented, "unimplemented")
}

func (s *Server) missingSource(ctx context.Context, sources []*graph.Source) ([]*graph.Source, error) {
	var mu sync.Mutex
	var missing []*graph.Source
	g, ctx := errgroup.WithContext(ctx)
	for _, src := range sources {
		src := src
		g.Go(func() error {
			key := src.SHA + src.Ext
			ok, err := s.Source.Has(ctx, key)
			if err != nil {
				return errors.Wrapf(err, "check %s", key)
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

type sources []*graph.Source

func (ss sources) Hashes() []string {
	list := make([]string, len(ss))
	for i, s := range ss {
		list[i] = s.SHA
	}
	return list
}

var _ api.Func = (*Server)(nil)

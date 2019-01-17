package server

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/func/func/api"
	"github.com/func/func/config"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/pkg/errors"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// A Server implements the server-side business logic.
type Server struct {
	Logger *zap.Logger
	Source source.Storage
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

	// Decode config
	root := &config.Root{}
	diags := gohcl.DecodeBody(&body, nil, root)
	if diags.HasErrors() {
		logger.Info("Could not unmarshal config payload", zap.Errors("diagnostics", diags.Errs()))
		twerr := twirp.NewError(twirp.InvalidArgument, "could not decode body")
		if j, err := json.Marshal(diags); err == nil {
			twerr = twerr.WithMeta("diagnostics", string(j))
		}
		return nil, twerr
	}
	logger = logger.With(zap.String("project", root.Project.Name))
	logger.Debug("Payload decoded", zap.Int("Resources", len(root.Resources)))

	// Check missing source files
	missing, err := s.checkDigests(ctx, root)
	if err != nil {
		logger.Error("Could not check source code availability", zap.Error(err))
		return nil, twirp.NewError(twirp.Unavailable, "could not check source code")
	}
	if len(missing) > 0 {
		// Request source code
		logger.Debug("Source code required", zap.Strings("hashes", sourceInfos(missing).Hashes()))
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

	return nil, twirp.NewError(twirp.Unimplemented, "unimplemented")
}

func (s *Server) checkDigests(ctx context.Context, root *config.Root) ([]*config.SourceInfo, error) {
	var mu sync.Mutex
	var missing []*config.SourceInfo
	g, ctx := errgroup.WithContext(ctx)
	for _, r := range root.Resources {
		if r.Source != nil {
			src := r.Source
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
	}
	if err := g.Wait(); err != nil {
		return nil, errors.WithStack(err)
	}
	return missing, nil
}

type sourceInfos []*config.SourceInfo

func (ss sourceInfos) Hashes() []string {
	list := make([]string, len(ss))
	for i, s := range ss {
		list[i] = s.SHA
	}
	return list
}

var _ api.Func = (*Server)(nil)

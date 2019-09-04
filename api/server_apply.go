package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/func/func/auth"
	"github.com/func/func/auth/permission"
	"github.com/func/func/config"
	"github.com/func/func/resource"
	"github.com/func/func/resource/hcldecoder"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// An ApplyRequest is the request to pass to Apply().
type ApplyRequest struct {
	// Project is the project to apply changes to.
	Project string

	// Config is the configuration to apply.
	Config hcl.Body
}

// ApplyResponse is returned from applying resources.
//
// The response may contain an UploadRequest, in which case the apply is
// rejected due to missing source code. The source code should be uploaded and
// then Apply should be retried.
type ApplyResponse struct {
	// SourcesRequired is set if source code uploads are required.
	SourcesRequired []*SourceRequest
}

// An SourceRequest describes a single upload request.
type SourceRequest struct {
	// Key is the key of the source code to upload.
	// The digest maps back to a resource in the ApplyRequest config.
	Key string

	// Url is the destination URL to upload to.
	// The request should be done as a HTTP PUT.
	URL string

	// Headers include the additional headers that must be set on the upload.
	Headers map[string]string
}

// Apply applies resource changes.
//
// If any resources require source code, the response will contain source
// requests. Once the sources have been uploaded, Apply should be retried.
//
// The returned error is always of type *Error.
func (s *Server) Apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	logger := s.Logger

	user, err := auth.UserFromContext(ctx)
	if err != nil {
		logger.Debug("No user")
		return nil, &Error{Code: AuthenticationError, Message: "No authentication token"}
	}
	if err := user.CheckPermissions(permission.ProjectDeploy); err != nil {
		logger.Debug("Invalid permissions", zap.Error(err))
		return nil, &Error{Code: AuthorizationError, Message: fmt.Sprintf("Authorization: %v", err)}
	}

	logger.Info("Apply", zap.String("project", req.Project))

	if req.Project == "" {
		logger.Debug("Project not set")
		return nil, &Error{Code: ValidationError, Message: "Project not set"}
	}

	resp := &ApplyResponse{}

	// Resolve graph and validate resource input
	g := &resource.Graph{}
	dec := &hcldecoder.Decoder{
		Resources: s.Registry,
		Validator: s.Validator,
	}

	srcs, diags := dec.DecodeBody(req.Config, g)
	if diags.HasErrors() {
		logger.Info("Config contains diagnostics errors", zap.Error(diags))
		return nil, &Error{
			Code:        ValidationError,
			Message:     "Config contains diagnostics errors",
			Diagnostics: diags,
		}
	}

	logger.Debug("Payload decoded", zap.Int("Resources", len(g.Resources)))

	// Check missing source files
	missing, err := s.missingSource(ctx, srcs)
	if err != nil {
		logger.Error("Could not check source code", zap.Error(err))
		return nil, &Error{Code: Unavailable}
	}
	if len(missing) > 0 {
		// Request source code
		logger.Debug("Source code required", zap.Strings("keys", sources(missing).Keys()))
		sr := make([]*SourceRequest, len(missing))
		for i, src := range missing {
			u, err := s.Source.NewUpload(source.UploadConfig{
				Filename:      src.Key,
				ContentMD5:    src.MD5,
				ContentLength: src.Len,
			})
			if err != nil {
				logger.Error("Could not generate upload url", zap.Error(err))
				return nil, &Error{Code: Unavailable}
			}
			sr[i] = &SourceRequest{Key: src.Key, URL: u.URL, Headers: u.Headers}
		}
		return &ApplyResponse{SourcesRequired: sr}, nil
	}

	if err := s.Storage.PutGraph(ctx, req.Project, g); err != nil {
		logger.Error("Could not store graph", zap.Error(err))
		return nil, &Error{Code: Unavailable}
	}

	if s.Reconciler != nil {
		id := ksuid.New().String()
		if err := s.Reconciler.Reconcile(ctx, id, req.Project, g); err != nil {
			logger.Error("Reconciler error", zap.Error(err))
			return nil, &Error{Code: Unavailable}
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

package server

import (
	"context"
	"encoding/json"

	"github.com/func/func/api"
	"github.com/func/func/config"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
)

// A Server implements the server-side business logic.
type Server struct {
	Logger *zap.Logger
}

// Apply applies resources.
func (s *Server) Apply(ctx context.Context, req *api.ApplyRequest) (*api.ApplyResponse, error) {
	logger := s.Logger.With(zap.String("ns", req.Namespace))
	logger.Info("Apply")

	var body hclpack.Body
	if err := json.Unmarshal(req.Config, &body); err != nil {
		logger.Error("Could not unmarshal config payload", zap.Error(err))
		return nil, twirp.InvalidArgumentError("config", err.Error())
	}

	var root config.Root
	diags := gohcl.DecodeBody(&body, nil, &root)
	if diags.HasErrors() {
		logger.Info("Could not unmarshal config payload", zap.Errors("diagnostics", diags.Errs()))
		twerr := twirp.NewError(twirp.InvalidArgument, "could not decode body")
		if j, err := json.Marshal(diags); err == nil {
			twerr = twerr.WithMeta("diagnostics", string(j))
		}
		return nil, twerr
	}

	logger.Debug(
		"Payload decoded",
		zap.String("Project", root.Project.Name),
		zap.Int("Resources", len(root.Resources)),
	)

	return nil, twirp.NewError(twirp.Unimplemented, "unimplemented")
}

var _ api.Func = (*Server)(nil)

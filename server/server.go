package server

import (
	"context"

	"github.com/func/func/api"
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

	return nil, twirp.NewError(twirp.Unimplemented, "unimplemented")
}

var _ api.Func = (*Server)(nil)

package rpc

import (
	"context"
	json "encoding/json"
	http "net/http"

	"github.com/func/func/api"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/pkg/errors"
	twirp "github.com/twitchtv/twirp"
	"go.uber.org/zap"
)

// NewHandler creates a new RPC handler.
// The handler will handle all func RPC traffic.
func NewHandler(logger *zap.Logger, api api.API) http.Handler {
	h := &handler{
		logger: logger,
		api:    api,
	}
	return NewFuncServer(h, nil)
}

// handler implements a HTTP handler for handling RPC responses.
type handler struct {
	logger *zap.Logger
	api    api.API
}

func (h *handler) Apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	// Unmarshal JSON config
	var body hclpack.Body
	if err := json.Unmarshal(req.GetConfig(), &body); err != nil {
		return nil, twirp.InvalidArgumentError("config", err.Error())
	}

	resp, err := h.api.Apply(ctx, &api.ApplyRequest{
		Namespace: req.GetNamespace(),
		Config:    &body,
	})
	if err != nil {
		// If error was caused by diagnostics, return invalid argument error
		// with diagnostics.
		if diags, ok := errors.Cause(err).(hcl.Diagnostics); ok {
			// Log diagnostics error.
			h.logger.Info("Apply diagnostics error", zap.Any("diagnostics", diags))

			j, err := json.Marshal(diags)
			if err != nil {
				h.logger.Error("Error marshalling diagnostics", zap.Error(err))
				return nil, twirp.NewError(twirp.Unavailable, "Configuration error could not be marshalled")
			}
			return nil, twirp.NewError(twirp.InvalidArgument, "Configuration contains errors").
				WithMeta("diagnostics", string(j))
		}

		// Log error. The original error is not returned to client as-is.
		h.logger.Error("Apply error", zap.Error(err))

		// Return generic twirp error
		return nil, twirp.NewError(twirp.Unavailable, "Could not apply changes")
	}

	rpcResp := &ApplyResponse{
		SourcesRequired: make([]*SourceRequest, len(resp.SourcesRequired)),
	}

	// Convert source requests.
	for i, sr := range resp.SourcesRequired {
		rpcResp.SourcesRequired[i] = &SourceRequest{
			Key:     sr.Key,
			Url:     sr.URL,
			Headers: sr.Headers,
		}
	}

	return rpcResp, nil
}

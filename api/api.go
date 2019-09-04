package api

import "context"

// API is the common interface for the target func api.
type API interface {
	Apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error)
}

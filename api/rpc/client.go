package rpc

import (
	"context"
	json "encoding/json"
	http "net/http"

	"github.com/func/func/api"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/pkg/errors"
	twirp "github.com/twitchtv/twirp"
)

// Client is an RPC client.
//
// The client implements the api.API interface, meaning it can be used in
// between to perform API calls on a remote server.
//
// The client should be created with NewClient().
type Client struct {
	cli Func
}

// NewClient creates a new RPC client.
//
// If httpClient is nil, http.DefaultClient is used.
func NewClient(address string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{cli: NewFuncProtobufClient(address, httpClient)}
}

// Apply marshals the request and sends it.
func (c *Client) Apply(ctx context.Context, req *api.ApplyRequest) (*api.ApplyResponse, error) {
	// Marshal body.
	j, err := json.Marshal(req.Config)
	if err != nil {
		return nil, errors.Wrap(err, "marshal config body")
	}

	// Send request.
	rpcResp, err := c.cli.Apply(ctx, &ApplyRequest{Namespace: req.Namespace, Config: j})
	if err != nil {
		if twerr, ok := err.(twirp.Error); ok {
			if diagJSON := twerr.Meta("diagnostics"); diagJSON != "" {
				var diags hcl.Diagnostics
				if derr := json.Unmarshal([]byte(diagJSON), &diags); derr == nil {
					return nil, diags
				}
			}
			return nil, errors.Errorf("%s: %s", twerr.Code(), twerr.Msg())
		}
		return nil, err
	}

	// Convert response.
	resp := &api.ApplyResponse{
		SourcesRequired: make([]api.SourceRequest, len(rpcResp.GetSourcesRequired())),
	}

	for i, sr := range rpcResp.GetSourcesRequired() {
		resp.SourcesRequired[i] = api.SourceRequest{
			Key:     sr.GetKey(),
			URL:     sr.GetUrl(),
			Headers: sr.GetHeaders(),
		}
	}

	return resp, nil
}

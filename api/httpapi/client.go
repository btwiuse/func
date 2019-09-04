package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/func/func/api"
	"github.com/hashicorp/hcl2/hclpack"
)

// ClientMiddleware is a middleware function to execute on an outgoing request.
//
// If the function returns an error, the outgoing request is not sent.
type ClientMiddleware func(r *http.Request) error

// HTTPClient is the client to use for communication.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// A Client marshals requests to send over the wire to a remote http server.
type Client struct {
	Endpoint   string
	HTTPClient HTTPClient

	middleware []ClientMiddleware
}

// AddMiddleware adds a middleware function to execute on every outgoing
// request.
func (c *Client) AddMiddleware(middleware ClientMiddleware) {
	c.middleware = append(c.middleware, middleware)
}

func (c *Client) httpClient() HTTPClient {
	cli := c.HTTPClient
	if cli == nil {
		cli = http.DefaultClient
	}
	return cli
}

// Apply marshals an ApplyRequest and sends it over the wire.
//
// The type of req.Config must be *hclpack.Body.
//
// If middleware is passed in, all middleware is executed against the outgoing
// request after any global middleware has been processed.
func (c *Client) Apply(ctx context.Context, req *api.ApplyRequest) (*api.ApplyResponse, error) { // nolint: lll
	if req.Project == "" {
		return nil, fmt.Errorf("project not set")
	}

	cfg, ok := req.Config.(*hclpack.Body)
	if !ok {
		return nil, fmt.Errorf("body must be *hclpack.Body")
	}

	r := applyRequest{
		Project: req.Project,
		Config:  cfg,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(r); err != nil {
		return nil, fmt.Errorf("encode request: %v", err)
	}
	httpreq, err := http.NewRequest(http.MethodPost, c.Endpoint+"apply", &buf)
	if err != nil {
		return nil, fmt.Errorf("build request: %v", err)
	}
	httpreq.Header.Add("Content-Type", "application/json")

	if err := c.applyMiddleware(httpreq); err != nil {
		return nil, err
	}

	cli := c.httpClient()
	resp, err := cli.Do(httpreq)
	if err != nil {
		return nil, fmt.Errorf("send request: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %v", err)
	}
	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusBadRequest:
		var response applyResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("decode error: %v", err)
		}
		if len(response.Diagnostics) > 0 {
			return nil, diagsToHCL(response.Diagnostics)
		}
		apiresp := &api.ApplyResponse{}
		if len(response.SourcesRequired) > 0 {
			apiresp.SourcesRequired = make([]*api.SourceRequest, len(response.SourcesRequired))
			for i, s := range response.SourcesRequired {
				apiresp.SourcesRequired[i] = &api.SourceRequest{
					Key:     s.Key,
					URL:     s.URL,
					Headers: s.Headers,
				}
			}
		}
		return apiresp, nil
	default:
		var errresp Error
		if err := json.Unmarshal(body, &errresp); err != nil {
			return nil, fmt.Errorf(resp.Status)
		}
		if errresp.Msg == "" {
			return nil, fmt.Errorf(resp.Status)
		}
		return nil, fmt.Errorf(errresp.Msg)
	}
}

func (c *Client) applyMiddleware(r *http.Request) error {
	for _, apply := range c.middleware {
		if err := apply(r); err != nil {
			return fmt.Errorf("global middleware: %v", err)
		}
	}
	return nil
}

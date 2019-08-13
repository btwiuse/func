package rpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/func/func/api"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	twirp "github.com/twitchtv/twirp"
)

func TestClientApply_Request(t *testing.T) {
	mock := &mockRPC{
		apply: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
			if req.GetNamespace() != "ns" {
				t.Errorf("Namespace = %s, want = %s", req.GetNamespace(), "ns")
			}

			var body hclpack.Body
			if err := json.Unmarshal(req.GetConfig(), &body); err != nil {
				t.Errorf("Parse config: %v", err)
			}
			return &ApplyResponse{}, nil
		},
	}

	req := &api.ApplyRequest{
		Namespace: "ns",
		Config:    &hclpack.Body{},
	}

	cli := &Client{cli: mock}
	_, err := cli.Apply(context.Background(), req)
	if err != nil {
		t.Fatalf("Apply() err = %v", err)
	}
}

func TestClientApply_Response(t *testing.T) {
	tests := []struct {
		name     string
		rpcResp  *ApplyResponse
		wantResp *api.ApplyResponse
	}{
		{
			name:     "Empty",
			rpcResp:  &ApplyResponse{},
			wantResp: &api.ApplyResponse{},
		},
		{
			name: "Source",
			rpcResp: &ApplyResponse{
				SourcesRequired: []*SourceRequest{
					{
						Key:     "abc",
						Url:     "https://abc.com",
						Headers: map[string]string{"foo": "bar"},
					},
				},
			},
			wantResp: &api.ApplyResponse{
				SourcesRequired: []api.SourceRequest{
					{
						Key:     "abc",
						URL:     "https://abc.com",
						Headers: map[string]string{"foo": "bar"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRPC{
				apply: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
					return tt.rpcResp, nil
				},
			}

			req := &api.ApplyRequest{
				Namespace: "ns",
				Config:    &hclpack.Body{},
			}

			cli := &Client{cli: mock}
			resp, err := cli.Apply(context.Background(), req)
			if err != nil {
				t.Fatalf("Apply() err = %v", err)
			}

			if diff := cmp.Diff(resp, tt.wantResp, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Apply() (-got, +want)\n%s", diff)
			}
		})
	}
}

func TestClientApply_ResponseErrRetry(t *testing.T) {
	tests := []struct {
		status twirp.ErrorCode
		retry  bool
	}{
		{twirp.Canceled, false},
		{twirp.Unknown, false},
		{twirp.InvalidArgument, false},
		{twirp.Malformed, false},
		{twirp.DeadlineExceeded, true},
		{twirp.NotFound, false},
		{twirp.BadRoute, false},
		{twirp.AlreadyExists, false},
		{twirp.PermissionDenied, false},
		{twirp.Unauthenticated, false},
		{twirp.ResourceExhausted, false},
		{twirp.FailedPrecondition, true},
		{twirp.Aborted, true},
		{twirp.OutOfRange, false},
		{twirp.Unimplemented, false},
		{twirp.Internal, true},
		{twirp.Unavailable, true},
		{twirp.DataLoss, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			mock := &mockRPC{
				apply: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
					return nil, twirp.NewError(tt.status, "err")
				},
			}

			req := &api.ApplyRequest{Namespace: "ns", Config: &hclpack.Body{}}
			cli := &Client{cli: mock}
			_, err := cli.Apply(context.Background(), req)
			if err == nil {
				t.Fatalf("Apply() err is nil")
			}
			rerr, got := err.(api.RetryableError)
			if got != tt.retry {
				t.Errorf("Apply() got retryable = %t, want = %t", got, tt.retry)
			}
			if got {
				if rerr.CanRetry() != tt.retry {
					t.Errorf("Apply() retryable error reports retry = %t", rerr.CanRetry())
				}
			}
		})
	}
}

func TestClientApply_ResponseErrDiagnostics(t *testing.T) {
	diags := hcl.Diagnostics{
		{Severity: hcl.DiagError, Summary: "example diagnostics"},
	}

	mock := &mockRPC{
		apply: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
			twerr := twirp.
				NewError(twirp.InvalidArgument, "example error").
				WithMeta("diagnostics", marshalDiagnostics(t, diags))
			return nil, twerr
		},
	}

	req := &api.ApplyRequest{Namespace: "ns", Config: &hclpack.Body{}}

	cli := &Client{cli: mock}
	_, err := cli.Apply(context.Background(), req)
	derr, ok := err.(api.DiagnosticsError)
	if !ok {
		t.Fatalf("Apply() err is not a DiagnosticsError")
	}
	if diff := cmp.Diff(derr.Diagnostics(), diags); diff != "" {
		t.Errorf("Apply() diagnostics (-got +want)\n%s", diff)
	}

	rerr, got := err.(api.RetryableError)
	if got && rerr.CanRetry() {
		t.Errorf("Apply() diagnostics error is retryable")
	}
}

type mockRPC struct {
	apply func(context.Context, *ApplyRequest) (*ApplyResponse, error)
}

func (m *mockRPC) Apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	return m.apply(ctx, req)
}

func marshalDiagnostics(t *testing.T, v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

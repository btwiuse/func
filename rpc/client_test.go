package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/func/func/core"
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

	req := &core.ApplyRequest{
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
		rpcErr   error
		wantResp *core.ApplyResponse
		wantErr  error
	}{
		{
			name:     "Empty",
			rpcResp:  &ApplyResponse{},
			wantResp: &core.ApplyResponse{},
		},
		{
			name: "Source",
			rpcResp: &ApplyResponse{
				SourcesRequired: []*SourceRequest{
					{
						Digest:  "abc",
						Url:     "https://abc.com",
						Headers: map[string]string{"foo": "bar"},
					},
				},
			},
			wantResp: &core.ApplyResponse{
				SourcesRequired: []core.SourceRequest{
					{
						Digest:  "abc",
						URL:     "https://abc.com",
						Headers: map[string]string{"foo": "bar"},
					},
				},
			},
		},
		{
			name:    "Error",
			rpcErr:  twirp.NewError(twirp.Unavailable, "example error"),
			wantErr: errors.New("unavailable: example error"),
		},
		{
			name: "Diagnostics",
			rpcErr: twirp.NewError(twirp.Unavailable, "example error").
				WithMeta("diagnostics", marshalDiagnostics(t, hcl.Diagnostics{
					{Severity: hcl.DiagError, Summary: "example diagnostics"},
				})),
			wantErr: hcl.Diagnostics{
				{Severity: hcl.DiagError, Summary: "example diagnostics"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRPC{
				apply: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
					return tt.rpcResp, tt.rpcErr
				},
			}

			req := &core.ApplyRequest{
				Namespace: "ns",
				Config:    &hclpack.Body{},
			}

			cli := &Client{cli: mock}
			resp, err := cli.Apply(context.Background(), req)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Apply() got <nil> error, want = %v", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("Apply() error\nGot  %v\nWant %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Apply() err = %v", err)
			}

			if diff := cmp.Diff(resp, tt.wantResp, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Apply() (-got, +want)\n%s", diff)
			}
		})
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

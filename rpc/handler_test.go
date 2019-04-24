package rpc

import (
	"context"
	"errors"
	"testing"

	"github.com/func/func/core"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	twirp "github.com/twitchtv/twirp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestHandler_Apply_Request(t *testing.T) {
	body := &hclpack.Body{}
	config, err := body.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	req := &ApplyRequest{
		Namespace: "ns",
		Config:    config,
	}

	h := &handler{
		logger: zaptest.NewLogger(t),
		api: &mockAPI{
			apply: func(ctx context.Context, req *core.ApplyRequest) (*core.ApplyResponse, error) {
				if req.Namespace != "ns" {
					t.Errorf("Namespace = %q, want = %q", req.Namespace, "ns")
				}
				if req.Config == nil {
					t.Error("Config is nil")
				}
				return &core.ApplyResponse{}, nil
			},
		},
	}

	_, err = h.Apply(context.Background(), req)
	if err != nil {
		t.Errorf("Apply() err = %v", err)
	}
}

func TestHandler_Apply_Response(t *testing.T) {
	tests := []struct {
		name            string
		coreResp        *core.ApplyResponse
		coreErr         error
		wantResp        *ApplyResponse
		wantErr         error
		wantDiagnostics bool
		wantLogs        []observer.LoggedEntry
	}{
		{
			name:     "Empty",
			coreResp: &core.ApplyResponse{},
			wantResp: &ApplyResponse{},
		},
		{
			name: "Source",
			coreResp: &core.ApplyResponse{
				SourcesRequired: []core.SourceRequest{
					{
						Digest:  "abc",
						URL:     "https://abc.com",
						Headers: map[string]string{"foo": "bar"},
					},
				},
			},
			wantResp: &ApplyResponse{
				SourcesRequired: []*SourceRequest{
					{
						Digest:  "abc",
						Url:     "https://abc.com",
						Headers: map[string]string{"foo": "bar"},
					},
				},
			},
		},
		{
			name:    "Error",
			coreErr: errors.New("some internal error"),
			wantErr: twirp.NewError(twirp.Unavailable, "Could not apply changes"), // Actual error is hidden
			wantLogs: []observer.LoggedEntry{
				{
					Entry: zapcore.Entry{Level: zapcore.ErrorLevel, Message: "Apply error"},
					Context: []zapcore.Field{
						{Key: "error", Type: zapcore.ErrorType, Interface: errors.New("some internal error")},
					},
				},
			},
		},
		{
			name: "Diagnostics",
			coreErr: hcl.Diagnostics{
				{Severity: hcl.DiagError, Summary: "summary", Detail: "detail"},
			},
			wantErr:         twirp.NewError(twirp.InvalidArgument, "Configuration contains errors"),
			wantDiagnostics: true,
			wantLogs: []observer.LoggedEntry{
				{
					Entry: zapcore.Entry{Level: zapcore.InfoLevel, Message: "Apply diagnostics error"},
					Context: []zapcore.Field{
						{Key: "diagnostics", Type: zapcore.ErrorType, Interface: hcl.Diagnostics{
							{Severity: hcl.DiagError, Summary: "summary", Detail: "detail"},
						}},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs, logs := observer.New(zap.InfoLevel)
			h := &handler{
				logger: zap.New(obs),
				api: &mockAPI{
					apply: func(ctx context.Context, req *core.ApplyRequest) (*core.ApplyResponse, error) {
						return tt.coreResp, tt.coreErr
					},
				},
			}

			body := &hclpack.Body{}
			config, err := body.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}

			req := &ApplyRequest{
				Namespace: "ns",
				Config:    config,
			}

			resp, err := h.Apply(context.Background(), req)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Apply() got <nil> error, want = %v", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("Apply() error\nGot  %v\nWant %v", err, tt.wantErr)
				}

				if tt.wantDiagnostics {
					twerr, ok := err.(twirp.Error)
					if !ok {
						t.Fatalf("error is not twirp.Error")
					}

					if d := twerr.Meta("diagnostics"); d == "" {
						t.Errorf("Diagnostics not set as meta field on error")
					}
				}

				opts := []cmp.Option{
					cmp.Comparer(func(x, y error) bool { return x.Error() == y.Error() }),
				}
				if diff := cmp.Diff(logs.AllUntimed(), tt.wantLogs, opts...); diff != "" {
					t.Errorf("Logs do not match (-got, +want)\n%s", diff)
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

type mockAPI struct {
	apply func(context.Context, *core.ApplyRequest) (*core.ApplyResponse, error)
}

func (m *mockAPI) Apply(ctx context.Context, req *core.ApplyRequest) (*core.ApplyResponse, error) {
	return m.apply(ctx, req)
}

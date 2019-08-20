package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/func/func/api/internal/rpc"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	"go.uber.org/zap/zaptest"
)

func TestClient_Apply_request(t *testing.T) {
	ns := "ns"

	config := []byte(`
		resource "foo" {
			bar = "123"
		}
	`)
	body, err := hclpack.PackNativeFile(config, "test.hcl", hcl.InitialPos)
	if err != nil {
		t.Fatal(err)
	}

	mock := mockRPC{
		apply: func(ctx context.Context, req *rpc.ApplyRequest) (*rpc.ApplyResponse, error) {
			if diff := cmp.Diff(req.Namespace, ns); diff != "" {
				t.Errorf("Namespace (-got +want)\n%s", diff)
			}

			got := &hclpack.Body{}
			if err := json.Unmarshal(req.GetConfig(), got); err != nil {
				return nil, err
			}
			if diff := cmp.Diff(got, body); diff != "" {
				t.Errorf("Body (-got +want)\n%s", diff)
			}
			return &rpc.ApplyResponse{}, nil
		},
	}
	ts := httptest.NewServer(rpc.NewRPCServer(mock, nil))
	defer ts.Close()

	cli := NewClient(ts.URL, zaptest.NewLogger(t), nil)

	if err := cli.Apply(context.Background(), ns, body); err != nil {
		t.Fatal(err)
	}
}

func TestClient_Apply_diagnostics(t *testing.T) {
	mock := mockRPC{
		apply: func(ctx context.Context, req *rpc.ApplyRequest) (*rpc.ApplyResponse, error) {
			return &rpc.ApplyResponse{
				Diagnostics: []*rpc.Diagnostic{{
					Error:   true,
					Summary: "An error",
				}},
			}, nil
		},
	}
	ts := httptest.NewServer(rpc.NewRPCServer(mock, nil))
	defer ts.Close()

	cli := NewClient(ts.URL, zaptest.NewLogger(t), nil)

	err := cli.Apply(context.Background(), "ns", &hclpack.Body{})
	if err == nil {
		t.Fatalf("Error is nil")
	}
	diags, ok := err.(hcl.Diagnostics)
	if !ok {
		t.Fatalf("Error is not hcl.Diagnostics")
	}
	wantDiags := hcl.Diagnostics{{
		Severity: hcl.DiagError,
		Summary:  "An error",
	}}
	if diff := cmp.Diff(diags, wantDiags); diff != "" {
		t.Errorf("Diagnostics do not match (-got +want)\n%s", diff)
	}
}

func TestClient_Apply_uploadSources(t *testing.T) {
	var uploads int64
	us := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that headers are correctly set
		hdr := r.Header.Get("foo")
		if hdr != "bar" {
			http.Error(w, "header foo is not bar", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		atomic.AddInt64(&uploads, 1) // Uploads happen concurrently
	}))
	defer us.Close()

	call := 0
	mock := mockRPC{
		apply: func(ctx context.Context, req *rpc.ApplyRequest) (*rpc.ApplyResponse, error) {
			n := call
			call++
			switch n {
			case 0:
				// Initial request
				// Return sources to upload
				return &rpc.ApplyResponse{
					SourcesRequired: []*rpc.SourceRequest{
						{Key: "foo", Url: us.URL + "/foo", Headers: map[string]string{"foo": "bar"}},
						{Key: "bar", Url: us.URL + "/bar", Headers: map[string]string{"foo": "bar"}},
					},
				}, nil
			case 1:
				// Retry after upload
				// Sources should be uploaded now
				if uploads != 2 {
					return nil, fmt.Errorf("files were not uploaded")
				}
				return &rpc.ApplyResponse{}, nil
			default:
				// Should not get this many apply requests:
				// 0: initial, response = sources
				// 1: next, sources present
				return nil, fmt.Errorf("too many requests")
			}
		},
	}
	ts := httptest.NewServer(rpc.NewRPCServer(mock, nil))
	defer ts.Close()

	sources := sourcemap(map[string][]byte{
		"foo": []byte("foofoo"),
		"bar": []byte("barbar"),
	})

	cli := NewClient(ts.URL, zaptest.NewLogger(t), sources)

	err := cli.Apply(context.Background(), "ns", &hclpack.Body{})
	if err != nil {
		t.Fatal(err)
	}
}

type mockRPC struct {
	apply func(context.Context, *rpc.ApplyRequest) (*rpc.ApplyResponse, error)
}

func (m mockRPC) Apply(ctx context.Context, req *rpc.ApplyRequest) (*rpc.ApplyResponse, error) {
	return m.apply(ctx, req)
}

type sourcemap map[string][]byte

func (s sourcemap) Source(sha string) *bytes.Buffer {
	return bytes.NewBuffer(s[sha])
}

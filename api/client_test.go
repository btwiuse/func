package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	"go.uber.org/zap/zaptest"
)

func TestClient_Apply_request(t *testing.T) {
	project := "testproject"

	config := []byte(`
		resource "foo" {
			bar = "123"
		}
	`)
	body, err := hclpack.PackNativeFile(config, "test.hcl", hcl.InitialPos)
	if err != nil {
		t.Fatal(err)
	}

	mock := &mockRPC{
		apply: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
			if diff := cmp.Diff(req.Project, project); diff != "" {
				t.Errorf("Project (-got +want)\n%s", diff)
			}

			if diff := cmp.Diff(req.Config, body); diff != "" {
				t.Errorf("Body (-got +want)\n%s", diff)
			}
			return &ApplyResponse{}, nil
		},
	}

	cli := &Client{
		API:    mock,
		Logger: zaptest.NewLogger(t),
		Source: nil,
	}

	req := &ApplyRequest{
		Project: project,
		Config:  body,
	}

	if err := cli.Apply(context.Background(), req); err != nil {
		t.Fatal(err)
	}
}

func TestClient_Apply_diagnostics(t *testing.T) {
	diags := hcl.Diagnostics{
		{Severity: hcl.DiagError, Summary: "An error"},
	}

	mock := &mockRPC{
		apply: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
			return nil, diags
		},
	}

	cli := &Client{
		API:    mock,
		Logger: zaptest.NewLogger(t),
		Source: nil,
	}

	req := &ApplyRequest{
		Project: "testproject",
		Config:  &hclpack.Body{},
	}

	err := cli.Apply(context.Background(), req)
	if err == nil {
		t.Fatalf("Error is nil")
	}
	got, ok := err.(hcl.Diagnostics)
	if !ok {
		t.Fatalf("Error is not hcl.Diagnostics")
	}
	if diff := cmp.Diff(got, diags); diff != "" {
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
	mock := &mockRPC{
		apply: func(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
			n := call
			call++
			switch n {
			case 0:
				// Initial request
				// Return sources to upload
				return &ApplyResponse{
					SourcesRequired: []*SourceRequest{
						{Key: "foo", URL: us.URL + "/foo", Headers: map[string]string{"foo": "bar"}},
						{Key: "bar", URL: us.URL + "/bar", Headers: map[string]string{"foo": "bar"}},
					},
				}, nil
			case 1:
				// Retry after upload
				// Sources should be uploaded now
				if uploads != 2 {
					return nil, fmt.Errorf("files were not uploaded")
				}
				return &ApplyResponse{}, nil
			default:
				// Should not get this many apply requests:
				// 0: initial, response = sources
				// 1: next, sources present
				return nil, fmt.Errorf("too many requests")
			}
		},
	}

	sources := sourcemap(map[string][]byte{
		"foo": []byte("foofoo"),
		"bar": []byte("barbar"),
	})

	cli := &Client{
		API:    mock,
		Logger: zaptest.NewLogger(t),
		Source: sources,
	}

	req := &ApplyRequest{
		Project: "testproject",
		Config:  &hclpack.Body{},
	}

	err := cli.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
}

type mockRPC struct {
	apply func(context.Context, *ApplyRequest) (*ApplyResponse, error)
}

func (m *mockRPC) Apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	return m.apply(ctx, req)
}

type sourcemap map[string][]byte

func (s sourcemap) Source(sha string) *bytes.Buffer {
	return bytes.NewBuffer(s[sha])
}

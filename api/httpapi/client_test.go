package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/func/func/api"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclpack"
)

func TestClient_Apply(t *testing.T) {
	tests := []struct {
		name    string
		req     *api.ApplyRequest
		handler func(t *testing.T) http.Handler
		want    *api.ApplyResponse
		wantErr error
	}{
		{
			name: "NoProject",
			req: &api.ApplyRequest{
				Project: "",
			},
			wantErr: fmt.Errorf("project not set"),
		},
		{
			name: "NotHCLPack",
			req: &api.ApplyRequest{
				Project: "proj",
				Config:  &hclsyntax.Body{}, // Not allowed
			},
			wantErr: fmt.Errorf("body must be *hclpack.Body"),
		},
		{
			name: "HandlerKnownError",
			req: &api.ApplyRequest{
				Project: "proj",
				Config:  &hclpack.Body{},
			},
			handler: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					respond(t, w, Error{Msg: "err"}, http.StatusInternalServerError)
				})
			},
			wantErr: fmt.Errorf("err"),
		},
		{
			name: "HandlerUnknownError",
			req: &api.ApplyRequest{
				Project: "proj",
				Config:  &hclpack.Body{},
			},
			handler: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Not json error response
					http.Error(w, "unknown error", http.StatusInternalServerError)
				})
			},
			wantErr: fmt.Errorf("500 Internal Server Error"),
		},
		{
			name: "Request",
			req: &api.ApplyRequest{
				Project: "proj",
				Config: &hclpack.Body{
					ChildBlocks: []hclpack.Block{
						{Type: "resource", Labels: []string{"lambda"}},
					},
				},
			},
			handler: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if v := r.Method; v != http.MethodPost {
						t.Errorf("Method does not match; got = %s, want = %s", v, http.MethodPost)
					}
					if v := r.Header.Get("Content-Type"); v != "application/json" {
						t.Errorf("Content-Type not match; got = %s, want = %s", v, "application/json")
					}
					if v := r.URL.Path; v != "/apply" {
						t.Errorf("Path not match; got = %s, want = %s", v, "/apply")
					}
					// Response does not matter for this test
					respond(t, w, &api.ApplyResponse{}, http.StatusOK)
				})
			},
			want: &api.ApplyResponse{},
		},
		{
			name: "Source",
			req: &api.ApplyRequest{
				Project: "proj",
				Config:  &hclpack.Body{},
			},
			handler: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					respond(t, w, applyResponse{
						SourcesRequired: []*sourceRequest{
							{Key: "b", URL: "b", Headers: map[string]string{"c": "d"}},
						},
					}, http.StatusOK)
				})
			},
			want: &api.ApplyResponse{
				SourcesRequired: []*api.SourceRequest{
					{Key: "b", URL: "b", Headers: map[string]string{"c": "d"}},
				},
			},
		},
		{
			name: "Diagnostics",
			req: &api.ApplyRequest{
				Project: "proj",
				Config:  &hclpack.Body{},
			},
			handler: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					respond(t, w, applyResponse{
						Diagnostics: []*diagnostic{
							{Error: true, Summary: "diags"},
						},
					}, http.StatusOK)
				})
			},
			wantErr: hcl.Diagnostics{
				{Severity: hcl.DiagError, Summary: "diags"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("Handler called but not defined")
			})
			if tt.handler != nil {
				handler = tt.handler(t)
			}
			ts := httptest.NewServer(handler)
			defer ts.Close()

			cli := &Client{Endpoint: ts.URL}
			got, err := cli.Apply(context.Background(), tt.req)
			if err != nil {
				opts := []cmp.Option{
					cmp.Comparer(func(a, b error) bool {
						return a.Error() == b.Error()
					}),
				}
				if diff := cmp.Diff(err, tt.wantErr, opts...); diff != "" {
					t.Errorf("Error (-got +want)\n%s", diff)
				}
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Apply() Response (-got +want)\n%s", diff)
			}
		})
	}
}

// func config(t *testing.T, source string) *hclpack.Body {
// 	body, err := hclpack.PackNativeFile([]byte(source), t.Name(), hcl.InitialPos)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	return body
// }

func respond(t *testing.T, w http.ResponseWriter, body interface{}, status int) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatal(err)
	}
}

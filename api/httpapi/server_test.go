package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/func/func/api"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
	"go.uber.org/zap/zaptest"
)

func TestServer_ServeHTTP(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	s.ServeHTTP(w, r) // Should not panic
}

func TestServer_HandleApply(t *testing.T) {
	tests := []struct {
		name  string
		req   *http.Request
		resp  *api.ApplyResponse
		err   error
		check func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "NotPost",
			req: &http.Request{
				Method: http.MethodGet,
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusMethodNotAllowed)
			},
		},
		{
			name: "NoBody",
			req: &http.Request{
				Method: http.MethodPost,
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusBadRequest)
			},
		},
		{
			name: "NotJSON",
			req: &http.Request{
				Method: http.MethodPost,
				Header: http.Header(map[string][]string{
					"Content-Type": {"text/plain"},
				}),
				Body: ioutil.NopCloser(strings.NewReader("foo")),
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusUnsupportedMediaType)
			},
		},
		{
			name: "InvalidBody",
			req: &http.Request{
				Method: http.MethodPost,
				Header: http.Header(map[string][]string{
					"Content-Type": {"application/json"},
				}),
				Body: ioutil.NopCloser(strings.NewReader("invalid")),
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusBadRequest)
			},
		},
		{
			name: "Error",
			req:  applyReq(t, applyRequest{}),
			err: &api.Error{
				Code:    api.ValidationError,
				Message: "Project not set",
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusBadRequest)
				checkBody(t, rec, &Error{Msg: "Project not set"})
			},
		},
		{
			name: "Diagnostics",
			req:  applyReq(t, applyRequest{}),
			err: &api.Error{
				Code: api.ValidationError,
				Diagnostics: hcl.Diagnostics{
					{
						Severity: hcl.DiagError,
						Summary:  "foo",
					},
				},
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusBadRequest)
				checkBody(t, rec, applyResponse{Diagnostics: []*diagnostic{{Error: true, Summary: "foo"}}})
			},
		},
		{
			name: "UnknownError",
			req:  applyReq(t, applyRequest{}),
			err:  fmt.Errorf("unknown error"),
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusInternalServerError)
				checkBody(t, rec, &Error{Msg: "Could not apply changes"})
			},
		},
		{
			name: "Source",
			req:  applyReq(t, applyRequest{}),
			resp: &api.ApplyResponse{
				SourcesRequired: []*api.SourceRequest{
					{
						Key: "key",
						URL: "url",
						Headers: map[string]string{
							"Content-MD5": "123",
						},
					},
				},
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusOK)
				checkBody(t, rec, applyResponse{
					SourcesRequired: []*sourceRequest{
						{
							Key: "key",
							URL: "url",
							Headers: map[string]string{
								"Content-MD5": "123",
							},
						},
					},
				})
			},
		},
		{
			name: "EmptyResponse",
			req:  applyReq(t, applyRequest{}),
			resp: &api.ApplyResponse{},
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				checkStatus(t, rec, http.StatusOK)
				checkBody(t, rec, applyResponse{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			s := &Server{
				API: &mockApply{
					resp: tt.resp,
					err:  tt.err,
				},
				Logger: zaptest.NewLogger(t),
			}
			s.handleApply()(rec, tt.req)
			checkContentType(t, rec, "application/json")
			tt.check(t, rec)
		})
	}
}

func applyReq(t *testing.T, req applyRequest) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatalf("encode request: %v", err)
	}
	httpreq := httptest.NewRequest(http.MethodPost, "/apply", &buf)
	httpreq.Header.Add("Content-Type", "application/json")
	return httpreq
}

func checkStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Errorf("Status does not match; got = %d, want = %d", rec.Code, want)
	}
}

func checkContentType(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	got := rec.Header().Get("Content-Type")
	if got != want {
		t.Errorf("Content-Type does not match\nGot  %q\nWant %q", got, want)
	}
}

func checkBody(t *testing.T, rec *httptest.ResponseRecorder, wantBody interface{}) {
	t.Helper()
	got := bytes.TrimSpace(rec.Body.Bytes())
	want, err := json.Marshal(wantBody)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if diff := cmp.Diff(string(got), string(want)); diff != "" {
		t.Errorf("Body does not match:\n%s", diff)
	}
}

type mockApply struct {
	resp *api.ApplyResponse
	err  error
}

func (m *mockApply) Apply(context.Context, *api.ApplyRequest) (*api.ApplyResponse, error) {
	return m.resp, m.err
}

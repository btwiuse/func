package source_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/func/func/source"
	"github.com/google/go-cmp/cmp"
)

func TestUpload(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		headers map[string]string
		servErr string
	}{
		{"Data", []byte("data"), map[string]string{"foo": "bar"}, ""},
		{"ServerError", []byte("data"), nil, "something went wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.servErr != "" {
					w.WriteHeader(http.StatusInternalServerError)
					if _, err := w.Write([]byte(tt.servErr)); err != nil {
						t.Fatal(err)
					}
					return
				}
				if r.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				for name, want := range tt.headers {
					got := r.Header.Get(name)
					if got != want {
						t.Errorf("Header %s = %s, want %s = %s", name, got, name, want)
					}
				}
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(body, tt.data); diff != "" {
					t.Errorf("Body (-got, +want)\n%s", diff)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			ctx := context.Background()
			err := source.Upload(ctx, nil, ts.URL, tt.headers, bytes.NewBuffer(tt.data))
			if tt.servErr != "" {
				if !strings.Contains(err.Error(), tt.servErr) {
					t.Errorf("Expected returned error to contain server error; got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

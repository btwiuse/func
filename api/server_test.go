package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"reflect"
	"testing"

	"github.com/func/func/api/internal/rpc"
	"github.com/func/func/resource"
	"github.com/func/func/source"
	"github.com/func/func/storage/teststore"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap/zaptest"
)

func TestServer_Apply_NoProject(t *testing.T) {
	s := &Server{
		Logger: zaptest.NewLogger(t),
	}

	req := &rpc.ApplyRequest{
		Project: "",
		Config:  []byte("{}"),
	}
	_, err := s.Apply(context.Background(), req)
	wantErr := twirp.NewError(twirp.InvalidArgument, "Project not set")
	if diff := cmp.Diff(err, wantErr, cmperropts...); diff != "" {
		t.Errorf("Error (-got +want)\n%s", diff)
	}
	t.Logf("Got expected error: %v", err)
}

func TestServer_Apply_InvalidConfig(t *testing.T) {
	s := &Server{
		Logger: zaptest.NewLogger(t),
	}

	req := &rpc.ApplyRequest{
		Project: "testproject",
		Config:  []byte("{"), // missing }
	}
	_, err := s.Apply(context.Background(), req)
	wantErr := twirp.NewError(twirp.InvalidArgument, "parse config: unexpected end of JSON input")
	if diff := cmp.Diff(err, wantErr, cmperropts...); diff != "" {
		t.Errorf("Error (-got +want)\n%s", diff)
	}
	t.Logf("Got expected error: %v", err)
}

func TestServer_Apply_Diagnostics(t *testing.T) {
	s := &Server{
		Logger:   zaptest.NewLogger(t),
		Registry: &resource.Registry{}, // Empty
	}

	req := &rpc.ApplyRequest{
		Project: "testproject",
		Config: configJSON(t, "file.hcl", `
			resource "foo" {
				type = "notsupported" # Not registered in registry
			}
		`),
	}
	resp, err := s.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Diagnostics) == 0 {
		t.Error("No diagnostics returned")
	}
}

func TestServer_Apply_RequestSource(t *testing.T) {
	src := &mockSource{
		files: map[string][]byte{},
		onUpload: func(cfg source.UploadConfig) (*source.UploadURL, error) {
			return &source.UploadURL{
				URL: "https://" + cfg.Filename,
				Headers: map[string]string{
					"Content-Length": fmt.Sprintf("%d", cfg.ContentLength),
					"Content-MD5":    cfg.ContentMD5,
				},
			}, nil
		},
	}

	s := &Server{
		Logger: zaptest.NewLogger(t),
		Registry: &resource.Registry{
			Types: map[string]reflect.Type{"bar": reflect.TypeOf(struct{}{})},
		},
		Source: src,
	}

	req := &rpc.ApplyRequest{
		Project: "testproject",
		Config: configJSON(t, "file.hcl", `
			resource "bar" {
				type   = "bar"
				source = "80:md5:sha"
			}
		`),
	}
	resp, err := s.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	got := resp.SourcesRequired
	want := []*rpc.SourceRequest{{
		Key: "sha",
		Url: "https://sha",
		Headers: map[string]string{
			"Content-Length": "128", // 0x80
			"Content-MD5":    "md5",
		},
	}}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("SourcesRequired (-got +want)\n%s", diff)
	}
}

func TestServer_Apply_OK(t *testing.T) {
	src := &mockSource{
		files: map[string][]byte{
			"foo": []byte("foo"),
		},
	}

	store := &teststore.Store{}

	s := &Server{
		Logger: zaptest.NewLogger(t),
		Registry: &resource.Registry{
			Types: map[string]reflect.Type{"bar": reflect.TypeOf(struct{}{})},
		},
		Source:  src,
		Storage: store,
	}

	req := &rpc.ApplyRequest{
		Project: "testproject",
		Config: configJSON(t, "file.hcl", `
			resource "bar" {
				type   = "bar"
				source = "80:md5:foo"
			}
		`),
	}
	_, err := s.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	g, err := store.GetGraph(context.Background(), "testproject")
	if err != nil {
		log.Fatal(err)
	}
	if g == nil {
		t.Error("Resolved graph was not stored")
	}

	// TODO: check reconciler
}

func configJSON(t *testing.T, filename, config string) []byte {
	t.Helper()
	body, diags := hclpack.PackNativeFile([]byte(config), filename, hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	j, err := body.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	return j
}

var cmperropts = []cmp.Option{
	cmp.Comparer(func(a, b twirp.Error) bool {
		return a.Code() == b.Code() &&
			a.Msg() == b.Msg() &&
			cmp.Equal(a.MetaMap(), b.MetaMap())
	}),
}

type mockSource struct {
	files    map[string][]byte
	onUpload func(cfg source.UploadConfig) (*source.UploadURL, error)
}

func (m *mockSource) Has(ctx context.Context, filename string) (bool, error) {
	_, ok := m.files[filename]
	return ok, nil
}

func (m *mockSource) Get(ctx context.Context, filename string) (io.ReadCloser, error) {
	b, ok := m.files[filename]
	if !ok {
		return nil, fmt.Errorf("source not found")
	}
	return ioutil.NopCloser(bytes.NewReader(b)), nil
}

func (m *mockSource) NewUpload(cfg source.UploadConfig) (*source.UploadURL, error) {
	return m.onUpload(cfg)
}

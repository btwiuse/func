package config_test

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"testing"

	"github.com/func/func/config"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
)

func TestLoader_Root(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		want    string
		wantErr bool
	}{
		{"Exact", "testdata/project", "testdata/project", false},
		{"Subdir", "testdata/project/src", "testdata/project", false},
		{"NoProject", os.TempDir(), "", true},
		{"NotFound", "nonexisting", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &config.Loader{}
			got, err := l.Root(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Loader.Root() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Loader.Root() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoader_Load(t *testing.T) {
	tests := []struct {
		name       string
		root       string
		compressor config.SourceCompressor
		want       *hclpack.Body
	}{
		{
			"Project",
			"testdata/project",
			&mockCompressor{
				data: []byte("targz data"),
			},
			&hclpack.Body{
				ChildBlocks: []hclpack.Block{
					{
						Type:   "resource",
						Labels: []string{"lambda"},
						Body: hclpack.Body{
							Attributes: map[string]hclpack.Attribute{
								"type": {
									Expr: hclpack.Expression{
										Source:     []byte(`"aws:lambda_function"`),
										SourceType: hclpack.ExprNative,
										Range_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 2, Column: 12, Byte: 31},
											End:      hcl.Pos{Line: 2, Column: 33, Byte: 52},
										},
										StartRange_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 2, Column: 13, Byte: 32},
											End:      hcl.Pos{Line: 2, Column: 32, Byte: 51},
										},
									},
									Range: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 2, Column: 3, Byte: 22},
										End:      hcl.Pos{Line: 2, Column: 33, Byte: 52},
									},
									NameRange: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 2, Column: 3, Byte: 22},
										End:      hcl.Pos{Line: 2, Column: 7, Byte: 26},
									},
								},
								"source": {
									Expr: hclpack.Expression{
										Source:     []byte(`"` + sourceInfoStr(t, []byte("targz data")) + `"`),
										SourceType: hclpack.ExprLiteralJSON,
										Range_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 4, Column: 12, Byte: 65},
											End:      hcl.Pos{Line: 4, Column: 19, Byte: 72},
										},
										StartRange_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 4, Column: 13, Byte: 66},
											End:      hcl.Pos{Line: 4, Column: 18, Byte: 71},
										},
									},
									Range: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 4, Column: 3, Byte: 56},
										End:      hcl.Pos{Line: 4, Column: 19, Byte: 72},
									},
									NameRange: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 4, Column: 3, Byte: 56},
										End:      hcl.Pos{Line: 4, Column: 9, Byte: 62},
									},
								},
								"handler": {
									Expr: hclpack.Expression{
										Source:     []byte(`"index.handler"`),
										SourceType: hclpack.ExprNative,
										Range_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 6, Column: 13, Byte: 86},
											End:      hcl.Pos{Line: 6, Column: 28, Byte: 101},
										},
										StartRange_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 6, Column: 14, Byte: 87},
											End:      hcl.Pos{Line: 6, Column: 27, Byte: 100},
										},
									},
									Range: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 6, Column: 3, Byte: 76},
										End:      hcl.Pos{Line: 6, Column: 28, Byte: 101},
									},
									NameRange: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 6, Column: 3, Byte: 76},
										End:      hcl.Pos{Line: 6, Column: 10, Byte: 83},
									},
								},
								"memory": {
									Expr: hclpack.Expression{
										Source:     []byte("512"),
										SourceType: hclpack.ExprNative,
										Range_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 7, Column: 13, Byte: 114},
											End:      hcl.Pos{Line: 7, Column: 16, Byte: 117},
										},
										StartRange_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 7, Column: 13, Byte: 114},
											End:      hcl.Pos{Line: 7, Column: 16, Byte: 117},
										},
									},
									Range: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 7, Column: 3, Byte: 104},
										End:      hcl.Pos{Line: 7, Column: 16, Byte: 117},
									},
									NameRange: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 7, Column: 3, Byte: 104},
										End:      hcl.Pos{Line: 7, Column: 9, Byte: 110},
									},
								},
							},
							MissingItemRange_: hcl.Range{
								Filename: "testdata/project/func.hcl",
								Start:    hcl.Pos{Line: 8, Column: 2, Byte: 119},
								End:      hcl.Pos{Line: 8, Column: 2, Byte: 119},
							},
						},
						DefRange: hcl.Range{
							Filename: "testdata/project/func.hcl",
							Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
							End:      hcl.Pos{Line: 1, Column: 18, Byte: 17},
						},
						TypeRange: hcl.Range{
							Filename: "testdata/project/func.hcl",
							Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
							End:      hcl.Pos{Line: 1, Column: 9, Byte: 8},
						},
						LabelRanges: []hcl.Range{
							{
								Filename: "testdata/project/func.hcl",
								Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
								End:      hcl.Pos{Line: 1, Column: 18, Byte: 17},
							},
						},
					},
					{
						Type:   "project",
						Labels: []string{"test"},
						Body: hclpack.Body{
							MissingItemRange_: hcl.Range{
								Filename: "testdata/project/proj.hcl",
								Start:    hcl.Pos{Line: 1, Column: 18, Byte: 17},
								End:      hcl.Pos{Line: 1, Column: 18, Byte: 17},
							},
						},
						DefRange: hcl.Range{
							Filename: "testdata/project/proj.hcl",
							Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
							End:      hcl.Pos{Line: 1, Column: 15, Byte: 14},
						},
						TypeRange: hcl.Range{
							Filename: "testdata/project/proj.hcl",
							Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
							End:      hcl.Pos{Line: 1, Column: 8, Byte: 7},
						},
						LabelRanges: []hcl.Range{
							{
								Filename: "testdata/project/proj.hcl",
								Start:    hcl.Pos{Line: 1, Column: 9, Byte: 8},
								End:      hcl.Pos{Line: 1, Column: 15, Byte: 14},
							},
						},
					},
				},
				MissingItemRange_: hcl.Range{
					Filename: "testdata/project/func.hcl",
					Start:    hcl.Pos{Line: 9, Column: 1, Byte: 120},
					End:      hcl.Pos{Line: 9, Column: 1, Byte: 120},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &config.Loader{
				Compressor: tt.compressor,
			}
			got, diags := l.Load(tt.root)
			if diags.HasErrors() {
				t.Fatalf("Loader.Load() error = %v", diags)
			}

			bytesAsString := cmp.Transformer("string", func(b []byte) string { return string(b) })
			if diff := cmp.Diff(got, tt.want, bytesAsString); diff != "" {
				t.Errorf("Load() (-got, +want)\n%s", diff)
			}
		})
	}
}

func TestLoader_Files(t *testing.T) {
	tests := []struct {
		name string
		root string
		want []string
	}{
		{
			"Project",
			"testdata/project",
			[]string{
				"testdata/project/func.hcl",
				"testdata/project/proj.hcl",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &config.Loader{
				Compressor: &mockCompressor{},
			}
			_, diags := l.Load(tt.root)
			if diags.HasErrors() {
				t.Fatalf("Load() error = %v", diags)
			}
			var got []string
			for name := range l.Files() {
				got = append(got, name)
			}
			sort.Strings(got)

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Files() (-want, +got)\n%s", diff)
			}
		})
	}
}

func TestLoader_Source(t *testing.T) {
	tests := []struct {
		name string
		root string
	}{
		{
			"Project",
			"testdata/project",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &config.Loader{
				Compressor: &mockCompressor{
					data: []byte("targz data"),
				},
			}
			got, err := l.Load(tt.root)
			if err != nil {
				t.Fatalf("Load() error = %v", err.Errs())
			}
			// Assert that every digest set returns files
			var root config.Root
			diags := gohcl.DecodeBody(got, nil, &root)
			if diags.HasErrors() {
				t.Errorf("Decode() error = %v", diags.Errs())
			}

			gotSources := 0
			for _, r := range root.Resources {
				if r.Source != "" {
					src, err := config.DecodeSourceString(r.Source)
					if err != nil {
						t.Fatalf("DecodeSourceString() err = %v", err)
					}
					if l.Source(src.Key) == nil {
						t.Errorf("Source() returned no source for %q", src.Key)
						continue
					}
					gotSources++
				}
			}

			wantSources := 1
			if gotSources != wantSources {
				t.Errorf("Sources do not match; got = %d, want = %d", gotSources, wantSources)
			}
		})
	}
}

func TestLoader_Source_notFound(t *testing.T) {
	l := &config.Loader{}
	got := l.Source("foo")
	if got != nil {
		t.Errorf("Source() got = %v, want = %v", got, nil)
	}
}

func TestLoader_jsonRoundTrip(t *testing.T) {
	// This doesn't specifically test against anything in config, it's just to
	// protect against breaking changes in the hcl library, which is very
	// critical here.

	l := &config.Loader{
		Compressor: &mockCompressor{
			data: []byte("targz data"),
		},
	}
	before, diags := l.Load("testdata/project")
	if diags.HasErrors() {
		t.Fatalf("Load() error = %v", diags)
	}

	j, err := json.Marshal(before)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	t.Logf("json %d bytes: %s", len(j), string(j))

	after := &hclpack.Body{}
	err = json.Unmarshal(j, after)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if diff := cmp.Diff(before, after); diff != "" {
		t.Fatalf("Content changed after json roundtrip\n%s", diff)
	}
}

var args = []string{"testdata/project"}

func Example_clientServer() {
	// Client

	// Create a loader
	l := &config.Loader{}

	// Find root, given user input
	rootDir, diags := l.Root(args[0])
	if diags.HasErrors() {
		log.Fatal(diags)
	}

	// Load config files from root
	cfg, diags := l.Load(rootDir)
	if diags.HasErrors() {
		log.Fatal(diags)
	}

	// Marshal config to json for transmission
	payload, err := json.Marshal(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Server

	// Parse payload
	var recv hclpack.Body
	if err := json.Unmarshal(payload, &recv); err != nil {
		log.Fatal(err)
	}

	// Decode config
	var root config.Root
	if err := gohcl.DecodeBody(&recv, nil, &root); err != nil {
		log.Fatal(err)
	}
}

type mockCompressor struct {
	data []byte
	err  error
}

func (m *mockCompressor) Compress(w io.Writer, dir string) error {
	if m.err != nil {
		return m.err
	}
	if _, err := bytes.NewBuffer(m.data).WriteTo(w); err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	return nil
}

func sourceInfoStr(t *testing.T, b []byte) string {
	md5 := md5.New()
	sha := sha256.New()
	w := io.MultiWriter(md5, sha)
	if _, err := w.Write(b); err != nil {
		t.Fatal(err)
	}
	src := config.SourceInfo{
		Len: len(b),
		MD5: base64.StdEncoding.EncodeToString(md5.Sum(nil)),
		Key: hex.EncodeToString(sha.Sum(nil)),
	}
	return src.EncodeToString()
}

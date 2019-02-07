package decoder_test

import (
	"strings"
	"testing"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/graph/decoder"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
)

func TestDecodeBody(t *testing.T) {
	tests := []struct {
		name  string
		body  hcl.Body
		ctx   *decoder.DecodeContext
		want  graph.Snapshot
		diags hcl.Diagnostics
	}{
		{
			name: "Empty",
			body: parseBody(t, ""),
			ctx:  &decoder.DecodeContext{},
			want: graph.Snapshot{},
		},
		{
			name: "Resource",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					input = "hello"
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooRes{})},
			want: graph.Snapshot{
				Projects: []config.Project{
					{Name: "test"},
				},
				Resources: []resource.Definition{
					&fooRes{Input: strptr("hello")},
				},
				ProjectResources: map[int][]int{
					0: {0},
				},
			},
		},
		{
			name: "ResourceSource",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					source ".tar.gz" {
						sha = "abc"
						md5 = "def"
						len = 123
					}
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooRes{})},
			want: graph.Snapshot{
				Projects: []config.Project{
					{Name: "test"},
				},
				Resources: []resource.Definition{
					&fooRes{},
				},
				Sources: []config.SourceInfo{
					{SHA: "abc", MD5: "def", Len: 123, Ext: ".tar.gz"},
				},
				ProjectResources: map[int][]int{
					0: {0},
				},
				ResourceSources: map[int][]int{
					0: {0},
				},
			},
		},
		{
			name: "RefInput",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					input = "hello"
				}
				resource "bar" "baz" {
					input = foo.bar.input # copy value
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooRes{}, &barRes{})},
			want: graph.Snapshot{
				Projects: []config.Project{
					{Name: "test"},
				},
				Resources: []resource.Definition{
					&fooRes{Input: strptr("hello")},
					&barRes{Input: strptr("hello")},
				},
				ProjectResources: map[int][]int{
					0: {0, 1},
				},
			},
		},
		{
			name: "RefOutput",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					input = "hello"
				}
				resource "bar" "foo" {
					input = foo.bar.output
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooRes{}, &barRes{})},
			want: graph.Snapshot{
				Projects: []config.Project{
					{Name: "test"},
				},
				Resources: []resource.Definition{
					&fooRes{Input: strptr("hello")},
					&barRes{},
				},
				ProjectResources: map[int][]int{
					0: {0, 1},
				},
				References: []graph.SnapshotRef{
					{Source: 0, Target: 1, SourceIndex: []int{1}, TargetIndex: []int{0}},
				},
			},
		},
		{
			name: "ConvertType",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					input = 3.14159 # convert to string
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooRes{})},
			want: graph.Snapshot{
				Projects: []config.Project{
					{Name: "test"},
				},
				Resources: []resource.Definition{
					&fooRes{Input: strptr("3.14159")},
				},
				ProjectResources: map[int][]int{
					0: {0},
				},
			},
		},
		{
			name: "InvalidType",
			body: parseBody(t, `
				project "test" {}
				resource "baz" "baz" {
					num = "this cannot be an int"
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&bazRes{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "Unsuitable value: a number is required",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 3, Column: 13},
					End:   hcl.Pos{Line: 3, Column: 34},
				},
				Context: &hcl.Range{
					Start: hcl.Pos{Line: 3, Column: 12},
					End:   hcl.Pos{Line: 3, Column: 35},
				},
			}},
		},
		{
			name: "ResourceNotFound",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {}
			`),
			ctx: &decoder.DecodeContext{Resources: &resource.Registry{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource not supported",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 2, Column: 14},
					End:   hcl.Pos{Line: 2, Column: 19},
				},
			}},
		},
		{
			name: "SuggestResource",
			body: parseBody(t, `
				project "test" {}
				resource "roo" "bar" {}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooRes{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource not supported",
				Detail:   "Did you mean \"foo\"?",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 2, Column: 14},
					End:   hcl.Pos{Line: 2, Column: 19},
				},
			}},
		},
		{
			name: "InvalidInputType",
			body: parseBody(t, `
				project "test" {}
				resource "bar" "a" {
					input = "hello"    # string
				}
				resource "baz" "b" {
					num = bar.a.input # int
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&barRes{}, &bazRes{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Cannot set num from string, number value is required",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 6, Column: 6},
					End:   hcl.Pos{Line: 6, Column: 23},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := graph.New()
			diags := decoder.DecodeBody(tt.body, tt.ctx, g)
			ignoreByte := cmp.Transformer("ignoreByteOffset", func(pos hcl.Pos) hcl.Pos {
				pos.Byte = 0
				return pos
			})
			if diff := cmp.Diff(diags, tt.diags, ignoreByte); diff != "" {
				t.Errorf("DecodeBody() diagnostics (-got, +want)\n%s", diff)
			}
			if tt.diags.HasErrors() {
				// Do not match snapshot if errors are expected.
				return
			}
			snap := g.Snapshot()
			if diff := snap.Diff(tt.want); diff != "" {
				t.Errorf("Snapshot does not match (-got, +want)\n%s", diff)
			}
		})
	}
}

func parseBody(t *testing.T, src string) hcl.Body {
	t.Helper()
	src = strings.TrimSpace(src)
	file, diags := hclsyntax.ParseConfig([]byte(src), "", hcl.Pos{Byte: 0, Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Errorf("Parse test body: %v", diags)
	}
	return file.Body
}

type fooRes struct {
	Input  *string `input:"input"`
	Output string  `output:"output"`
}

func (r *fooRes) Type() string { return "foo" }

type barRes struct {
	Input *string `input:"input"`
}

func (r *barRes) Type() string { return "bar" }

type bazRes struct {
	Num int `input:"num"`
}

func (r *bazRes) Type() string { return "baz" }

func strptr(str string) *string { return &str }

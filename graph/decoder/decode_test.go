package decoder_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/graph/decoder"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
)

func TestDecodeBody(t *testing.T) {
	tests := []struct {
		name  string
		body  hcl.Body
		ctx   *decoder.DecodeContext
		check func(t *testing.T, g *graph.Graph)
		diags hcl.Diagnostics
	}{
		{
			name: "Empty",
			body: parseBody(t, ""),
			ctx:  &decoder.DecodeContext{},
			check: func(t *testing.T, g *graph.Graph) {
				assertResources(t, g, nil)
			},
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
			check: func(t *testing.T, g *graph.Graph) {
				wantDef := []resource.Definition{
					&fooRes{
						Input: strptr("hello"),
					},
				}
				assertResources(t, g, wantDef)
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
			check: func(t *testing.T, g *graph.Graph) {
				rr := g.Resources()
				if len(rr) != 1 {
					t.Fatalf("len(Resources) got = %d, want = %d", len(rr), 1)
				}
				for _, r := range rr {
					got := r.Sources()
					want := []*graph.Source{{
						SourceInfo: config.SourceInfo{
							Ext: ".tar.gz",
							SHA: "abc",
							MD5: "def",
							Len: 123,
						},
					}}
					opts := []cmp.Option{
						cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".Node" }, cmp.Ignore()),
						cmpopts.IgnoreUnexported(graph.Source{}),
					}
					if diff := cmp.Diff(got, want, opts...); diff != "" {
						t.Errorf("Source (-got, +want)\n%s", diff)
					}
				}
			},
		},
		{
			name: "RefInput",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					input = "hello"
				}
				resource "foo" "baz" {
					input = foo.bar.input # copy value
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooRes{})},
			check: func(t *testing.T, g *graph.Graph) {
				wantDef := []resource.Definition{
					&fooRes{Input: strptr("hello")},
					&fooRes{Input: strptr("hello")},
				}
				assertResources(t, g, wantDef)
			},
		},
		{
			name: "RefOutput",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {}
				resource "bar" "foo" {
					input = foo.bar.output
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooRes{}, &barRes{})},
			check: func(t *testing.T, g *graph.Graph) {
				got := make(map[string][]graph.Reference)
				for _, r := range g.Resources() {
					name := fmt.Sprintf("%T", r.Definition)
					got[name] = r.Dependencies()
				}
				want := map[string][]graph.Reference{
					"*decoder_test.fooRes": {},
					"*decoder_test.barRes": {
						{
							Source: graph.Field{Resource: &graph.Resource{Definition: &fooRes{}}, Index: []int{1}},
							Target: graph.Field{Resource: &graph.Resource{Definition: &barRes{}}, Index: []int{0}},
						},
					},
				}
				opts := []cmp.Option{
					cmpopts.EquateEmpty(),
					cmp.FilterPath(func(p cmp.Path) bool { return p.Last().String() == ".Node" }, cmp.Ignore()),
					cmpopts.IgnoreUnexported(graph.Resource{}),
				}
				if diff := cmp.Diff(got, want, opts...); diff != "" {
					t.Errorf("Dependencies do not match (-got, +want)\n%s", diff)
				}
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
			check: func(t *testing.T, g *graph.Graph) {
				wantDef := []resource.Definition{
					&fooRes{Input: strptr("3.14159")},
				}
				assertResources(t, g, wantDef)
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
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&barRes{}, &bazRes{})},
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
			if tt.check != nil {
				tt.check(t, g)
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

// assertResources checks that the given resources exist in the graph.
//
// The order of resources returned from the graph does not matter.
func assertResources(t *testing.T, g *graph.Graph, want []resource.Definition) {
	t.Helper()
	rr := g.Resources()
	got := make([]resource.Definition, len(rr))
	for i, r := range rr {
		got[i] = r.Definition
	}
	if diff := cmp.Diff(got, want, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("resource.Resources (-got, +want)\n%s", diff)
	}
}

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
		name     string
		body     hcl.Body
		ctx      *decoder.DecodeContext
		wantSnap graph.Snapshot
		wantProj config.Project
		diags    hcl.Diagnostics
	}{
		{
			name: "Empty",
			body: parseBody(t, `
				project "test" {}
			`),
			ctx:      &decoder.DecodeContext{},
			wantSnap: graph.Snapshot{},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "Resource",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					input = "hello"
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{})},
			wantSnap: graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "bar", Def: &fooDef{Input: "hello"}},
				},
			},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "ResourceSource",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					input = "hello"
					source ".tar.gz" {
						sha = "abc"
						md5 = "def"
						len = 123
					}
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{})},
			wantSnap: graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "bar", Def: &fooDef{Input: "hello"}},
				},
				Sources: []config.SourceInfo{
					{SHA: "abc", MD5: "def", Len: 123, Ext: ".tar.gz"},
				},
				ResourceSources: map[int][]int{
					0: {0},
				},
			},
			wantProj: config.Project{Name: "test"},
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
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{}, &barDef{})},
			wantSnap: graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "bar", Def: &fooDef{Input: "hello"}},
					{Name: "baz", Def: &barDef{Input: strptr("hello")}},
				},
			},
			wantProj: config.Project{Name: "test"},
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
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{}, &barDef{})},
			wantSnap: graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "bar", Def: &fooDef{Input: "hello"}},
					{Name: "foo", Def: &barDef{}},
				},
				References: []graph.SnapshotRef{
					{Source: 0, Target: 1, SourceIndex: []int{2}, TargetIndex: []int{1}},
				},
			},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "ConvertType",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "bar" {
					input = 3.14159 # convert to string
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{})},
			wantSnap: graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "bar", Def: &fooDef{Input: "3.14159"}},
				},
			},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "Map",
			body: parseBody(t, `
				project "test" {}
				resource "complex" "foo" {
					map = {
						foo = "bar"
					}
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&complexDef{})},
			wantSnap: graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "foo", Def: &complexDef{
						Map: &map[string]string{"foo": "bar"},
					}},
				},
			},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "Slice",
			body: parseBody(t, `
				project "test" {}
				resource "complex" "foo" {
					slice = ["hello", "world"]
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&complexDef{})},
			wantSnap: graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "foo", Def: &complexDef{
						Slice: &[]string{"hello", "world"},
					}},
				},
			},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "StructBlock",
			body: parseBody(t, `
				project "test" {}
				resource "complex" "foo" {
					nested {
						sub {
							value = "hello"
						}
					}
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&complexDef{})},
			wantSnap: graph.Snapshot{
				Resources: []resource.Resource{
					{Name: "foo", Def: &complexDef{
						Child: &Child{
							Sub: sub{
								Val: "hello",
							},
						},
					}},
				},
			},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "MultipleBlocks",
			body: parseBody(t, `
				project "test" {}
				resource "complex" "foo" {
					nested {
						value = "hello"
					}
					nested {
						value = "hello"
					}
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&complexDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Duplicate nested block",
				Detail:   "Only one nested block is allowed. Another was defined on line 3",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 6, Column: 6},
					End:   hcl.Pos{Line: 6, Column: 12},
				},
			}},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "MissingBlock",
			body: parseBody(t, `
				project "test" {}
				resource "qux" "foo" {
					# required block not set
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&quxDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required_field block",
				Detail:   "A required_field block is required.",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 4, Column: 6},
					End:   hcl.Pos{Line: 4, Column: 6},
				},
			}},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "MissingNestedBlock",
			body: parseBody(t, `
				project "test" {}
				resource "complex" "foo" {
					nested {
					}
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&complexDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing sub block",
				Detail:   "A sub block is required.",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 4, Column: 7},
					End:   hcl.Pos{Line: 4, Column: 7},
				},
			}},
			wantProj: config.Project{Name: "test"},
		},
		{
			name: "NoProject",
			body: parseBody(t, `
				resource "foo" "bar" {
					input = "hello"
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing project block",
				Detail:   "A project block is required",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 3, Column: 6},
					End:   hcl.Pos{Line: 3, Column: 6},
				},
			}},
		},
		{
			name: "NoProjectName",
			body: parseBody(t, `
				project "" {}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Project name not set",
				Detail:   "A project name is required",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 9},
					End:   hcl.Pos{Line: 1, Column: 11},
				},
				Context: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1},
					End:   hcl.Pos{Line: 1, Column: 11},
				},
			}},
		},
		{
			name: "InvalidType",
			body: parseBody(t, `
				project "test" {}
				resource "baz" "baz" {
					num = "this cannot be an int"
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&bazDef{})},
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
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{})},
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
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&barDef{}, &bazDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Cannot set num from string, number value is required",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 6, Column: 6},
					End:   hcl.Pos{Line: 6, Column: 23},
				},
			}},
		},
		{
			name: "MissingRequiredArg",
			body: parseBody(t, `
				project "test" {}
				resource "foo" "a" {
					# input not set
				}
			`),
			ctx: &decoder.DecodeContext{Resources: resource.RegistryFromResources(&fooDef{})},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   `The argument "input" is required, but no definition was found.`,
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 4, Column: 6},
					End:   hcl.Pos{Line: 4, Column: 6},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := graph.New()
			proj, diags := decoder.DecodeBody(tt.body, tt.ctx, g)
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
			if diff := snap.Diff(tt.wantSnap); diff != "" {
				t.Errorf("Snapshot does not match (-got, +want)\n%s", diff)
			}
			if diff := cmp.Diff(proj, tt.wantProj); diff != "" {
				t.Errorf("Project does not match (-got, +want)\n%s", diff)
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

type fooDef struct {
	resource.Definition
	Input  string `input:"input"`
	Output string `output:"output"`
}

func (r *fooDef) Type() string { return "foo" }

type barDef struct {
	resource.Definition
	Input *string `input:"input"`
}

func (r *barDef) Type() string { return "bar" }

type bazDef struct {
	resource.Definition
	Num int `input:"num"`
}

func (r *bazDef) Type() string { return "baz" }

type quxDef struct {
	resource.Definition
	Required Child `input:"required_field"`
}

func (r *quxDef) Type() string { return "qux" }

type complexDef struct {
	resource.Definition

	Map   *map[string]string `input:"map"`
	Slice *[]string          `input:"slice"`
	Child *Child             `input:"nested"`
}

type Child struct {
	Sub sub `input:"sub"`
}

type sub struct {
	Val string `input:"value"`
}

func (r *complexDef) Type() string { return "complex" }

func strptr(str string) *string { return &str }

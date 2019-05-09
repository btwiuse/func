package decoder_test

import (
	"runtime/debug"
	"strings"
	"testing"

	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/graph/decoder"
	"github.com/func/func/graph/snapshot"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
)

func TestDecodeBody(t *testing.T) {
	tests := []struct {
		name      string
		body      hcl.Body
		resources []resource.Definition
		wantSnap  snapshot.Snap
		wantProj  *config.Project
	}{
		{
			name: "Project",
			body: parseBody(t, `
				project "test" {}
			`),
			wantProj: &config.Project{Name: "test"},
		},
		{
			name: "StaticInput",
			body: parseBody(t, `
				resource "simple" "bar" {
					input = "hello"
				}
				resource "simple" "baz" {
					input = "world"
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "bar", Def: &simpleDef{Input: "hello"}},
					{Name: "baz", Def: &simpleDef{Input: "world"}},
				},
			},
		},
		{
			name: "Source",
			body: parseBody(t, `
				resource "simple" "bar" {
					input  = "src"
					source = "ff:abc:def"
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "bar", Def: &simpleDef{Input: "src"}},
				},
				Sources: []config.SourceInfo{
					{Key: "def", MD5: "abc", Len: 0xFF},
				},
				ResourceSources: map[int][]int{
					0: {0},
				},
			},
		},
		{
			name: "DependencyToInput",
			body: parseBody(t, `
				resource "simple" "bar" {
					input = "hello"
				}
				resource "simple" "baz" {
					input = simple.bar.input
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "bar", Def: &simpleDef{Input: "hello"}},
					{Name: "baz", Def: &simpleDef{Input: "hello"}}, // Input can be statically resolved.
				},
			},
		},
		{
			name: "DependencyToInputExtended",
			body: parseBody(t, `
				resource "simple" "bar" {
					input = "hello"
				}
				resource "simple" "baz" {
					input = simple.bar.input
				}
				resource "simple" "qux" {
					input = simple.baz.input
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "bar", Def: &simpleDef{Input: "hello"}},
					{Name: "baz", Def: &simpleDef{Input: "hello"}},
					{Name: "qux", Def: &simpleDef{Input: "hello"}}, // Input can be statically resolved through baz.
				},
			},
		},
		{
			name: "DependencyToOutput",
			body: parseBody(t, `
				resource "simple" "bar" {
					input = "hello"
				}
				resource "simple" "baz" {
					input = simple.bar.output
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "bar", Def: &simpleDef{Input: "hello"}},
					{Name: "baz", Def: &simpleDef{}}, // Input is dynamic.
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${simple.baz.input}": "${simple.bar.output}",
				},
			},
		},
		{
			name: "DependencyExpression",
			body: parseBody(t, `
				resource "simple" "bar" {
					input = "hello"
				}
				resource "simple" "baz" {
					input = ":: ${simple.bar.input} - ${simple.bar.output} <<<"
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "bar", Def: &simpleDef{Input: "hello"}},
					{Name: "baz", Def: &simpleDef{}},
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${simple.baz.input}": ":: hello - ${simple.bar.output} <<<", // Partially resolved.
				},
			},
		},
		{
			name: "ConvertType",
			body: parseBody(t, `
				resource "simple" "bar" {
					input = 3.14
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "bar", Def: &simpleDef{Input: "3.14"}}, // Converted to string.
				},
			},
		},
		{
			name: "Map",
			body: parseBody(t, `
				resource "complex" "foo" {
					map = {
						foo = "bar"
					}
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &complexDef{
						Map: &map[string]string{"foo": "bar"},
					}},
				},
			},
		},
		{
			name: "Slice",
			body: parseBody(t, `
				resource "complex" "foo" {
					slice = ["hello", "world"]
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &complexDef{
						Slice: &[]string{"hello", "world"},
					}},
				},
			},
		},
		{
			name: "Struct",
			body: parseBody(t, `
				resource "complex" "foo" {
					nested {
						sub {
							value = "hello"
						}
					}
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			wantSnap: snapshot.Snap{
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
		},
		{
			name: "MultipleBlocksAllowed",
			body: parseBody(t, `
				resource "complex" "foo" {
					multi {
						value = "hello"
					}
					multi {
						value = "world"
					}
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &complexDef{
						Multiple: &[]sub{
							{Val: "hello"},
							{Val: "world"},
						},
					}},
				},
			},
		},
		{
			name: "MultipleBlocksToPointers",
			body: parseBody(t, `
				resource "slice_ptr" "foo" {
					sub {
						value = "hello"
					}
					sub {
						value = "world"
					}
				}
			`),
			resources: []resource.Definition{&slicePtrDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &slicePtrDef{
						Subs: []*sub{
							{Val: "hello"},
							{Val: "world"},
						},
					}},
				},
			},
		},
		{
			name: "ConvertStructArgument",
			body: parseBody(t, `
				resource "complex" "foo" {
					nested {
						sub {
							value = 3.14 # assigned to string
						}
					}
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Name: "foo", Def: &complexDef{
						Child: &Child{
							Sub: sub{
								Val: "3.14",
							},
						},
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer checkPanic(t)
			g := graph.New()
			ctx := &decoder.DecodeContext{Resources: resource.RegistryFromResources(tt.resources...)}
			proj, diags := decoder.DecodeBody(tt.body, ctx, g)
			if diags.HasErrors() {
				t.Fatalf("DecodeBody() Diagnostics\n%s", diags)
			}
			snap := snapshot.Take(g)
			if diff := cmp.Diff(snap, tt.wantSnap, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Snapshot does not match (-got, +want)\n%s", diff)
			}
			if diff := cmp.Diff(proj, tt.wantProj); diff != "" {
				t.Errorf("Project does not match (-got, +want)\n%s", diff)
			}
		})
	}
}

func TestDecodeBody_Diagnostics(t *testing.T) {
	tests := []struct {
		name      string
		body      hcl.Body
		resources []resource.Definition
		diags     hcl.Diagnostics
	}{
		{
			name: "UnsupportedArgument",
			body: parseBody(t, `
				resource "simple" "foo" {
					input = "hello"
					notsupported = 123
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported argument",
				Detail:   `An argument named "notsupported" is not expected here.`,
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 3, Column: 6},
					End:   hcl.Pos{Line: 3, Column: 18},
				},
			}},
		},
		{
			name: "InvalidSource",
			body: parseBody(t, `
				resource "simple" "foo" {
					input = "hello"

					source = "xxx"
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Could not decode source information",
				Detail:   "Error: string must contain 3 parts separated by ':'. This is always a bug.",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1},
					End:   hcl.Pos{Line: 1, Column: 24},
				},
			}},
		},
		{
			name: "NonexistingDependencies",
			body: parseBody(t, `
				resource "simple" "bar" {
					input = "hello"
				}
				resource "simple" "baz" {
					input = sample.bar.output
				}
				resource "simple" "qux" {
					input = simple.baz.input
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Referenced value not found",
					Detail:   "Field sample.bar.output does not exist",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 5, Column: 14},
						End:   hcl.Pos{Line: 5, Column: 31},
					},
				},
				{
					Severity: hcl.DiagError,
					Summary:  "Referenced value not found",
					Detail:   "Nested field sample.bar.output does not exist",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 8, Column: 14},
						End:   hcl.Pos{Line: 8, Column: 30},
					},
				},
			},
		},
		{
			name: "InvalidReference",
			body: parseBody(t, `
				resource "simple" "foo" {
					input = "hello"
				}
				resource "simple" "syntax" {
					input = "${simple.foo.output.qux}"
				}
				resource "simple" "syntax" {
					input = simple.foo.output.qux
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference",
					Detail:   "A reference must have 3 fields: {type}.{name}.{field}.",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 5, Column: 17},
						End:   hcl.Pos{Line: 5, Column: 38},
					},
				},
				{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference",
					Detail:   "A reference must have 3 fields: {type}.{name}.{field}.",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 8, Column: 14},
						End:   hcl.Pos{Line: 8, Column: 35},
					},
				},
			},
		},
		{
			name: "StructWithDependency",
			body: parseBody(t, `
				resource "simple" "foo" {
					input = "hello"
				}
				resource "complex" "foo" {
					nested {
						sub {
							value = "arn::${simple.foo.output}"
						}
					}
				}
			`),
			resources: []resource.Definition{&simpleDef{}, &complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Variables not allowed",           // Would be nice to support variables
				Detail:   "Variables may not be used here.", // but for now this is out of scope.
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 7, Column: 24},
					End:   hcl.Pos{Line: 7, Column: 30},
				},
			}},
		},
		{
			name: "StructMissingAttribute",
			body: parseBody(t, `
				resource "complex" "foo" {
					nested {
						sub {
							# missing required value
						}
					}
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   `The argument "value" is required, but no definition was found.`,
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 5, Column: 8},
					End:   hcl.Pos{Line: 5, Column: 8},
				},
			}},
		},
		{
			name: "StructAssignInvalid",
			body: parseBody(t, `
				resource "complex" "foo" {
					nested {
						sub {
							value = ["hello", "world"]
						}
					}
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "Unsuitable value: string required",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 4, Column: 16},
					End:   hcl.Pos{Line: 4, Column: 17},
				},
			}},
		},
		{
			name: "MultipleBlocksNotAllowed",
			body: parseBody(t, `
				resource "complex" "foo" {
					nested {
						value = "hello"
					}
					nested {
						value = "world"
					}
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Duplicate nested block",
				Detail:   "Only one nested block is allowed. Another was defined on line 2",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 5, Column: 6},
					End:   hcl.Pos{Line: 5, Column: 12},
				},
			}},
		},
		{
			name: "MissingBlock",
			body: parseBody(t, `
				resource "required" "foo" {
					# required block not set
				}
			`),
			resources: []resource.Definition{&requiredBlockDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required block",
				Detail:   "A required block is required.",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 3, Column: 6},
					End:   hcl.Pos{Line: 3, Column: 6},
				},
			}},
		},
		{
			name: "MissingNestedBlock",
			body: parseBody(t, `
				resource "complex" "foo" {
					nested {
					}
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing sub block",
				Detail:   "A sub block is required.",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 3, Column: 7},
					End:   hcl.Pos{Line: 3, Column: 7},
				},
			}},
		},
		{
			name: "NoProjectName",
			body: parseBody(t, `
				project "" {}
			`),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Project name not set",
				Detail:   "A project name cannot be blank.",
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
			name: "NoResourceType",
			body: parseBody(t, `
				resource "" "foo" {}
			`),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource type not set",
				Detail:   "A resource type cannot be blank.",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 10},
					End:   hcl.Pos{Line: 1, Column: 12},
				},
				Context: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1},
					End:   hcl.Pos{Line: 1, Column: 18},
				},
			}},
		},
		{
			name: "NoResourceName",
			body: parseBody(t, `
				resource "foo" "" {}
			`),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource name not set",
				Detail:   "A resource name cannot be blank.",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 16},
					End:   hcl.Pos{Line: 1, Column: 18},
				},
				Context: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1},
					End:   hcl.Pos{Line: 1, Column: 18},
				},
			}},
		},
		{
			name: "InvalidType",
			body: parseBody(t, `
				resource "complex" "foo" {
					int = "this cannot be an int"
				}
			`),
			resources: []resource.Definition{&complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "Unsuitable value: a number is required",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 2, Column: 13},
					End:   hcl.Pos{Line: 2, Column: 34},
				},
				Context: &hcl.Range{
					Start: hcl.Pos{Line: 2, Column: 12},
					End:   hcl.Pos{Line: 2, Column: 35},
				},
			}},
		},
		{
			name: "ResourceNotFound",
			body: parseBody(t, `
				resource "notfound" "bar" {}
			`),
			resources: []resource.Definition{&simpleDef{}}, // resource "notfound" not registered
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource not supported",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 10},
					End:   hcl.Pos{Line: 1, Column: 20},
				},
			}},
		},
		{
			name: "SuggestResource",
			body: parseBody(t, `
				resource "sample" "bar" {}
			`),
			resources: []resource.Definition{&simpleDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource not supported",
				Detail:   "Did you mean \"simple\"?",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 10},
					End:   hcl.Pos{Line: 1, Column: 18},
				},
			}},
		},
		{
			name: "InvalidInputType",
			body: parseBody(t, `
				resource "simple" "a" {
					input = "hello"    # string
				}
				resource "complex" "b" {
					int = simple.a.input # cannot assign string to int
				}
			`),
			resources: []resource.Definition{&simpleDef{}, &complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "Unsuitable value: a number is required",
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 5, Column: 12},
					End:   hcl.Pos{Line: 5, Column: 26},
				},
			}},
		},
		{
			name: "MissingRequiredArg",
			body: parseBody(t, `
				resource "simple" "a" {
					# input not set
				}
			`),
			resources: []resource.Definition{&simpleDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   `The argument "input" is required, but no definition was found.`,
				Subject: &hcl.Range{
					Start: hcl.Pos{Line: 3, Column: 6},
					End:   hcl.Pos{Line: 3, Column: 6},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer checkPanic(t)
			g := graph.New()
			ctx := &decoder.DecodeContext{Resources: resource.RegistryFromResources(tt.resources...)}
			_, diags := decoder.DecodeBody(tt.body, ctx, g)
			if diff := diffDiagnostics(diags, tt.diags); diff != "" {
				t.Errorf("DecodeBody() diagnostics (-got, +want)\n%s", diff)
			}
		})
	}
}

// ---

func checkPanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Fatalf("Panic: %v\n\n%s", err, debug.Stack())
	}
}

func diffDiagnostics(got, want hcl.Diagnostics) string {
	e1 := make([]string, len(got))
	e2 := make([]string, len(want))
	for i, d := range got {
		e1[i] = d.Error()
	}
	for i, d := range want {
		e2[i] = d.Error()
	}
	return cmp.Diff(e1, e2, cmpopts.SortSlices(func(a, b string) bool { return a < b }))
}

func parseBody(t *testing.T, src string) hcl.Body {
	t.Helper()
	// NOTE: we could use hclsyntax.ParseConfig but we'll use hclpack to ensure
	// the special types there are handled correctly.
	src = strings.TrimSpace(src)
	body, diags := hclpack.PackNativeFile([]byte(src), "", hcl.Pos{Byte: 0, Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Errorf("Parse test body: %v", diags)
	}
	return body
}

type simpleDef struct {
	resource.Definition
	Input  string `input:"input"`
	Output string `output:"output"`
}

func (d *simpleDef) Type() string { return "simple" }

type complexDef struct {
	resource.Definition

	Map      *map[string]string `input:"map"`
	Slice    *[]string          `input:"slice"`
	Child    *Child             `input:"nested"`
	Multiple *[]sub             `input:"multi"`
	Int      *int               `input:"int"`
}

type Child struct {
	Sub sub `input:"sub"`
}

type sub struct {
	Val      string  `input:"value"`
	Optional *string `input:"optional"`
}

func (*complexDef) Type() string { return "complex" }

type requiredBlockDef struct {
	resource.Definition
	Child Child `input:"required"`
}

func (*requiredBlockDef) Type() string { return "required" }

type slicePtrDef struct {
	resource.Definition
	Subs []*sub `input:"sub"`
}

func (*slicePtrDef) Type() string { return "slice_ptr" }

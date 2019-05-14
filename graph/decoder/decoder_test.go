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
		resources map[string]resource.Definition
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
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = "world"
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &simpleDef{Input: "hello"}},
					{Type: "a", Name: "bar", Def: &simpleDef{Input: "world"}},
				},
			},
		},
		{
			name: "Source",
			body: parseBody(t, `
				resource "foo" {
					type   = "a"
					input  = "src"
					source = "ff:abc:def"
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &simpleDef{Input: "src"}},
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
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = foo.input
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &simpleDef{Input: "hello"}},
					{Type: "a", Name: "bar", Def: &simpleDef{Input: "hello"}}, // Input can be statically resolved.
				},
			},
		},
		{
			name: "DependencyToInputExtended",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = foo.input
				}
				resource "baz" {
					type  = "a"
					input = bar.input
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &simpleDef{Input: "hello"}},
					{Type: "a", Name: "bar", Def: &simpleDef{Input: "hello"}},
					{Type: "a", Name: "baz", Def: &simpleDef{Input: "hello"}}, // Input can be statically resolved through baz.
				},
			},
		},
		{
			name: "DependencyToOutput",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = foo.output
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &simpleDef{Input: "hello"}},
					{Type: "a", Name: "bar", Def: &simpleDef{}}, // Input is dynamic.
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${bar.input}": "${foo.output}",
				},
			},
		},
		{
			name: "DependencyExpression",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = ":: ${foo.input} - ${foo.output} <<<"
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &simpleDef{Input: "hello"}},
					{Type: "a", Name: "bar", Def: &simpleDef{}},
				},
				Dependencies: map[snapshot.Expr]snapshot.Expr{
					"${bar.input}": ":: hello - ${foo.output} <<<", // Partially resolved.
				},
			},
		},
		{
			name: "ConvertType",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = 3.14
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &simpleDef{Input: "3.14"}}, // Converted to string.
				},
			},
		},
		{
			name: "Map",
			body: parseBody(t, `
				resource "foo" {
					type = "a"
					map = {
						foo = "bar"
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &complexDef{
						Map: &map[string]string{"foo": "bar"},
					}},
				},
			},
		},
		{
			name: "Slice",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					slice = ["hello", "world"]
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &complexDef{
						Slice: &[]string{"hello", "world"},
					}},
				},
			},
		},
		{
			name: "Struct",
			body: parseBody(t, `
				resource "foo" {
					type = "a"
					nested {
						sub {
							value = "hello"
						}
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &complexDef{
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
				resource "foo" {
					type = "a"
					multi {
						value = "hello"
					}
					multi {
						value = "world"
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &complexDef{
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
				resource "foo" {
					type = "a"
					sub {
						value = "hello"
					}
					sub {
						value = "world"
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &slicePtrDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &slicePtrDef{
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
				resource "foo" {
					type = "a"
					nested {
						sub {
							value = 3.14 # assigned to string
						}
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			wantSnap: snapshot.Snap{
				Resources: []resource.Resource{
					{Type: "a", Name: "foo", Def: &complexDef{
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
			ctx := &decoder.DecodeContext{Resources: resource.RegistryFromResources(tt.resources)}
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
		resources map[string]resource.Definition
		diags     hcl.Diagnostics
	}{
		{
			name: "MissingType",
			body: parseBody(t, `
				resource "foo" {
					input = "a"
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   `The argument "type" is required, but no definition was found.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 6},
					End:      hcl.Pos{Line: 3, Column: 6},
				},
			}},
		},
		{
			name: "UnsupportedArgument",
			body: parseBody(t, `
				resource "foo" {
					type         = "a"
					input        = "hello"
					notsupported = 123
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported argument",
				Detail:   `An argument named "notsupported" is not expected here.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 4, Column: 6},
					End:      hcl.Pos{Line: 4, Column: 18},
				},
			}},
		},
		{
			name: "InvalidSource",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"

					source = "xxx"
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Could not decode source information",
				Detail:   "Error: string must contain 3 parts separated by ':'. This is always a bug.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1},
					End:      hcl.Pos{Line: 1, Column: 15},
				},
			}},
		},
		{
			name: "NonexistingDependencies",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = zoo.output
				}
				resource "baz" {
					type  = "a"
					input = bar.input
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Referenced value not found",
					Detail:   "An object with name \"zoo\" is not defined",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 7, Column: 14},
						End:      hcl.Pos{Line: 7, Column: 24},
					},
				},
				{
					Severity: hcl.DiagError,
					Summary:  "Referenced value not found",
					Detail:   "A nested object with name \"zoo\" is not defined",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 11, Column: 14},
						End:      hcl.Pos{Line: 11, Column: 23},
					},
				},
			},
		},
		{
			name: "NonexistingDependencyField",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = foo.nonexisting
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Referenced value not found",
					Detail:   "Object foo does not have a field \"nonexisting\"",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 7, Column: 14},
						End:      hcl.Pos{Line: 7, Column: 29},
					},
				},
			},
		},
		{
			name: "InvalidReference",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type  = "a"
					input = foo.output.value # nested ref not supported
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			diags: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference",
					Detail:   "A reference must have 2 fields: {name}.{field}.",
					Subject: &hcl.Range{
						Filename: "file.hcl",
						Start:    hcl.Pos{Line: 7, Column: 14},
						End:      hcl.Pos{Line: 7, Column: 30},
					},
				},
			},
		},
		{
			name: "StructWithDependency",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "bar" {
					type = "b"
					nested {
						sub {
							value = "arn::${simple.foo.output}"
						}
					}
				}
			`),
			resources: map[string]resource.Definition{
				"a": &simpleDef{},
				"b": &complexDef{},
			},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Variables not allowed",           // Would be nice to support variables
				Detail:   "Variables may not be used here.", // but for now this is out of scope.
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 9, Column: 24},
					End:      hcl.Pos{Line: 9, Column: 30},
				},
			}},
		},
		{
			name: "StructMissingAttribute",
			body: parseBody(t, `
				resource "foo" {
					type = "a"
					nested {
						sub {
							# missing required value
						}
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   `The argument "value" is required, but no definition was found.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 6, Column: 8},
					End:      hcl.Pos{Line: 6, Column: 8},
				},
			}},
		},
		{
			name: "StructAssignInvalid",
			body: parseBody(t, `
				resource "foo" {
					type = "a"
					nested {
						sub {
							value = ["hello", "world"]
						}
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "Unsuitable value: string required",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 5, Column: 16},
					End:      hcl.Pos{Line: 5, Column: 17},
				},
			}},
		},
		{
			name: "MultipleBlocksNotAllowed",
			body: parseBody(t, `
				resource "foo" {
					type = "a"
					nested {
						value = "hello"
					}
					nested {
						value = "world"
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Duplicate nested block",
				Detail:   "Only one nested block is allowed. Another was defined on line 3",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 6, Column: 6},
					End:      hcl.Pos{Line: 6, Column: 12},
				},
			}},
		},
		{
			name: "MissingBlock",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					# required block not set
				}
			`),
			resources: map[string]resource.Definition{
				"a": &struct { // nolint: maligned
					resource.Definition
					RequiredChild struct{} `func:"input,required"`
				}{},
			},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required block",
				Detail:   "A required_child block is required.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 4, Column: 6},
					End:      hcl.Pos{Line: 4, Column: 6},
				},
			}},
		},
		{
			name: "MissingNestedBlock",
			body: parseBody(t, `
				resource "foo" {
					type = "a"
					nested {
					}
				}
			`),
			resources: map[string]resource.Definition{"a": &complexDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required block",
				Detail:   "A sub block is required.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 4, Column: 7},
					End:      hcl.Pos{Line: 4, Column: 7},
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
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 9},
					End:      hcl.Pos{Line: 1, Column: 11},
				},
				Context: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1},
					End:   hcl.Pos{Line: 1, Column: 11},
				},
			}},
		},
		{
			name: "NoResourceName",
			body: parseBody(t, `
				resource "" {}
			`),
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource name not set",
				Detail:   "A resource name cannot be blank.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 10},
					End:      hcl.Pos{Line: 1, Column: 12},
				},
				Context: &hcl.Range{
					Start: hcl.Pos{Line: 1, Column: 1},
					End:   hcl.Pos{Line: 1, Column: 18},
				},
			}},
		},
		{
			name: "DuplicateResource",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"
					input = "hello"
				}
				resource "foo" {        # duplicate
					type  = "a"
					input = "world"
				}
			`),
			resources: map[string]resource.Definition{"a": &simpleDef{}},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Duplicate resource",
				Detail:   `Another resource "foo" was defined on in file.hcl on line 1`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 5, Column: 5},
					End:      hcl.Pos{Line: 5, Column: 19},
				},
				Context: &hcl.Range{
					Start: hcl.Pos{Line: 5, Column: 5},
					End:   hcl.Pos{Line: 5, Column: 19},
				},
			}},
		},
		{
			name: "InvalidType",
			body: parseBody(t, `
				resource "foo" {
					type = "a"
					int = "this cannot be an int"
				}
			`),
			resources: map[string]resource.Definition{
				"a": &struct {
					resource.Definition
					Int int `func:"input"`
				}{},
			},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "Unsuitable value: a number is required",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 13},
					End:      hcl.Pos{Line: 3, Column: 34},
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
				resource "bar" {
					type = "notfound"
				}
			`),
			resources: map[string]resource.Definition{}, // resource "notfound" not registered
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource not supported",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1},
					End:      hcl.Pos{Line: 1, Column: 15},
				},
			}},
		},
		{
			name: "SuggestResource",
			body: parseBody(t, `
				resource "bar" {
					type = "sample"
				}
			`),
			resources: map[string]resource.Definition{
				"simple": &simpleDef{},
			},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource not supported",
				Detail:   "Did you mean \"simple\"?",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1},
					End:      hcl.Pos{Line: 1, Column: 15},
				},
			}},
		},
		{
			name: "InvalidInputType",
			body: parseBody(t, `
				resource "a" {
					type  = "a"
					input = "hello"    # string
				}
				resource "b" {
					type = "b"
					int  = a.input # cannot assign string to int
				}
			`),
			resources: map[string]resource.Definition{
				"a": &simpleDef{},
				"b": &complexDef{},
			},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "Unsuitable value: a number is required",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 7, Column: 13},
					End:      hcl.Pos{Line: 7, Column: 20},
				},
			}},
		},
		{
			name: "MissingRequiredArg",
			body: parseBody(t, `
				resource "a" {
					type  = "a"
					# input not set
				}
			`),
			resources: map[string]resource.Definition{
				"a": &struct {
					resource.Definition
					Input string `func:"input,required"`
				}{},
			},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   `The argument "input" is required, but no definition was found.`,
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 4, Column: 6},
					End:      hcl.Pos{Line: 4, Column: 6},
				},
			}},
		},
		{
			name: "ValidationError",
			body: parseBody(t, `
				resource "foo" {
					type  = "a"

					season = "tuesday"
				}
			`),
			resources: map[string]resource.Definition{
				"a": &struct {
					resource.Definition
					Season string `func:"input" validate:"oneof=spring summer fall winter"`
				}{},
			},
			diags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Validation error",
				Detail:   "Value for season must be one of: [spring summer fall winter]",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 4, Column: 16},
					End:      hcl.Pos{Line: 4, Column: 23},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer checkPanic(t)
			g := graph.New()
			ctx := &decoder.DecodeContext{Resources: resource.RegistryFromResources(tt.resources)}
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
	body, diags := hclpack.PackNativeFile([]byte(src), "file.hcl", hcl.Pos{Byte: 0, Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Errorf("Parse test body: %v", diags)
	}
	return body
}

type simpleDef struct {
	resource.Definition
	Input  string `func:"input,required"`
	Output string `func:"output"`
}

type complexDef struct {
	resource.Definition

	Map      *map[string]string `func:"input"`
	Slice    *[]string          `func:"input"`
	Child    *Child             `func:"input" name:"nested"`
	Multiple *[]sub             `func:"input" name:"multi"`
	Int      *int               `func:"input"`
}

type Child struct {
	Sub sub `func:"input"`
}

type sub struct {
	Val      string  `func:"input,required" name:"value"`
	Optional *string `func:"input"`
}

type slicePtrDef struct {
	resource.Definition
	Subs []*sub `func:"input" name:"sub"`
}

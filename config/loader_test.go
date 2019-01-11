package config_test

import (
	"encoding/json"
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
		name string
		root string
		want *hclpack.Body
	}{
		{
			"Project",
			"testdata/project",
			&hclpack.Body{
				ChildBlocks: []hclpack.Block{
					{
						Type:   "resource",
						Labels: []string{"aws_lambda_function", "func"},
						Body: hclpack.Body{
							Attributes: map[string]hclpack.Attribute{
								"digest": {
									Expr: hclpack.Expression{
										Source:     []byte(`"b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c"`),
										SourceType: hclpack.ExprLiteralJSON,
										Range_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 2, Column: 12, Byte: 51},
											End:      hcl.Pos{Line: 2, Column: 19, Byte: 58},
										},
										StartRange_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 2, Column: 13, Byte: 52},
											End:      hcl.Pos{Line: 2, Column: 18, Byte: 57},
										},
									},
									Range: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 2, Column: 3, Byte: 42},
										End:      hcl.Pos{Line: 2, Column: 19, Byte: 58},
									},
									NameRange: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 2, Column: 3, Byte: 42},
										End:      hcl.Pos{Line: 2, Column: 9, Byte: 48},
									},
								},
								"handler": {
									Expr: hclpack.Expression{
										Source:     []byte(`"index.handler"`),
										SourceType: hclpack.ExprNative,
										Range_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 4, Column: 13, Byte: 72},
											End:      hcl.Pos{Line: 4, Column: 28, Byte: 87},
										},
										StartRange_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 4, Column: 14, Byte: 73},
											End:      hcl.Pos{Line: 4, Column: 27, Byte: 86},
										},
									},
									Range: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 4, Column: 3, Byte: 62},
										End:      hcl.Pos{Line: 4, Column: 28, Byte: 87},
									},
									NameRange: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 4, Column: 3, Byte: 62},
										End:      hcl.Pos{Line: 4, Column: 10, Byte: 69},
									},
								},
								"memory": {
									Expr: hclpack.Expression{
										Source:     []byte("512"),
										SourceType: hclpack.ExprNative,
										Range_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 5, Column: 13, Byte: 100},
											End:      hcl.Pos{Line: 5, Column: 16, Byte: 103},
										},
										StartRange_: hcl.Range{
											Filename: "testdata/project/func.hcl",
											Start:    hcl.Pos{Line: 5, Column: 13, Byte: 100},
											End:      hcl.Pos{Line: 5, Column: 16, Byte: 103},
										},
									},
									Range: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 5, Column: 3, Byte: 90},
										End:      hcl.Pos{Line: 5, Column: 16, Byte: 103},
									},
									NameRange: hcl.Range{
										Filename: "testdata/project/func.hcl",
										Start:    hcl.Pos{Line: 5, Column: 3, Byte: 90},
										End:      hcl.Pos{Line: 5, Column: 9, Byte: 96},
									},
								},
							},
							MissingItemRange_: hcl.Range{
								Filename: "testdata/project/func.hcl",
								Start:    hcl.Pos{Line: 6, Column: 2, Byte: 105},
								End:      hcl.Pos{Line: 6, Column: 2, Byte: 105},
							},
						},
						DefRange: hcl.Range{
							Filename: "testdata/project/func.hcl",
							Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
							End:      hcl.Pos{Line: 1, Column: 38, Byte: 37},
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
								End:      hcl.Pos{Line: 1, Column: 31, Byte: 30},
							},
							{
								Filename: "testdata/project/func.hcl",
								Start:    hcl.Pos{Line: 1, Column: 32, Byte: 31},
								End:      hcl.Pos{Line: 1, Column: 38, Byte: 37},
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
					{
						Type:   "resource",
						Labels: []string{"aws_iam_role", "role"},
						Body: hclpack.Body{
							Attributes: map[string]hclpack.Attribute{
								"role_name": {
									Expr: hclpack.Expression{
										Source:     []byte(`"tester"`),
										SourceType: hclpack.ExprNative,
										Range_: hcl.Range{
											Filename: "testdata/project/proj.hcl",
											Start:    hcl.Pos{Line: 4, Column: 15, Byte: 66},
											End:      hcl.Pos{Line: 4, Column: 23, Byte: 74},
										},
										StartRange_: hcl.Range{
											Filename: "testdata/project/proj.hcl",
											Start:    hcl.Pos{Line: 4, Column: 16, Byte: 67},
											End:      hcl.Pos{Line: 4, Column: 22, Byte: 73},
										},
									},
									Range: hcl.Range{
										Filename: "testdata/project/proj.hcl",
										Start:    hcl.Pos{Line: 4, Column: 3, Byte: 54},
										End:      hcl.Pos{Line: 4, Column: 23, Byte: 74},
									},
									NameRange: hcl.Range{
										Filename: "testdata/project/proj.hcl",
										Start:    hcl.Pos{Line: 4, Column: 3, Byte: 54},
										End:      hcl.Pos{Line: 4, Column: 12, Byte: 63},
									},
								},
							},
							ChildBlocks: nil,
							MissingItemRange_: hcl.Range{
								Filename: "testdata/project/proj.hcl",
								Start:    hcl.Pos{Line: 5, Column: 2, Byte: 76},
								End:      hcl.Pos{Line: 5, Column: 2, Byte: 76},
							},
						},
						DefRange: hcl.Range{
							Filename: "testdata/project/proj.hcl",
							Start:    hcl.Pos{Line: 3, Column: 1, Byte: 19},
							End:      hcl.Pos{Line: 3, Column: 31, Byte: 49},
						},
						TypeRange: hcl.Range{
							Filename: "testdata/project/proj.hcl",
							Start:    hcl.Pos{Line: 3, Column: 1, Byte: 19},
							End:      hcl.Pos{Line: 3, Column: 9, Byte: 27},
						},
						LabelRanges: []hcl.Range{
							{
								Filename: "testdata/project/proj.hcl",
								Start:    hcl.Pos{Line: 3, Column: 10, Byte: 28},
								End:      hcl.Pos{Line: 3, Column: 24, Byte: 42},
							},
							{
								Filename: "testdata/project/proj.hcl",
								Start:    hcl.Pos{Line: 3, Column: 25, Byte: 43},
								End:      hcl.Pos{Line: 3, Column: 31, Byte: 49},
							},
						},
					},
				},
				MissingItemRange_: hcl.Range{
					Filename: "testdata/project/func.hcl",
					Start:    hcl.Pos{Line: 8, Column: 1, Byte: 107},
					End:      hcl.Pos{Line: 8, Column: 1, Byte: 107},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &config.Loader{}
			got, diags := l.Load(tt.root)
			if diags.HasErrors() {
				t.Fatalf("Loader.Load() error = %v", diags)
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Load() (-got, +want)\n%s", diff)
			}
		})
	}
}

var projectWithSyntaxErrors = "testdata/invalid"

func ExampleLoader_PrintDiagnostics() {
	l := &config.Loader{}
	_, diags := l.Load(projectWithSyntaxErrors)
	l.PrintDiagnostics(os.Stdout, diags)
	// Output:
	// Error: Missing newline after block definition
	//
	//   on testdata/invalid/invalid.hcl line 6:
	//    4: resource "invalid" "syntax" {
	//    5:   # too many closing braces
	//    6: } }
	//
	// A block definition must end with a newline.
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
			l := &config.Loader{}
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
			l := &config.Loader{}
			got, err := l.Load(tt.root)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			// Assert that every digest set returns files
			var root config.Root
			diags := gohcl.DecodeBody(got, nil, &root)
			if diags.HasErrors() {
				t.Errorf("Decode() error = %v", diags.Error())
			}

			for _, r := range root.Resources {
				if r.SourceDigest != nil {
					files := l.Source(*r.SourceDigest)
					if len(files) == 0 {
						t.Errorf("Files() returned no source for digest %q", *r.SourceDigest)
					}
				}
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

	l := &config.Loader{}
	before, diags := l.Load("testdata/project")
	if diags.HasErrors() {
		t.Fatalf("Load() error = %v", diags)
	}

	j, err := json.Marshal(before)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

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
		l.PrintDiagnostics(os.Stderr, diags)
		os.Exit(1)
	}

	// Load config files from root
	cfg, diags := l.Load(rootDir)
	if diags.HasErrors() {
		l.PrintDiagnostics(os.Stderr, diags)
		os.Exit(1)
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

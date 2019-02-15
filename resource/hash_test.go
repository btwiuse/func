package resource_test

import (
	"fmt"
	"testing"

	"github.com/func/func/resource"
)

// nolint: maligned
type kitchensink struct {
	resource.Definition
	Bool       bool           `input:""`
	Int        int            `input:""`
	Int8       int8           `input:""`
	Int16      int16          `input:""`
	Int32      int32          `input:""`
	Int64      int64          `input:""`
	Uint       uint           `input:""`
	Uint8      uint8          `input:""`
	Uint16     uint16         `input:""`
	Uint32     uint32         `input:""`
	Uint64     uint64         `input:""`
	Uintptr    uintptr        `input:""`
	Float32    float32        `input:""`
	Float64    float64        `input:""`
	Complex64  complex64      `input:""`
	Complex128 complex128     `input:""`
	Array      [3]int         `input:""`
	Map        map[string]int `input:""`
	Ptr        *string        `input:""`
	Slice      []string       `input:""`
	String     string         `input:""`
	Struct     nested         `input:""`
	StructPtr  *nested        `input:""`
}

type nested struct {
	String string
	Ptr    *string
}

func (r *kitchensink) Type() string { return "kitchensink" }

func strptr(v string) *string { return &v }
func int64ptr(v int64) *int64 { return &v }
func boolptr(v bool) *bool    { return &v } // nolint: unparam

func TestHash_identity(t *testing.T) {
	tests := []resource.Definition{
		&kitchensink{},
		&kitchensink{Bool: true},
		&kitchensink{Int: 1},
		&kitchensink{Int8: 8},
		&kitchensink{Int16: 16},
		&kitchensink{Int32: 32},
		&kitchensink{Int64: 64},
		&kitchensink{Uint: 1},
		&kitchensink{Uint8: 8},
		&kitchensink{Uint16: 16},
		&kitchensink{Uint32: 32},
		&kitchensink{Uint64: 64},
		&kitchensink{Uintptr: 1},
		&kitchensink{Float32: 3.14159},
		&kitchensink{Float64: 2.71828},
		&kitchensink{Complex64: 3.0},
		&kitchensink{Complex128: 5.0},
		&kitchensink{Array: [3]int{1, 2, 3}},
		&kitchensink{Map: map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}},
		&kitchensink{Ptr: strptr("ptr")},
		&kitchensink{Slice: []string{"a", "b", "c"}},
		&kitchensink{Struct: nested{String: "a"}},
		&kitchensink{Struct: nested{Ptr: strptr("a")}},
		&kitchensink{StructPtr: &nested{String: "a"}},
	}

	for i, def := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			// Run same test many times to ensure maps are iterated in a
			// deterministic order.
			checks := 100
			results := make([]string, checks)
			for i := range results {
				// Hash the randomly generated resource.
				got := resource.Hash(def)
				results[i] = got
			}

			if results[0] == "" {
				t.Fatal("Empty hash")
			}

			// Ensure all values are the same
			want := results[0]
			for _, got := range results {
				if got != want {
					t.Logf("%#v", def)
					t.Fatalf("Did not hash to same value: %s != %s", got, want)
				}
			}
		})
	}
}

func TestHash(t *testing.T) {
	tests := []struct {
		name string
		a, b resource.Definition
		same bool
	}{
		{
			"DiffType",
			&def1{FunctionName: "foo"},
			&realistic{FunctionName: "foo"},
			false,
		},
		{
			"Realistic",
			&realistic{
				FunctionName: "test",
				Handler:      "index.handler",
				MemorySize:   int64ptr(512),
				Publish:      boolptr(true),
				Role:         "arn:...",
				Timeout:      int64ptr(5),
				CodeSHA256:   "abc",
			},
			&realistic{
				FunctionName: "test",
				Handler:      "index.handler",
				MemorySize:   int64ptr(512),
				Publish:      boolptr(true),
				Role:         "arn:...",
				Timeout:      int64ptr(5),
				CodeSHA256:   "abc",
			},
			true,
		},
		{
			"DiffInput",
			&realistic{
				FunctionName: "test",
				Handler:      "index.handler",
				MemorySize:   int64ptr(512),
				Publish:      boolptr(true),
				Role:         "arn:...",
				Timeout:      int64ptr(5),
				CodeSHA256:   "abc",
			},
			&realistic{
				FunctionName: "test",
				Handler:      "index.handler",
				MemorySize:   int64ptr(1024), // increase
				Publish:      boolptr(true),
				Role:         "arn:...",
				Timeout:      int64ptr(5),
				CodeSHA256:   "abc",
			},
			false,
		},
		{
			"DiffOutput",
			&realistic{
				FunctionName: "test",
				Handler:      "index.handler",
				MemorySize:   int64ptr(512),
				Publish:      boolptr(true),
				Role:         "arn:...",
				Timeout:      int64ptr(5),
				CodeSHA256:   "abc",
			},
			&realistic{
				FunctionName: "test",
				Handler:      "index.handler",
				MemorySize:   int64ptr(512),
				Publish:      boolptr(true),
				Role:         "arn:...",
				Timeout:      int64ptr(5),
				CodeSHA256:   "def", // changed
			},
			true,
		},
		{
			"MapDiffKey",
			&realistic{
				FunctionName: "test",
				Environment: &env{
					Variables: map[string]string{
						"foo": "foo",
						"bar": "bar",
						"baz": "baz",
					},
				},
			},
			&realistic{
				FunctionName: "test",
				Environment: &env{
					Variables: map[string]string{
						"foo": "qux",
						"bar": "bar",
						"baz": "baz",
					},
				},
			},
			false,
		},
		{
			"MapDiffVal",
			&realistic{
				FunctionName: "test",
				Environment: &env{
					Variables: map[string]string{
						"foo": "foo",
						"bar": "bar",
						"baz": "baz",
					},
				},
			},
			&realistic{
				FunctionName: "test",
				Environment: &env{
					Variables: map[string]string{
						"qux": "foo",
						"bar": "foo",
						"baz": "baz",
					},
				},
			},
			false,
		},
		{
			"IgnoreNonIO",
			&def2{NotIO: "foo", Output: "a"}, // Field 'NotIO' does not have
			&def2{NotIO: "bar", Output: "a"}, // an input or output struct tag.
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := resource.Hash(tt.a)
			b := resource.Hash(tt.b)

			got := a == b
			if got != tt.same {
				t.Errorf("Hash() got same %t, want %t", a == b, tt.same)
			}
		})
	}
}

type def1 struct {
	resource.Definition
	FunctionName string `input:"v"`
	ARN          bool   `output:"o"`
}

func (def1) Type() string { return "def1" }

type def2 struct {
	resource.Definition
	NotIO  string // no input or output tag
	Output string `output:"v"`
}

func (def2) Type() string { return "def2" }

// realistic is a resource roughly resembling an aws lambda resource
type realistic struct {
	resource.Definition

	// Inputs
	DeadLetterConfig *struct {
		TargetArn *string `input:"target_arn"`
	} `input:"dead_letter_config"`
	Description   *string           `input:"description"`
	Environment   *env              `input:"environment"`
	FunctionName  string            `input:"function_name"`
	Handler       string            `input:"handler"`
	KMSKeyArn     *string           `input:"kms_key_arn"`
	Layers        []string          `input:"layers"`
	MemorySize    *int64            `input:"memory_size"`
	Publish       *bool             `input:"publish"`
	Role          string            `input:"role"`
	Runtime       string            `input:"runtime"` // actually an enum
	Targs         map[string]string `input:"tags"`
	Timeout       *int64            `input:"timeout"`
	TracingConfig *struct {
		Mode string `input:"mode"`
	} `input:"tracing_config"`
	VPCConfig *struct {
		SecurityGroupIDs []string `input:"security_group_ids"`
		SubnetIDs        []string `input:"subnet_ids"`
	} `input:"vpc_config"`

	// Outputs
	CodeSHA256   string `output:"code_sha_256"`
	CodeSize     int64  `output:"code_size"`
	FunctionARN  string `output:"function_arn"`
	LastModified string `output:"last_modified"` // ISO8601
	MasterARN    string `output:"master_arn"`
	RevisionID   string `output:"revision_id"`
	Version      string `output:"version"`
}

type env struct {
	Variables map[string]string `input:"variables"`
}

func (realistic) Type() string { return "realistic" }

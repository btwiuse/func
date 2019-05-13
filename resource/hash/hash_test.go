package hash_test

import (
	"fmt"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/resource/hash"
)

// nolint: maligned
type kitchensink struct {
	resource.Definition
	Bool       bool           `func:"input"`
	Int        int            `func:"input"`
	Int8       int8           `func:"input"`
	Int16      int16          `func:"input"`
	Int32      int32          `func:"input"`
	Int64      int64          `func:"input"`
	Uint       uint           `func:"input"`
	Uint8      uint8          `func:"input"`
	Uint16     uint16         `func:"input"`
	Uint32     uint32         `func:"input"`
	Uint64     uint64         `func:"input"`
	Uintptr    uintptr        `func:"input"`
	Float32    float32        `func:"input"`
	Float64    float64        `func:"input"`
	Complex64  complex64      `func:"input"`
	Complex128 complex128     `func:"input"`
	Array      [3]int         `func:"input"`
	Map        map[string]int `func:"input"`
	Ptr        *string        `func:"input"`
	Slice      []string       `func:"input"`
	String     string         `func:"input"`
	Struct     nested         `func:"input"`
	StructPtr  *nested        `func:"input"`
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
				got := hash.Compute(def)
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
			&def2{NotIO: "foo"}, // Field 'NotIO' does not have
			&def2{NotIO: "bar"}, // an input or output struct tag.
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := hash.Compute(tt.a)
			b := hash.Compute(tt.b)

			got := a == b
			if got != tt.same {
				t.Errorf("Hash() got same %t, want %t", a == b, tt.same)
			}
		})
	}
}

type def1 struct {
	resource.Definition
	FunctionName string `func:"input"`
}

type def2 struct {
	resource.Definition
	NotIO string // no input or output tag
}

// realistic is a resource roughly resembling an aws lambda resource
type realistic struct {
	resource.Definition

	// Inputs
	DeadLetterConfig *struct {
		TargetArn *string `func:"input"`
	} `func:"input"`
	Description   *string           `func:"input"`
	Environment   *env              `func:"input"`
	FunctionName  string            `func:"input"`
	Handler       string            `func:"input"`
	KMSKeyArn     *string           `func:"input"`
	Layers        []string          `func:"input"`
	MemorySize    *int64            `func:"input"`
	Publish       *bool             `func:"input"`
	Role          string            `func:"input"`
	Runtime       string            `func:"input"`
	Targs         map[string]string `func:"input"`
	Timeout       *int64            `func:"input"`
	TracingConfig *struct {
		Mode string `func:"input"`
	} `func:"input"`
	VPCConfig *struct {
		SecurityGroupIDs []string `func:"input"`
		SubnetIDs        []string `func:"input"`
	} `func:"input"`

	// Outputs
	CodeSHA256   string `func:"output"`
	CodeSize     int64  `func:"output"`
	FunctionARN  string `func:"output"`
	LastModified string `func:"output"`
	MasterARN    string `func:"output"`
	RevisionID   string `func:"output"`
	Version      string `func:"output"`
}

type env struct {
	Variables map[string]string `func:"input"`
}

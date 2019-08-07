package schema_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/func/func/resource/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type lambda struct {
	DeadLetterConfig *DeadLetterConfig `func:"input"`
	Description      *string           `func:"input"`
	Environment      Environment       `func:"input"`
	FunctionName     *string           `func:"input" name:"name"`
	Handler          *string           `func:"input,required"`
	KMSKeyARN        *string           `func:"input"`
	Layers           []string          `func:"input"`
	MemorySize       *int64            `func:"input"`
	Publish          *bool             `func:"input"`
	Region           string            `func:"input,required"`
	Role             *string           `func:"input,required"`
	Runtime          *string           `func:"input,required"`
	Tags             map[string]string `func:"input"`
	Timeout          *int64            `func:"input"`
	TracingConfig    *TracingConfig    `func:"input"`
	VPCConfig        *VPCConfig        `func:"input"`

	CodeSHA256   string    `func:"output"`
	CodeSize     int64     `func:"output"`
	FunctionArn  string    `func:"output"`
	LastModified time.Time `func:"output"`
	MasterARN    *string   `func:"output"`
	RevisionID   string    `func:"output" name:"rev"`
	Version      string    `func:"output"`

	NotIOField string // ignored
}

type DeadLetterConfig struct {
	TargetARN string
}
type Environment struct {
	Variables map[string]string
}
type TracingConfig struct {
	Mode string
}
type VPCConfig struct {
	SecurityGroupIDs []string
	SubnetIds        []string
}

var (
	str     string
	strptr  = &str
	boolean bool
	boolptr = &boolean
	i64     int64
	i64ptr  = &i64
)

func TestInputs(t *testing.T) {
	tests := []struct {
		name   string
		target reflect.Type
		want   map[string]schema.InputField
	}{
		{
			name:   "lambda",
			target: reflect.TypeOf(lambda{}),
			want: map[string]schema.InputField{
				"dead_letter_config": {Index: 0, Required: false, Type: reflect.TypeOf(&DeadLetterConfig{})},
				"description":        {Index: 1, Required: false, Type: reflect.TypeOf(strptr)},
				"environment":        {Index: 2, Required: false, Type: reflect.TypeOf(Environment{})},
				"name":               {Index: 3, Required: false, Type: reflect.TypeOf(strptr)},
				"handler":            {Index: 4, Required: true, Type: reflect.TypeOf(strptr)},
				"kms_key_arn":        {Index: 5, Required: false, Type: reflect.TypeOf(strptr)},
				"layers":             {Index: 6, Required: false, Type: reflect.TypeOf([]string{})},
				"memory_size":        {Index: 7, Required: false, Type: reflect.TypeOf(i64ptr)},
				"publish":            {Index: 8, Required: false, Type: reflect.TypeOf(boolptr)},
				"region":             {Index: 9, Required: true, Type: reflect.TypeOf(str)},
				"role":               {Index: 10, Required: true, Type: reflect.TypeOf(strptr)},
				"runtime":            {Index: 11, Required: true, Type: reflect.TypeOf(strptr)},
				"tags":               {Index: 12, Required: false, Type: reflect.TypeOf(map[string]string{})},
				"timeout":            {Index: 13, Required: false, Type: reflect.TypeOf(i64ptr)},
				"tracing_config":     {Index: 14, Required: false, Type: reflect.TypeOf(&TracingConfig{})},
				"vpc_config":         {Index: 15, Required: false, Type: reflect.TypeOf(&VPCConfig{})},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schema.Inputs(tt.target)
			opts := []cmp.Option{
				cmp.Comparer(func(a, b reflect.Type) bool { return a == b }),
				cmpopts.IgnoreUnexported(schema.InputField{}),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("Inputs() (-got, +want)\n%s", diff)
			}
		})
	}
}

func TestOutputs(t *testing.T) {
	tests := []struct {
		name   string
		target reflect.Type
		want   map[string]schema.OutputField
	}{
		{
			name:   "lambda",
			target: reflect.TypeOf(lambda{}),
			want: map[string]schema.OutputField{
				"code_sha256":   {Index: 16, Type: reflect.TypeOf(str)},
				"code_size":     {Index: 17, Type: reflect.TypeOf(i64)},
				"function_arn":  {Index: 18, Type: reflect.TypeOf(str)},
				"last_modified": {Index: 19, Type: reflect.TypeOf(time.Time{})},
				"master_arn":    {Index: 20, Type: reflect.TypeOf(strptr)},
				"rev":           {Index: 21, Type: reflect.TypeOf(str)},
				"version":       {Index: 22, Type: reflect.TypeOf(str)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schema.Outputs(tt.target)
			opts := []cmp.Option{
				cmp.Comparer(func(a, b reflect.Type) bool { return a == b }),
				cmpopts.IgnoreUnexported(schema.InputField{}),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("Inputs() (-got, +want)\n%s", diff)
			}
		})
	}
}

func TestInputs_panic(t *testing.T) {
	tests := []struct {
		name   string
		target reflect.Type
	}{
		{
			"Pointer",
			reflect.TypeOf(&struct{}{}),
		},
		{
			"Unexported",
			reflect.TypeOf(struct {
				value string `func:"input"` // nolint: unused
			}{}),
		},
		{
			"UnsupportedAttr",
			reflect.TypeOf(struct {
				Value string `func:"input,unsupported"` // nolint: unused
			}{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Did not panic")
				}
			}()
			schema.Inputs(tt.target)
		})
	}
}

func TestOutputs_panic(t *testing.T) {
	tests := []struct {
		name   string
		target reflect.Type
	}{
		{
			"Pointer",
			reflect.TypeOf(&struct{}{}),
		},
		{
			"Unexported",
			reflect.TypeOf(struct {
				value string `func:"output"` // nolint: unused
			}{}),
		},
		{
			"UnsupportedAttr",
			reflect.TypeOf(struct {
				Value string `func:"output,unsupported"`
			}{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Did not panic")
				}
			}()
			schema.Outputs(tt.target)
		})
	}
}

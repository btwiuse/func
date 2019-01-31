package resource_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
)

func TestFieldDirection(t *testing.T) {
	tests := []struct {
		dir    resource.FieldDirection
		input  bool
		output bool
		str    string
	}{
		{resource.Input, true, false, "input"},
		{resource.Output, false, true, "output"},
		{resource.IO, true, true, "input / output"},
		{resource.FieldDirection(0), false, false, "unknown"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.dir), func(t *testing.T) {
			if gotIn := tt.dir.Input(); gotIn != tt.input {
				t.Errorf("Input() = %t, want = %t", gotIn, tt.input)
			}
			if gotOut := tt.dir.Output(); gotOut != tt.output {
				t.Errorf("Output() = %t, want = %t", gotOut, tt.output)
			}
			if gotStr := tt.dir.String(); gotStr != tt.str {
				t.Errorf("String() = %q, want = %q", gotStr, tt.str)
			}
		})
	}
}

func TestFields(t *testing.T) {
	res := struct {
		Input  string `input:"in,extra"`
		Output int    `output:"out"`
	}{}

	tests := []struct {
		name   string
		target reflect.Type
		dir    resource.FieldDirection
		want   []resource.Field
	}{
		{
			"Empty",
			reflect.TypeOf(struct {
				Not, Input, Or, Output, Fields string
			}{}),
			resource.IO,
			nil,
		},
		{
			"Input",
			reflect.TypeOf(res),
			resource.Input,
			[]resource.Field{
				{
					Dir:   resource.Input,
					Name:  "in",
					Attr:  "extra",
					Type:  reflect.TypeOf("string"),
					Index: 0,
				},
			},
		},
		{
			"Output",
			reflect.TypeOf(res),
			resource.Output,
			[]resource.Field{
				{
					Dir:   resource.Output,
					Name:  "out",
					Type:  reflect.TypeOf(42),
					Index: 1,
				},
			},
		},
		{
			"IO",
			reflect.TypeOf(res),
			resource.IO,
			[]resource.Field{
				{
					Dir:   resource.Input,
					Name:  "in",
					Attr:  "extra",
					Type:  reflect.TypeOf("string"),
					Index: 0,
				},
				{
					Dir:   resource.Output,
					Name:  "out",
					Type:  reflect.TypeOf(42),
					Index: 1,
				},
			},
		},
		{
			"Neither",
			reflect.TypeOf(res),
			resource.FieldDirection(0),
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resource.Fields(tt.target, tt.dir)
			typeToString := cmp.Transformer("string", func(t reflect.Type) string {
				return t.String()
			})
			if diff := cmp.Diff(got, tt.want, typeToString); diff != "" {
				t.Errorf("Fields() (-got, +want)\n%s", diff)
			}
		})
	}
}

// Ensure useful panic messages
func TestFields_panic(t *testing.T) {
	tests := []struct {
		name   string
		target reflect.Type
		want   string
	}{
		{
			"InputOutput",
			reflect.TypeOf(struct {
				IO int `input:"foo" output:"bar"` // cannot have input and output on same field
			}{}),
			`Field "IO" is marked as an input and output; only one is allowed`,
		},
		{
			"UnexportedInput",
			reflect.TypeOf(struct {
				unexported int `input:"foo"` // nolint: unused
			}{}),
			`Unexporeted field "unexported" set as input`,
		},
		{
			"UnexportedOutput",
			reflect.TypeOf(struct {
				unexported int `output:"foo"` // nolint: unused
			}{}),
			`Unexporeted field "unexported" set as output`,
		},
		{
			"StructPointer",
			reflect.TypeOf(&struct{}{}),
			`Target must be a struct, not ptr`,
		},
		{
			"NotStruct",
			reflect.TypeOf("string"),
			`Target must be a struct, not string`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				err := recover()
				if err == nil {
					t.Fatal("Did not panic")
				}
				if got := fmt.Sprintf("%v", err); got != tt.want {
					t.Errorf("Panic()\nGot:  %v\nWant: %v", got, tt.want)
				}
			}()

			_ = resource.Fields(tt.target, resource.IO)
		})
	}
}

package schema_test

import (
	"reflect"
	"testing"

	"github.com/func/func/resource/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFields(t *testing.T) {
	tests := []struct {
		name        string
		target      reflect.Type
		wantInputs  schema.FieldSet
		wantOutputs schema.FieldSet
	}{
		{
			name: "Input",
			target: reflect.TypeOf(struct {
				Foo int `func:"input"`
			}{}),
			wantInputs: schema.FieldSet{
				"foo": {
					Index: 0,
					Type:  reflect.TypeOf(123),
				},
			},
			wantOutputs: nil,
		},
		{
			name: "Output",
			target: reflect.TypeOf(struct {
				Foo int `func:"output"`
			}{}),
			wantInputs: nil,
			wantOutputs: schema.FieldSet{
				"foo": {
					Index: 0,
					Type:  reflect.TypeOf(123),
				},
			},
		},
		{
			name: "Unexported",
			target: reflect.TypeOf(struct {
				foo int `func:"input"` // nolint: unused
			}{}),
			wantInputs:  nil,
			wantOutputs: nil,
		},
		{
			name: "CustomName",
			target: reflect.TypeOf(struct {
				Foo int    `func:"input" name:"bar"`
				Bar string `func:"input" name:"baz"`
			}{}),
			wantInputs: map[string]schema.Field{
				"bar": {
					Index: 0,
					Type:  reflect.TypeOf(123),
				},
				"baz": {
					Index: 1,
					Type:  reflect.TypeOf("string"),
				},
			},
		},
		{
			name: "Tag",
			target: reflect.TypeOf(struct {
				Foo int `func:"input" validate:"test"`
			}{}),
			wantInputs: map[string]schema.Field{
				"foo": {
					Index: 0,
					Type:  reflect.TypeOf(123),
					Tags: map[string]string{
						"validate": "test",
					},
				},
			},
		},
		{
			name: "Pointer",
			target: reflect.TypeOf(&struct {
				Foo int `func:"input"`
			}{}),
			wantInputs: schema.FieldSet{
				"foo": {
					Index: 0,
					Type:  reflect.TypeOf(123),
				},
			},
			wantOutputs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schema.Fields(tt.target)
			inputs := got.Inputs()
			outputs := got.Outputs()
			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(schema.Field{}),
				cmpopts.EquateEmpty(),
				cmp.Comparer(func(a, b reflect.Type) bool {
					return a == b
				}),
			}
			if diff := cmp.Diff(inputs, tt.wantInputs, opts...); diff != "" {
				t.Errorf("Diff() inputs (-got +want)\n%s", diff)
			}
			if diff := cmp.Diff(outputs, tt.wantOutputs, opts...); diff != "" {
				t.Errorf("Diff() outputs (-got +want)\n%s", diff)
			}
		})
	}
}

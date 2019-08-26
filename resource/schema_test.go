package resource_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"
)

func TestFields(t *testing.T) {
	tests := []struct {
		name        string
		target      reflect.Type
		wantInputs  resource.FieldSet
		wantOutputs resource.FieldSet
	}{
		{
			name: "Input",
			target: reflect.TypeOf(struct {
				Foo int `func:"input"`
			}{}),
			wantInputs: resource.FieldSet{
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
			wantOutputs: resource.FieldSet{
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
			wantInputs: map[string]resource.Field{
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
			wantInputs: map[string]resource.Field{
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
			wantInputs: resource.FieldSet{
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
			got := resource.Fields(tt.target)
			inputs := got.Inputs()
			outputs := got.Outputs()
			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(resource.Field{}),
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

func TestFieldSet_CtyType(t *testing.T) {
	tests := []struct {
		name   string
		fields resource.FieldSet
		want   cty.Type
	}{
		{
			"Simple",
			resource.FieldSet{
				"foo": {
					Index: 0,
					Type:  reflect.TypeOf("string"),
				},
			},
			cty.Object(map[string]cty.Type{
				"foo": cty.String,
			}),
		},
		{
			"Nested",
			resource.FieldSet{
				"foo": {
					Index: 0,
					Type: reflect.TypeOf(struct {
						Bar string
						Baz *int
					}{}),
				},
			},
			cty.Object(map[string]cty.Type{
				"foo": cty.Object(map[string]cty.Type{
					"bar": cty.String,
					"baz": cty.Number,
				}),
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fields.CtyType()
			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Type) bool {
					return a.Equals(b)
				}),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("CtyType() (-got +want)\n%s", diff)
			}
		})
	}
}

func ExampleFieldName_camel() {
	field := reflect.StructField{
		Name: "DeadLetterConfig",
	}
	got := resource.FieldName(field)
	fmt.Println(got)
	// Output: dead_letter_config
}

func ExampleFieldName_camel2() {
	field := reflect.StructField{
		Name: "KMSKeyArn",
	}
	got := resource.FieldName(field)
	fmt.Println(got)
	// Output: kms_key_arn
}

func ExampleFieldName_withoutCustom() {
	field := reflect.StructField{
		Name: "RestAPIID", // Will not split before ID
	}
	got := resource.FieldName(field)
	fmt.Println(got)
	// Output: rest_apiid
}

func ExampleFieldName_withCustom() {
	field := reflect.StructField{
		Name: "RestAPIID",
		Tag:  reflect.StructTag(`name:"rest_api_id"`),
	}
	got := resource.FieldName(field)
	fmt.Println(got)
	// Output: rest_api_id
}

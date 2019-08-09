package schema_test

import (
	"reflect"
	"testing"

	"github.com/func/func/resource/schema"
	"github.com/zclconf/go-cty/cty"
)

func TestImpliedType(t *testing.T) {
	tests := []struct {
		input reflect.Type
		want  cty.Type
	}{
		// Primitive types
		{reflect.TypeOf(int8(0)), cty.Number},
		{reflect.TypeOf(int8(0)), cty.Number},
		{reflect.TypeOf(int16(0)), cty.Number},
		{reflect.TypeOf(int32(0)), cty.Number},
		{reflect.TypeOf(int64(0)), cty.Number},
		{reflect.TypeOf(uint(0)), cty.Number},
		{reflect.TypeOf(uint8(0)), cty.Number},
		{reflect.TypeOf(uint16(0)), cty.Number},
		{reflect.TypeOf(uint32(0)), cty.Number},
		{reflect.TypeOf(uint64(0)), cty.Number},
		{reflect.TypeOf(float32(0)), cty.Number},
		{reflect.TypeOf(float64(0)), cty.Number},
		{reflect.TypeOf(true), cty.Bool},
		{reflect.TypeOf(""), cty.String},
		// Collection types
		{reflect.TypeOf([]string{}), cty.List(cty.String)},
		{reflect.TypeOf([][]string{}), cty.List(cty.List(cty.String))},
		{reflect.TypeOf(map[string]int{}), cty.Map(cty.Number)},
		{reflect.TypeOf(map[string]map[string]int{}), cty.Map(cty.Map(cty.Number))},
		{reflect.TypeOf(map[string][]int{}), cty.Map(cty.List(cty.Number))},
		// Struct
		{
			reflect.TypeOf(struct {
				Foo string
			}{}),
			cty.Object(map[string]cty.Type{
				"foo": cty.String,
			}),
		},
		// Pointers unwrapped
		{reflect.PtrTo(reflect.TypeOf("")), cty.String},
		{reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(""))), cty.String},
	}
	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			got := schema.ImpliedType(tt.input)
			if !got.Equals(tt.want) {
				t.Fatalf("ImpliedType()\ngot:   %#v\nwant:  %#v", got, tt.want)
			}
		})
	}
}

package ctyext_test

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/func/func/ctyext"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/set"
)

func TestToCtyValue(t *testing.T) {
	boolptr := func(v bool) *bool { return &v }
	strptr := func(v string) *string { return &v }

	tests := []struct {
		val       interface{}
		target    cty.Type
		fieldName func(field reflect.StructField) string
		want      cty.Value
	}{
		// Bool
		{
			val:    true,
			target: cty.Bool,
			want:   cty.True,
		},
		{
			val:    (*bool)(nil),
			target: cty.Bool,
			want:   cty.NullVal(cty.Bool),
		},
		{
			val:    boolptr(true),
			target: cty.Bool,
			want:   cty.True,
		},

		// String
		{
			val:    "hello",
			target: cty.String,
			want:   cty.StringVal("hello"),
		},
		{
			val:    strptr("hello"),
			target: cty.String,
			want:   cty.StringVal("hello"),
		},
		{
			val:    strptr("hello"),
			target: cty.String,
			want:   cty.StringVal("hello"),
		},
		{
			val:    (*string)(nil),
			target: cty.String,
			want:   cty.NullVal(cty.String),
		},
		{
			val:    nil, // any nil is convertable to a null of any type
			target: cty.String,
			want:   cty.NullVal(cty.String),
		},
		{
			val:    (*bool)(nil), // any nil is convertable to a null of any type
			target: cty.String,
			want:   cty.NullVal(cty.String),
		},

		// Number
		{
			val:    int(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    int8(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    int16(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    int32(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    int64(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    uint(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    uint8(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    uint16(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    uint32(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    uint64(1),
			target: cty.Number,
			want:   cty.NumberIntVal(1),
		},
		{
			val:    float32(1.5),
			target: cty.Number,
			want:   cty.NumberFloatVal(1.5),
		},
		{
			val:    float64(1.5),
			target: cty.Number,
			want:   cty.NumberFloatVal(1.5),
		},
		{
			val:    big.NewFloat(1.5),
			target: cty.Number,
			want:   cty.NumberFloatVal(1.5),
		},
		{
			val:    big.NewInt(5),
			target: cty.Number,
			want:   cty.NumberIntVal(5),
		},
		{
			val:    (*int)(nil),
			target: cty.Number,
			want:   cty.NullVal(cty.Number),
		},

		// Lists
		{
			val:    []int{},
			target: cty.List(cty.Number),
			want:   cty.ListValEmpty(cty.Number),
		},
		{
			val:    []int{1, 2},
			target: cty.List(cty.Number),
			want: cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
		},
		{
			val:    &[]int{1, 2},
			target: cty.List(cty.Number),
			want: cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
		},
		{
			val:    []int(nil),
			target: cty.List(cty.Number),
			want:   cty.NullVal(cty.List(cty.Number)),
		},
		{
			val:    (*[]int)(nil),
			target: cty.List(cty.Number),
			want:   cty.NullVal(cty.List(cty.Number)),
		},
		{
			val:    [2]int{1, 2},
			target: cty.List(cty.Number),
			want: cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
		},
		{
			val:    [0]int{},
			target: cty.List(cty.Number),
			want:   cty.ListValEmpty(cty.Number),
		},
		{
			val:    []int{},
			target: cty.Set(cty.Number),
			want:   cty.SetValEmpty(cty.Number),
		},

		// Sets
		{
			val:    []int{1, 2},
			target: cty.Set(cty.Number),
			want: cty.SetVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
		},
		{
			val:    []int{2, 2},
			target: cty.Set(cty.Number),
			want: cty.SetVal([]cty.Value{
				cty.NumberIntVal(2),
			}),
		},
		{
			val:    &[]int{1, 2},
			target: cty.Set(cty.Number),
			want: cty.SetVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
		},
		{
			val:    []int(nil),
			target: cty.Set(cty.Number),
			want:   cty.NullVal(cty.Set(cty.Number)),
		},
		{
			val:    (*[]int)(nil),
			target: cty.Set(cty.Number),
			want:   cty.NullVal(cty.Set(cty.Number)),
		},
		{
			val:    [2]int{1, 2},
			target: cty.Set(cty.Number),
			want: cty.SetVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
		},
		{
			val:    [0]int{},
			target: cty.Set(cty.Number),
			want:   cty.SetValEmpty(cty.Number),
		},
		{
			val:    set.NewSet(&testSetRules{}),
			target: cty.Set(cty.Number),
			want:   cty.SetValEmpty(cty.Number),
		},
		{
			val:    set.NewSetFromSlice(&testSetRules{}, []interface{}{1, 2}),
			target: cty.Set(cty.Number),
			want: cty.SetVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			}),
		},

		// Maps
		{
			val:    map[string]int{},
			target: cty.Map(cty.Number),
			want:   cty.MapValEmpty(cty.Number),
		},
		{
			val:    map[string]int{"one": 1, "two": 2},
			target: cty.Map(cty.Number),
			want: cty.MapVal(map[string]cty.Value{
				"one": cty.NumberIntVal(1),
				"two": cty.NumberIntVal(2),
			}),
		},

		// Objects
		{
			val:    struct{}{},
			target: cty.EmptyObject,
			want:   cty.EmptyObjectVal,
		},
		{
			val:    struct{ Ignored int }{1},
			target: cty.EmptyObject,
			want:   cty.EmptyObjectVal,
		},
		{
			val: struct{}{},
			target: cty.Object(map[string]cty.Type{
				"name": cty.String,
			}),
			want: cty.ObjectVal(map[string]cty.Value{
				"name": cty.NullVal(cty.String),
			}),
		},
		{
			val: struct {
				Name   string `cty:"name"`
				Number int    `cty:"number"`
			}{"Steven", 1},
			target: cty.Object(map[string]cty.Type{
				"name":   cty.String,
				"number": cty.Number,
			}),
			fieldName: func(field reflect.StructField) string {
				return field.Tag.Get("cty")
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("Steven"),
				"number": cty.NumberIntVal(1),
			}),
		},
		{
			val: struct {
				Name   string `cty:"name"`
				Number int
			}{"Steven", 1},
			target: cty.Object(map[string]cty.Type{
				"name":   cty.String,
				"number": cty.Number,
			}),
			fieldName: func(field reflect.StructField) string {
				return field.Tag.Get("cty")
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("Steven"),
				"number": cty.NullVal(cty.Number),
			}),
		},
		{
			val: map[string]interface{}{
				"name":   "Steven",
				"number": 1,
			},
			target: cty.Object(map[string]cty.Type{
				"name":   cty.String,
				"number": cty.Number,
			}),
			fieldName: func(field reflect.StructField) string {
				return field.Tag.Get("cty")
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("Steven"),
				"number": cty.NumberIntVal(1),
			}),
		},
		{
			val: map[string]interface{}{
				"number": 1,
			},
			target: cty.Object(map[string]cty.Type{
				"name":   cty.String,
				"number": cty.Number,
			}),
			fieldName: func(field reflect.StructField) string {
				return field.Tag.Get("cty")
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.NullVal(cty.String),
				"number": cty.NumberIntVal(1),
			}),
		},

		// Tuples
		{
			val:    []interface{}{},
			target: cty.EmptyTuple,
			want:   cty.EmptyTupleVal,
		},
		{
			val:    struct{}{},
			target: cty.EmptyTuple,
			want:   cty.EmptyTupleVal,
		},
		{
			val:    testTupleStruct{"Stephen", 23},
			target: cty.Tuple([]cty.Type{cty.String, cty.Number}),
			want: cty.TupleVal([]cty.Value{
				cty.StringVal("Stephen"),
				cty.NumberIntVal(23),
			}),
		},
		{
			val: []interface{}{1, 2, 3},
			target: cty.Tuple([]cty.Type{
				cty.Number,
				cty.Number,
				cty.Number,
			}),
			want: cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
		},
		{
			val: []interface{}{1, "hello", 3},
			target: cty.Tuple([]cty.Type{
				cty.Number,
				cty.String,
				cty.Number,
			}),
			want: cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.StringVal("hello"),
				cty.NumberIntVal(3),
			}),
		},
		{
			val:    []interface{}(nil),
			target: cty.Tuple([]cty.Type{cty.Number}),
			want:   cty.NullVal(cty.Tuple([]cty.Type{cty.Number})),
		},
	}

	for i, tt := range tests {
		from := fmt.Sprintf("%T", tt.val)
		to := tt.target.FriendlyName()
		name := fmt.Sprintf("%d_%s_%s", i, from, to)
		t.Run(name, func(t *testing.T) {
			got, err := ctyext.ToCtyValue(tt.val, tt.target, tt.fieldName)
			if err != nil {
				t.Fatalf("ToCtyValue() err = %v", err)
			}
			if got == cty.NilVal {
				t.Fatalf("ToCtyValue() NilVal with no error")
			}
			if !got.RawEquals(tt.want) {
				t.Errorf(`ToCtyValue()
input:       %#v
target type: %#v
got:         %#v
want:        %#v`, tt.val, tt.target, got, tt.want)
			}
		})
	}
}

func TestToCtyValue_error(t *testing.T) {
	tests := []struct {
		name      string
		val       interface{}
		target    cty.Type
		fieldName func(field reflect.StructField) string
	}{
		{
			name:   "TypeMismatch/bool",
			val:    "string",
			target: cty.Bool,
		},
		{
			name:   "TypeMismatch/string",
			val:    false,
			target: cty.String,
		},
		{
			name:   "TypeMismatch/number",
			val:    "string",
			target: cty.Number,
		},
		{
			name:   "TypeMismatch/map",
			val:    "string",
			target: cty.Map(cty.String),
		},
		{
			name:   "TypeMismatch/list",
			val:    "string",
			target: cty.List(cty.String),
		},
		{
			name:   "TypeMismatch/object",
			val:    "string",
			target: cty.Object(map[string]cty.Type{"foo": cty.String}),
		},
		{
			name:   "TypeMismatch/tuple",
			val:    "string",
			target: cty.Tuple([]cty.Type{cty.Number, cty.String}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ctyext.ToCtyValue(tt.val, tt.target, tt.fieldName)
			if err != nil {
				t.Logf("Got expected error: %v", err)
				return
			}
			t.Errorf("Error is nil")
		})
	}
}

type testSetRules struct{}

func (r testSetRules) Hash(v interface{}) int                         { return v.(int) }
func (r testSetRules) Equivalent(v1 interface{}, v2 interface{}) bool { return v1 == v2 }

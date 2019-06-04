package ctyext_test

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/func/func/ctyext"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestFromCtyValue(t *testing.T) {
	boolptr := func(v bool) *bool { return &v }
	strptr := func(v string) *string { return &v }
	intptr := func(v int) *int { return &v }

	type (
		boolAlias    bool
		stringAlias  string
		intAlias     int
		float32Alias float32
		float64Alias float64
		listIntAlias []int
		mapIntAlias  map[string]int
	)
	var (
		bigFloatType = reflect.TypeOf(big.Float{})
		bigIntType   = reflect.TypeOf(big.Int{})
	)

	tests := []struct {
		val       cty.Value
		target    reflect.Type
		fieldName func(field reflect.StructField) string
		want      interface{}
	}{
		// Bool
		{
			val:    cty.True,
			target: reflect.TypeOf(false),
			want:   true,
		},
		{
			val:    cty.False,
			target: reflect.TypeOf(false),
			want:   false,
		},
		{
			val:    cty.True,
			target: reflect.PtrTo(reflect.TypeOf(false)),
			want:   boolptr(true),
		},
		{
			val:    cty.NullVal(cty.Bool),
			target: reflect.PtrTo(reflect.TypeOf(false)),
			want:   (*bool)(nil),
		},
		{
			val:    cty.True,
			target: reflect.TypeOf((*boolAlias)(nil)).Elem(),
			want:   boolAlias(true),
		},

		// String
		{
			val:    cty.StringVal("hello"),
			target: reflect.TypeOf(""),
			want:   "hello",
		},
		{
			val:    cty.StringVal(""),
			target: reflect.TypeOf(""),
			want:   "",
		},
		{
			val:    cty.StringVal("hello"),
			target: reflect.PtrTo(reflect.TypeOf("")),
			want:   strptr("hello"),
		},
		{
			val:    cty.NullVal(cty.String),
			target: reflect.PtrTo(reflect.TypeOf("")),
			want:   (*string)(nil),
		},
		{
			val:    cty.StringVal("hello"),
			target: reflect.TypeOf((*stringAlias)(nil)).Elem(),
			want:   stringAlias("hello"),
		},

		// Number
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(int(0)),
			want:   int(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(int8(0)),
			want:   int8(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(int16(0)),
			want:   int16(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(int32(0)),
			want:   int32(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(int64(0)),
			want:   int64(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(uint(0)),
			want:   uint(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(uint8(0)),
			want:   uint8(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(uint16(0)),
			want:   uint16(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(uint32(0)),
			want:   uint32(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf(uint64(0)),
			want:   uint64(5),
		},
		{
			val:    cty.NumberFloatVal(1.5),
			target: reflect.TypeOf(float32(0)),
			want:   float32(1.5),
		},
		{
			val:    cty.NumberFloatVal(1.5),
			target: reflect.TypeOf(float64(0)),
			want:   float64(1.5),
		},
		{
			val:    cty.NumberFloatVal(1.5),
			target: reflect.PtrTo(bigFloatType),
			want:   big.NewFloat(1.5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.PtrTo(bigIntType),
			want:   big.NewInt(5),
		},
		{
			val:    cty.NumberIntVal(5),
			target: reflect.TypeOf((*intAlias)(nil)).Elem(),
			want:   intAlias(5),
		},
		{
			val:    cty.NumberFloatVal(1.5),
			target: reflect.TypeOf((*float32Alias)(nil)).Elem(),
			want:   float32Alias(1.5),
		},
		{
			val:    cty.NumberFloatVal(1.5),
			target: reflect.TypeOf((*float64Alias)(nil)).Elem(),
			want:   float64Alias(1.5),
		},

		// Lists
		{
			val:    cty.ListValEmpty(cty.Number),
			target: reflect.TypeOf(([]int)(nil)),
			want:   []int{},
		},
		{
			val:    cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(5)}),
			target: reflect.TypeOf(([]int)(nil)),
			want:   []int{1, 5},
		},
		{
			val:    cty.NullVal(cty.List(cty.Number)),
			target: reflect.TypeOf(([]int)(nil)),
			want:   ([]int)(nil),
		},
		{
			val:    cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(5)}),
			target: reflect.ArrayOf(2, reflect.TypeOf(0)),
			want:   [2]int{1, 5},
		},
		{
			val:    cty.ListValEmpty(cty.Number),
			target: reflect.ArrayOf(0, reflect.TypeOf(0)),
			want:   [0]int{},
		},
		{
			val:    cty.ListValEmpty(cty.Number),
			target: reflect.PtrTo(reflect.ArrayOf(0, reflect.TypeOf(0))),
			want:   &[0]int{},
		},
		{
			val:    cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(5)}),
			target: reflect.TypeOf((listIntAlias)(nil)),
			want:   listIntAlias{1, 5},
		},

		// Maps
		{
			val:    cty.MapValEmpty(cty.Number),
			target: reflect.TypeOf((map[string]int)(nil)),
			want:   map[string]int{},
		},
		{
			val: cty.MapVal(map[string]cty.Value{
				"one":  cty.NumberIntVal(1),
				"five": cty.NumberIntVal(5),
			}),
			target: reflect.TypeOf(map[string]int{}),
			want: map[string]int{
				"one":  1,
				"five": 5,
			},
		},
		{
			val:    cty.NullVal(cty.Map(cty.Number)),
			target: reflect.TypeOf((map[string]int)(nil)),
			want:   (map[string]int)(nil),
		},
		{
			val: cty.MapVal(map[string]cty.Value{
				"one":  cty.NumberIntVal(1),
				"five": cty.NumberIntVal(5),
			}),
			target: reflect.TypeOf(mapIntAlias(nil)),
			want: mapIntAlias{
				"one":  1,
				"five": 5,
			},
		},

		// Sets
		{
			val:    cty.SetValEmpty(cty.Number),
			target: reflect.TypeOf(([]int)(nil)),
			want:   []int{},
		},
		{
			val:    cty.SetVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(5)}),
			target: reflect.TypeOf(([]int)(nil)),
			want:   []int{1, 5},
		},
		{
			val:    cty.SetVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(5)}),
			target: reflect.TypeOf([2]int{}),
			want:   [2]int{1, 5},
		},

		// Objects
		{
			val:    cty.EmptyObjectVal,
			target: reflect.TypeOf(struct{}{}),
			want:   struct{}{},
		},
		{
			val: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("Stephen"),
			}),
			target: reflect.TypeOf(testStruct{}),
			fieldName: func(field reflect.StructField) string {
				return field.Tag.Get("cty")
			},
			want: testStruct{
				Name: "Stephen",
			},
		},
		{
			val: cty.ObjectVal(map[string]cty.Value{
				"city": cty.StringVal("New York"),
			}),
			target: reflect.TypeOf(testStruct{}),
			fieldName: func(field reflect.StructField) string {
				return field.Tag.Get("other")
			},
			want: testStruct{
				City: "New York",
			},
		},
		{
			val: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("Stephen"),
				"city":   cty.StringVal("New York"),
				"number": cty.NumberIntVal(12),
			}),
			target: reflect.TypeOf(testStruct{}),
			fieldName: func(field reflect.StructField) string {
				if c := field.Tag.Get("cty"); c != "" {
					return c
				}
				return field.Tag.Get("other")
			},
			want: testStruct{
				Name:   "Stephen",
				City:   "New York",
				Number: intptr(12),
			},
		},

		// Tuples
		{
			val:    cty.EmptyTupleVal,
			target: reflect.TypeOf(struct{}{}),
			want:   struct{}{},
		},
		{
			val: cty.TupleVal([]cty.Value{
				cty.StringVal("Stephen"),
				cty.NumberIntVal(5),
			}),
			target: reflect.TypeOf(testTupleStruct{}),
			want:   testTupleStruct{"Stephen", 5},
		},
	}

	for i, tt := range tests {
		from := tt.val.Type().FriendlyName()
		if tt.val.IsNull() {
			from = "null(" + from + ")"
		}
		to := tt.target.String()
		name := fmt.Sprintf("%d_%s_%s", i, from, to)
		t.Run(name, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Fatalf("Panic: %v", err)
				}
			}()
			target := reflect.New(tt.target)
			err := ctyext.FromCtyValue(tt.val, target.Interface(), tt.fieldName)
			if err != nil {
				t.Fatalf("FromCtyValue() err = %v", err)
			}

			got := target.Elem().Interface()

			opts := []cmp.Option{
				cmp.Transformer("String", func(v big.Float) string { return v.String() }),
				cmp.Transformer("String", func(v big.Int) string { return v.String() }),
			}
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Errorf("FromCtyValue() (-got +want)\n%s", diff)
			}
		})
	}
}

func TestFromCtyValue_error(t *testing.T) {
	tests := []struct {
		name      string
		val       cty.Value
		target    reflect.Type
		fieldName func(field reflect.StructField) string
	}{
		{
			name:   "TypeMismatch/bool",
			val:    cty.True,
			target: reflect.TypeOf("string"),
		},
		{
			name:   "TypeMismatch/string",
			val:    cty.StringVal(""),
			target: reflect.TypeOf(123),
		},
		{
			name:   "TypeMismatch/number",
			val:    cty.NumberIntVal(123),
			target: reflect.TypeOf("string"),
		},
		{
			name:   "TypeMismatch/map",
			val:    cty.MapVal(map[string]cty.Value{"a": cty.StringVal("a")}),
			target: reflect.TypeOf("string"),
		},
		{
			name:   "TypeMismatch/list",
			val:    cty.ListVal([]cty.Value{cty.StringVal("a")}),
			target: reflect.TypeOf("string"),
		},
		{
			name:   "TypeMismatch/object",
			val:    cty.ObjectVal(map[string]cty.Value{"a": cty.StringVal("a")}),
			target: reflect.TypeOf("string"),
		},
		{
			name:   "TypeMismatch/tuple",
			val:    cty.TupleVal([]cty.Value{cty.StringVal("Stephen"), cty.NumberIntVal(5)}),
			target: reflect.TypeOf("string"),
		},
		{
			name:   "NullArray",
			val:    cty.NullVal(cty.List(cty.Number)),
			target: reflect.TypeOf([1]string{}),
		},
		{
			name:   "ArrayLengthMismatch",
			val:    cty.ListVal([]cty.Value{cty.StringVal("a")}),
			target: reflect.TypeOf([10]string{}),
		},
		{
			name:   "TupleLengthMistmatch",
			val:    cty.TupleVal([]cty.Value{cty.StringVal("Stephen")}),
			target: reflect.TypeOf(testTupleStruct{}),
		},
		{
			name: "UnsupportedAttr",
			val: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("name"),
				"foo":  cty.StringVal("foo"),
			}),
			target: reflect.TypeOf(testStruct{}),
			fieldName: func(field reflect.StructField) string {
				return field.Tag.Get("cty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := reflect.New(tt.target)
			err := ctyext.FromCtyValue(tt.val, target.Interface(), tt.fieldName)
			if err != nil {
				t.Logf("Got expected error: %v", err)
				return
			}
			t.Errorf("Error is nil")
		})
	}
}

func TestFromCtyValue_panic(t *testing.T) {
	tests := []struct {
		name   string
		target reflect.Value
	}{
		{"NotPointer", reflect.ValueOf(testStruct{})},
		{"NilPointer", reflect.ValueOf((*testStruct)(nil))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Did not panic")
				}
			}()
			val := cty.EmptyObjectVal
			nop := func(reflect.StructField) string { return "" }
			_ = ctyext.FromCtyValue(val, tt.target, nop)
		})
	}
}

type testStruct struct {
	Name   string `cty:"name"`
	City   string `other:"city"`
	Number *int   `cty:"number"`
}

type testTupleStruct struct {
	Name   string
	Number int
}

package attr

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	. "github.com/aws/aws-sdk-go-v2/service/dynamodb" // Dot import to remove a lot of redundant dynamodb.
	"github.com/func/func/ctyext"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestFromBool(t *testing.T) {
	tests := []struct {
		val  bool
		want AttributeValue
	}{
		{true, AttributeValue{BOOL: aws.Bool(true)}},
		{false, AttributeValue{BOOL: aws.Bool(false)}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%t", i, tt.val), func(t *testing.T) {
			got := FromBool(tt.val)
			compare(t, got, tt.want)
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		attr    AttributeValue
		want    bool
		wantErr bool
	}{
		{AttributeValue{BOOL: aws.Bool(true)}, true, false},
		{AttributeValue{BOOL: aws.Bool(false)}, false, false},
		{AttributeValue{BOOL: nil}, false, true}, // Not set
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.attr), func(t *testing.T) {
			got, err := ToBool(tt.attr)
			compareErr(t, err, tt.wantErr)
			compare(t, got, tt.want)
		})
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		val  string
		want AttributeValue
	}{
		{"", AttributeValue{NULL: aws.Bool(true)}}, // String cannot be empty
		{"foo", AttributeValue{S: aws.String("foo")}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%s", i, tt.val), func(t *testing.T) {
			got := FromString(tt.val)
			compare(t, got, tt.want)
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		attr    AttributeValue
		want    string
		wantErr bool
	}{
		{AttributeValue{S: aws.String("")}, "", false},
		{AttributeValue{S: aws.String("foo")}, "foo", false},
		{AttributeValue{S: nil}, "", true}, // Not set
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.attr), func(t *testing.T) {
			got, err := ToString(tt.attr)
			compareErr(t, err, tt.wantErr)
			compare(t, got, tt.want)
		})
	}
}

func TestFromInt64(t *testing.T) {
	tests := []struct {
		val  int64
		want AttributeValue
	}{
		{0, AttributeValue{N: aws.String("0")}},
		{123, AttributeValue{N: aws.String("123")}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%d", i, tt.val), func(t *testing.T) {
			got := FromInt64(tt.val)
			compare(t, got, tt.want)
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		attr    AttributeValue
		want    int64
		wantErr bool
	}{
		{AttributeValue{N: aws.String("0")}, 0, false},
		{AttributeValue{N: aws.String("123")}, 123, false},
		{AttributeValue{N: nil}, 0, true},               // Not set
		{AttributeValue{N: aws.String("abc")}, 0, true}, // Not a number
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.attr), func(t *testing.T) {
			got, err := ToInt64(tt.attr)
			compareErr(t, err, tt.wantErr)
			compare(t, got, tt.want)
		})
	}
}

func TestFromFloat64(t *testing.T) {
	tests := []struct {
		val  float64
		want AttributeValue
	}{
		{0, AttributeValue{N: aws.String("0")}},
		{123.45, AttributeValue{N: aws.String("123.45")}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.val), func(t *testing.T) {
			got := FromFloat64(tt.val)
			compare(t, got, tt.want)
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		attr    AttributeValue
		want    float64
		wantErr bool
	}{
		{AttributeValue{N: aws.String("0")}, 0, false},
		{AttributeValue{N: aws.String("123.456789")}, 123.456789, false},
		{AttributeValue{N: nil}, 0, true},               // Not set
		{AttributeValue{N: aws.String("abc")}, 0, true}, // Not a number
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.attr), func(t *testing.T) {
			got, err := ToFloat64(tt.attr)
			compareErr(t, err, tt.wantErr)
			compare(t, got, tt.want)
		})
	}
}

func TestFromStringSlice(t *testing.T) {
	tests := []struct {
		val  []string
		want AttributeValue
	}{
		{nil, AttributeValue{L: []AttributeValue{}}},
		{[]string{}, AttributeValue{L: []AttributeValue{}}},
		{[]string{"a"}, AttributeValue{L: []AttributeValue{{S: aws.String("a")}}}},
		{[]string{"a", "b"}, AttributeValue{L: []AttributeValue{{S: aws.String("a")}, {S: aws.String("b")}}}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.val), func(t *testing.T) {
			got := FromStringSlice(tt.val)
			compare(t, got, tt.want)
		})
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		attr    AttributeValue
		want    []string
		wantErr bool
	}{
		{AttributeValue{L: []AttributeValue{}}, nil, false},
		{AttributeValue{L: []AttributeValue{{S: aws.String("a")}}}, []string{"a"}, false},
		{AttributeValue{L: nil}, nil, false},                       // Not set -> empty slice
		{AttributeValue{L: []AttributeValue{{S: nil}}}, nil, true}, // String not set
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.attr), func(t *testing.T) {
			got, err := ToStringSlice(tt.attr)
			compareErr(t, err, tt.wantErr)
			compare(t, got, tt.want)
		})
	}
}

func TestFromStringSet(t *testing.T) {
	tests := []struct {
		val  []string
		want AttributeValue
	}{
		{nil, AttributeValue{NULL: aws.Bool(true)}},
		{[]string{}, AttributeValue{NULL: aws.Bool(true)}},
		{[]string{"a"}, AttributeValue{SS: []string{"a"}}},
		{[]string{"a", "b"}, AttributeValue{SS: []string{"a", "b"}}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.val), func(t *testing.T) {
			got := FromStringSet(tt.val)
			compare(t, got, tt.want)
		})
	}
}

func TestToStringSet(t *testing.T) {
	tests := []struct {
		attr AttributeValue
		want []string
	}{
		{AttributeValue{SS: []string{}}, []string{}},
		{AttributeValue{SS: []string{"a"}}, []string{"a"}},
		{AttributeValue{SS: nil}, nil},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%v", i, tt.attr), func(t *testing.T) {
			got := ToStringSet(tt.attr)
			compare(t, got, tt.want)
		})
	}
}

func TestFromCtyValue(t *testing.T) {
	tests := []struct {
		val  cty.Value
		want AttributeValue
	}{
		// Booleans
		{cty.True, AttributeValue{BOOL: aws.Bool(true)}},
		{cty.False, AttributeValue{BOOL: aws.Bool(false)}},
		{cty.NullVal(cty.Bool), AttributeValue{NULL: aws.Bool(true)}},

		// Strings
		{cty.StringVal(""), AttributeValue{NULL: aws.Bool(true)}}, // String cannot be empty
		{cty.StringVal("foo"), AttributeValue{S: aws.String("foo")}},
		{cty.NullVal(cty.String), AttributeValue{NULL: aws.Bool(true)}},

		// Numbers
		{cty.NumberIntVal(0), AttributeValue{N: aws.String("0")}},
		{cty.NumberIntVal(123), AttributeValue{N: aws.String("123")}},
		{cty.NumberUIntVal(1), AttributeValue{N: aws.String("1")}},
		{cty.NumberUIntVal(234), AttributeValue{N: aws.String("234")}},
		{cty.NumberFloatVal(1.23), AttributeValue{N: aws.String("1.23")}},
		{cty.NullVal(cty.Number), AttributeValue{NULL: aws.Bool(true)}},

		// Lists
		{cty.ListValEmpty(cty.String), AttributeValue{L: []AttributeValue{}}},
		{
			cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			AttributeValue{L: []AttributeValue{{S: aws.String("a")}, {S: aws.String("b")}}},
		},
		{
			cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)}),
			AttributeValue{L: []AttributeValue{{N: aws.String("1")}, {N: aws.String("2")}}},
		},
		{
			cty.ListVal([]cty.Value{cty.NumberUIntVal(1), cty.NumberUIntVal(2)}),
			AttributeValue{L: []AttributeValue{{N: aws.String("1")}, {N: aws.String("2")}}},
		},
		{
			cty.ListVal([]cty.Value{cty.NumberFloatVal(1.23), cty.NumberFloatVal(2.34)}),
			AttributeValue{L: []AttributeValue{{N: aws.String("1.23")}, {N: aws.String("2.34")}}},
		},
		{
			cty.ListVal([]cty.Value{cty.True, cty.False}),
			AttributeValue{L: []AttributeValue{{BOOL: aws.Bool(true)}, {BOOL: aws.Bool(false)}}},
		},
		{cty.NullVal(cty.List(cty.String)), AttributeValue{NULL: aws.Bool(true)}},
		{cty.NullVal(cty.List(cty.Number)), AttributeValue{NULL: aws.Bool(true)}},
		{cty.NullVal(cty.List(cty.Bool)), AttributeValue{NULL: aws.Bool(true)}},

		// Maps
		{cty.MapValEmpty(cty.String), AttributeValue{M: map[string]AttributeValue{}}},
		{
			cty.MapVal(map[string]cty.Value{"a": cty.StringVal("A"), "b": cty.StringVal("B")}),
			AttributeValue{M: map[string]AttributeValue{"a": {S: aws.String("A")}, "b": {S: aws.String("B")}}},
		},
		{
			cty.MapVal(map[string]cty.Value{"a": cty.NumberIntVal(1), "b": cty.NumberIntVal(2)}),
			AttributeValue{M: map[string]AttributeValue{"a": {N: aws.String("1")}, "b": {N: aws.String("2")}}},
		},
		{
			cty.MapVal(map[string]cty.Value{"a": cty.NumberUIntVal(1), "b": cty.NumberUIntVal(2)}),
			AttributeValue{M: map[string]AttributeValue{"a": {N: aws.String("1")}, "b": {N: aws.String("2")}}},
		},
		{
			cty.MapVal(map[string]cty.Value{"a": cty.NumberFloatVal(1.23), "b": cty.NumberFloatVal(2.34)}),
			AttributeValue{M: map[string]AttributeValue{"a": {N: aws.String("1.23")}, "b": {N: aws.String("2.34")}}},
		},
		{cty.NullVal(cty.Map(cty.String)), AttributeValue{NULL: aws.Bool(true)}},
		{cty.NullVal(cty.Map(cty.Number)), AttributeValue{NULL: aws.Bool(true)}},
		{cty.NullVal(cty.Map(cty.Bool)), AttributeValue{NULL: aws.Bool(true)}},

		// Sets
		{cty.SetValEmpty(cty.Bool), AttributeValue{L: []AttributeValue{}}},
		{
			// String set has native DynamoDB type.
			cty.SetVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			AttributeValue{SS: []string{"a", "b"}},
		},
		{
			// Number set has native DynamoDB type.
			cty.SetVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)}),
			AttributeValue{NS: []string{"1", "2"}},
		},
		{
			cty.SetVal([]cty.Value{cty.False, cty.True}),
			AttributeValue{L: []AttributeValue{{BOOL: aws.Bool(false)}, {BOOL: aws.Bool(true)}}},
		},
		{cty.SetValEmpty(cty.String), AttributeValue{NULL: aws.Bool(true)}}, // String set cannot be empty
		{cty.SetValEmpty(cty.Number), AttributeValue{NULL: aws.Bool(true)}}, // Number set cannot be empty
		{cty.NullVal(cty.Set(cty.String)), AttributeValue{NULL: aws.Bool(true)}},
		{cty.NullVal(cty.Set(cty.Number)), AttributeValue{NULL: aws.Bool(true)}},
		{cty.NullVal(cty.Set(cty.Bool)), AttributeValue{NULL: aws.Bool(true)}},

		// Objects
		{cty.EmptyObjectVal, AttributeValue{M: map[string]AttributeValue{}}},
		{
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("A"),
				"b": cty.NumberIntVal(-1),
				"c": cty.NumberUIntVal(1),
				"d": cty.NumberFloatVal(2.34),
				"e": cty.True,
				"f": cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			}),
			AttributeValue{M: map[string]AttributeValue{
				"a": {S: aws.String("A")},
				"b": {N: aws.String("-1")},
				"c": {N: aws.String("1")},
				"d": {N: aws.String("2.34")},
				"e": {BOOL: aws.Bool(true)},
				"f": {L: []AttributeValue{{S: aws.String("a")}, {S: aws.String("b")}}},
			}},
		},
		{cty.NullVal(cty.Object(map[string]cty.Type{"a": cty.String})), AttributeValue{NULL: aws.Bool(true)}},

		// Tuples
		{cty.EmptyTupleVal, AttributeValue{L: []AttributeValue{}}},
		{
			cty.TupleVal([]cty.Value{cty.StringVal("a"), cty.False}),
			AttributeValue{L: []AttributeValue{{S: aws.String("a")}, {BOOL: aws.Bool(false)}}},
		},
		{
			cty.TupleVal([]cty.Value{cty.NumberIntVal(1), cty.NumberUIntVal(2), cty.NumberFloatVal(3.45)}),
			AttributeValue{L: []AttributeValue{{N: aws.String("1")}, {N: aws.String("2")}, {N: aws.String("3.45")}}},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%#v", i, tt.val), func(t *testing.T) {
			got := FromCtyValue(tt.val)
			compare(t, got, tt.want)
		})
	}
}

func TestToCtyValue(t *testing.T) {
	tests := []struct {
		attr    AttributeValue
		typ     cty.Type
		want    cty.Value
		wantErr bool
	}{
		// B: Binary
		// not supported

		// BOOL	Boolean
		{AttributeValue{BOOL: aws.Bool(true)}, cty.Bool, cty.BoolVal(true), false},
		{AttributeValue{BOOL: aws.Bool(true)}, cty.String, cty.NilVal, true}, // Type does not match
		{AttributeValue{BOOL: nil}, cty.Bool, cty.NilVal, true},

		// BS: Boolean set
		// not supported

		// L: List
		{
			AttributeValue{L: []AttributeValue{{S: aws.String("a")}, {S: aws.String("b")}}},
			cty.List(cty.String),
			cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{{N: aws.String("1")}, {N: aws.String("2.34")}}},
			cty.List(cty.Number),
			cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.MustParseNumberVal("2.34")}),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{{BOOL: aws.Bool(true)}, {BOOL: aws.Bool(false)}}},
			cty.List(cty.Bool),
			cty.ListVal([]cty.Value{cty.True, cty.False}),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{{S: aws.String("str")}, {BOOL: aws.Bool(false)}}},
			cty.Tuple([]cty.Type{cty.String, cty.Bool}),
			cty.TupleVal([]cty.Value{cty.StringVal("str"), cty.False}),
			false,
		},
		{
			AttributeValue{L: nil},
			cty.List(cty.String),
			cty.ListValEmpty(cty.String),
			false,
		},
		{
			AttributeValue{L: nil},
			cty.Tuple([]cty.Type{cty.String, cty.Number}),
			cty.EmptyTupleVal,
			false,
		},
		{
			AttributeValue{L: []AttributeValue{{S: aws.String("str")}, {BOOL: aws.Bool(false)}}},
			cty.Tuple([]cty.Type{cty.Bool, cty.Number}),
			cty.NilVal,
			true, // Types do not match
		},
		{
			AttributeValue{L: []AttributeValue{{S: aws.String("str")}, {BOOL: aws.Bool(false)}}},
			cty.List(cty.String),
			cty.NilVal,
			true, // Mixed types cannot be added to list
		},
		{
			AttributeValue{L: []AttributeValue{{S: aws.String("str")}, {BOOL: aws.Bool(false)}}},
			cty.String,
			cty.NilVal,
			true, // Cannot assign list to string
		},

		// M: Map
		{
			AttributeValue{M: map[string]AttributeValue{"a": {S: aws.String("A")}, "b": {S: aws.String("B")}}},
			cty.Map(cty.String),
			cty.MapVal(map[string]cty.Value{"a": cty.StringVal("A"), "b": cty.StringVal("B")}),
			false,
		},
		{
			AttributeValue{M: map[string]AttributeValue{"a": {S: aws.String("A")}, "b": {BOOL: aws.Bool(true)}}},
			cty.Object(map[string]cty.Type{"a": cty.String, "b": cty.Bool}),
			cty.ObjectVal(map[string]cty.Value{"a": cty.StringVal("A"), "b": cty.True}),
			false,
		},
		{
			AttributeValue{M: nil},
			cty.Map(cty.String),
			cty.MapValEmpty(cty.String),
			false,
		},
		{
			AttributeValue{M: nil},
			cty.Object(map[string]cty.Type{"a": cty.String, "b": cty.Bool}),
			cty.EmptyObjectVal,
			false,
		},
		{
			AttributeValue{M: map[string]AttributeValue{"a": {S: aws.String("A")}, "b": {BOOL: aws.Bool(true)}}},
			cty.Map(cty.String),
			cty.NilVal,
			true, // Mixed
		},
		{
			AttributeValue{M: map[string]AttributeValue{"a": {S: aws.String("A")}, "b": {N: aws.String("123")}}},
			cty.Object(map[string]cty.Type{"a": cty.String, "b": cty.Bool}),
			cty.NilVal,
			true, // Type does not match
		},

		// N: Number
		{AttributeValue{N: aws.String("1")}, cty.Number, cty.NumberIntVal(1), false},
		{
			AttributeValue{N: aws.String("3.14159265358979323846264338327950288419716939937510582097494459")},
			cty.Number,
			cty.MustParseNumberVal("3.14159265358979323846264338327950288419716939937510582097494459"),
			false,
		},
		{AttributeValue{N: aws.String("123")}, cty.String, cty.NilVal, true}, // Type does not match
		{AttributeValue{N: nil}, cty.Number, cty.NilVal, true},

		// NS: Number set
		{
			AttributeValue{NS: []string{"1", "2"}},
			cty.List(cty.Number),
			cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)}),
			false,
		},
		{
			AttributeValue{NS: []string{"2", "3"}},
			cty.Set(cty.Number),
			cty.SetVal([]cty.Value{cty.NumberIntVal(2), cty.NumberIntVal(3)}),
			false,
		},
		{
			AttributeValue{NS: nil},
			cty.Set(cty.Number),
			cty.SetValEmpty(cty.Number),
			false,
		},
		{
			AttributeValue{NS: nil},
			cty.List(cty.Number),
			cty.ListValEmpty(cty.Number),
			false,
		},
		{
			AttributeValue{NS: []string{"x"}},
			cty.Set(cty.Number),
			cty.NilVal,
			true, // Cannot parse number. Unlikely that DynamoDB would return this.
		},
		{
			AttributeValue{NS: []string{"x"}},
			cty.List(cty.Number),
			cty.NilVal,
			true, // Cannot parse number. Unlikely that DynamoDB would return this.
		},
		{
			AttributeValue{NS: []string{"3", "4"}},
			cty.List(cty.String),
			cty.NilVal,
			true, // Number set cannot be assigned to list of strings
		},
		{
			AttributeValue{NS: []string{"3", "4"}},
			cty.Set(cty.String),
			cty.NilVal,
			true, // Number set cannot be assigned to set of strings
		},
		{
			AttributeValue{NS: []string{"3", "4"}},
			cty.String,
			cty.NilVal,
			true, // Type does not match
		},

		// NULL
		{AttributeValue{NULL: aws.Bool(true)}, cty.String, cty.NullVal(cty.String), false},
		{AttributeValue{NULL: aws.Bool(true)}, cty.Number, cty.NullVal(cty.Number), false},
		{AttributeValue{NULL: aws.Bool(true)}, cty.List(cty.String), cty.NullVal(cty.List(cty.String)), false},

		// S: String
		{AttributeValue{S: aws.String("a")}, cty.String, cty.StringVal("a"), false},
		{AttributeValue{S: aws.String("a")}, cty.Number, cty.NilVal, true}, // Type does not match
		{AttributeValue{S: nil}, cty.String, cty.NilVal, true},

		// SS: String set
		{
			AttributeValue{SS: []string{"a", "b"}},
			cty.List(cty.String),
			cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			false,
		},
		{
			AttributeValue{SS: []string{"b", "c"}},
			cty.Set(cty.String),
			cty.SetVal([]cty.Value{cty.StringVal("b"), cty.StringVal("c")}),
			false,
		},
		{
			AttributeValue{SS: nil},
			cty.Set(cty.String),
			cty.SetValEmpty(cty.String),
			false,
		},
		{
			AttributeValue{SS: nil},
			cty.List(cty.String),
			cty.ListValEmpty(cty.String),
			false,
		},
		{
			AttributeValue{SS: []string{"d", "e"}},
			cty.List(cty.Number),
			cty.NilVal,
			true, // String set cannot be assigned to list of numbers
		},
		{
			AttributeValue{SS: []string{"e", "f"}},
			cty.Set(cty.Number),
			cty.NilVal,
			true, // String set cannot be assigned to set of numbers
		},
		{
			AttributeValue{SS: []string{"f", "g"}},
			cty.String,
			cty.NilVal,
			true, // Type does not match
		},
	}
	for i, tt := range tests {
		name := fmt.Sprintf("%d_%s", i, strings.ReplaceAll(tt.attr.String(), " ", ""))
		t.Run(name, func(t *testing.T) {
			got, err := ToCtyValue(tt.attr, tt.typ)
			compareErr(t, err, tt.wantErr)
			compare(t, got, tt.want)
		})
	}
}

func TestFromCtyPath(t *testing.T) {
	tests := []struct {
		path cty.Path
		want AttributeValue
	}{
		{nil, AttributeValue{L: []AttributeValue{}}},
		{cty.GetAttrPath("a"), AttributeValue{L: []AttributeValue{
			{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
		}}},
		{cty.IndexPath(cty.NumberIntVal(1)), AttributeValue{L: []AttributeValue{
			{M: map[string]AttributeValue{"Index": {N: aws.String("1")}}},
		}}},
		{cty.IndexPath(cty.StringVal("a")), AttributeValue{L: []AttributeValue{
			{M: map[string]AttributeValue{"Index": {S: aws.String("a")}}},
		}}},
		{cty.GetAttrPath("a").GetAttr("b"), AttributeValue{L: []AttributeValue{
			{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
			{M: map[string]AttributeValue{"Attr": {S: aws.String("b")}}},
		}}},
		{cty.GetAttrPath("a").Index(cty.NumberIntVal(1)), AttributeValue{L: []AttributeValue{
			{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
			{M: map[string]AttributeValue{"Index": {N: aws.String("1")}}},
		}}},
		{cty.GetAttrPath("a").Index(cty.StringVal("b")), AttributeValue{L: []AttributeValue{
			{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
			{M: map[string]AttributeValue{"Index": {S: aws.String("b")}}},
		}}},
	}
	for i, tt := range tests {
		name := fmt.Sprintf("%d_%s", i, ctyext.PathString(tt.path))
		t.Run(name, func(t *testing.T) {
			got := FromCtyPath(tt.path)
			compare(t, got, tt.want)
		})
	}
}

func TestToCtyPath(t *testing.T) {
	tests := []struct {
		attr    AttributeValue
		want    cty.Path
		wantErr bool
	}{
		{
			AttributeValue{L: []AttributeValue{}},
			nil,
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
			}},
			cty.GetAttrPath("a"),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Index": {N: aws.String("0")}}},
			}},
			cty.IndexPath(cty.NumberIntVal(0)),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Index": {S: aws.String("a")}}},
			}},
			cty.IndexPath(cty.StringVal("a")),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
				{M: map[string]AttributeValue{"Attr": {S: aws.String("b")}}},
			}},
			cty.GetAttrPath("a").GetAttr("b"),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
				{M: map[string]AttributeValue{"Index": {N: aws.String("2")}}},
			}},
			cty.GetAttrPath("a").Index(cty.NumberIntVal(2)),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
				{M: map[string]AttributeValue{"Index": {S: aws.String("b")}}},
			}},
			cty.GetAttrPath("a").Index(cty.StringVal("b")),
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Attr": {S: nil}}},
			}},
			nil,
			true, // Attr name not set
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Index": {S: nil, N: nil}}},
			}},
			nil,
			true, // Index is not a string or a number
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Attr": {S: aws.String("a")}}},
				{M: map[string]AttributeValue{"Index": {N: aws.String("b")}}},
			}},
			nil,
			true, // Cannot parse b as number
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Other": {S: aws.String("a")}}},
			}},
			nil,
			true, // "Other" is not a valid path type
		},
		{
			AttributeValue{L: []AttributeValue{
				{S: aws.String("a")},
			}},
			nil,
			true, // List does contain maps
		},
		{AttributeValue{L: nil}, nil, true}, // List not set
	}
	for i, tt := range tests {
		name := fmt.Sprintf("%d_%s", i, strings.ReplaceAll(tt.attr.String(), " ", ""))
		t.Run(name, func(t *testing.T) {
			got, err := ToCtyPath(tt.attr)
			compareErr(t, err, tt.wantErr)
			compare(t, got, tt.want)
		})
	}
}

func TestFromExpression(t *testing.T) {
	tests := []struct {
		expr resource.Expression
		want AttributeValue
	}{
		{
			resource.Expression{
				resource.ExprLiteral{Value: cty.StringVal("foo")},
			},
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Literal": {S: aws.String("foo")}}},
			}},
		},
		{
			resource.Expression{
				resource.ExprReference{Path: cty.GetAttrPath("abc")},
			},
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Reference": FromCtyPath(cty.GetAttrPath("abc"))}},
			}},
		},
		{
			resource.Expression{
				resource.ExprLiteral{Value: cty.StringVal("foo")},
				resource.ExprReference{Path: cty.GetAttrPath("bar").Index(cty.NumberIntVal(2))},
				resource.ExprLiteral{Value: cty.StringVal("baz")},
			},
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Literal": {S: aws.String("foo")}}},
				{M: map[string]AttributeValue{"Reference": FromCtyPath(cty.GetAttrPath("bar").Index(cty.NumberIntVal(2)))}},
				{M: map[string]AttributeValue{"Literal": {S: aws.String("baz")}}},
			}},
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := FromExpression(tt.expr)
			compare(t, got, tt.want)
		})
	}
}

func TestToExpression(t *testing.T) {
	tests := []struct {
		attr    AttributeValue
		want    resource.Expression
		wantErr bool
	}{
		{AttributeValue{L: nil}, nil, false},
		{AttributeValue{L: []AttributeValue{}}, nil, false},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Literal": {S: aws.String("foo")}}},
			}},
			resource.Expression{
				resource.ExprLiteral{Value: cty.StringVal("foo")},
			},
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Reference": FromCtyPath(cty.GetAttrPath("abc"))}},
			}},
			resource.Expression{
				resource.ExprReference{Path: cty.GetAttrPath("abc")},
			},
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Literal": {S: aws.String("foo")}}},
				{M: map[string]AttributeValue{"Reference": FromCtyPath(cty.GetAttrPath("bar").Index(cty.NumberIntVal(2)))}},
				{M: map[string]AttributeValue{"Literal": {S: aws.String("baz")}}},
			}},
			resource.Expression{
				resource.ExprLiteral{Value: cty.StringVal("foo")},
				resource.ExprReference{Path: cty.GetAttrPath("bar").Index(cty.NumberIntVal(2))},
				resource.ExprLiteral{Value: cty.StringVal("baz")},
			},
			false,
		},
		{
			AttributeValue{L: []AttributeValue{
				{S: aws.String("foo")},
			}},
			nil,
			true, // List must contain maps
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Other": {S: aws.String("foo")}}},
			}},
			nil,
			true, // Literal or Expression must be set
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Literal": {N: aws.String("1")}}},
			}},
			nil,
			true, // Literal must be a string
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Reference": {L: nil}}},
			}},
			nil,
			true, // Reference must be a set
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Reference": {L: []AttributeValue{}}}},
			}},
			nil,
			true, // Reference must be a set
		},
		{
			AttributeValue{L: []AttributeValue{
				{M: map[string]AttributeValue{"Reference": {N: aws.String("1")}}},
			}},
			nil,
			true, // Reference must be a string
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := ToExpression(tt.attr)
			compareErr(t, err, tt.wantErr)
			compare(t, got, tt.want)
		})
	}
}

func compare(t *testing.T, got, want interface{}) {
	t.Helper()
	opts := []cmp.Option{
		cmp.Comparer(func(a, b cty.Path) bool { return a.Equals(b) }),
		cmp.Comparer(func(a, b resource.Expression) bool { return a.Equals(b) }),
		cmp.Transformer("GoString", func(v cty.Value) string { return v.GoString() }),
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func compareErr(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Fatalf("err = %v, want err = %t", err, wantErr)
	}
}

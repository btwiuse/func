package resource

import (
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"
)

// CtyType converts a reflect type to the cty type system.
//
// The function is essentially the same as gocty.ImpliedType, except nested
// structs do not require a cty struct tag. Instead, Fields() is used to get
// the fields of the nested struct.
//
// Panics if the type cannot be converted. In practice this only applies to
// more complex types, such as functions and slices.
func CtyType(t reflect.Type) cty.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		return Fields(t).CtyType()
	case reflect.Slice, reflect.Array:
		return cty.List(CtyType(t.Elem()))
	case reflect.Map:
		return cty.Map(CtyType(t.Elem()))
	case reflect.Bool:
		return cty.Bool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cty.Number
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cty.Number
	case reflect.Float32, reflect.Float64:
		return cty.Number
	case reflect.String:
		return cty.String
	default:
		// This should not happen; all supported conversions should be listed above.
		panic(fmt.Sprintf("no type for %s", t))
	}
}

package apigatewaypatch

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/pkg/errors"
)

// A Field is a field to compare. If the value has changed, a PatchOperation is
// created.
type Field struct {
	Name string // Name of the field in the struct. Dot.Path for nested path.
	Path string // Resulting path in PatchOperation.

	// Optional modifier for modifying the patch operations for a given field, if the
	// default resolver is not sufficient. The function is called with the ops
	// for the current field. The modifier is only called if there are
	// operations to apply to the field.
	Modifier func(input []apigateway.PatchOperation) (output []apigateway.PatchOperation)
}

// Resolve resolves differences between two inputs and creates a list of AWS
// APIGateway patch operations.
//
// Both prevVersion and nextVersion must be structs or pointers to structs, and
// must have the same type.
//
// The results are returned in the order the fields are defined in the struct.
func Resolve(prevVersion, nextVersion interface{}, fields ...Field) ([]apigateway.PatchOperation, error) {
	prev := reflect.Indirect(reflect.ValueOf(prevVersion))
	next := reflect.Indirect(reflect.ValueOf(nextVersion))
	if prev.Type() != next.Type() {
		// This is always a bug
		panic("Types do not match")
	}

	var ops []apigateway.PatchOperation // nolint: prealloc
	indices := fieldIndices(next.Type())

	for _, f := range fields {
		var fieldOps []apigateway.PatchOperation

		index, ok := indices[f.Name]
		if !ok {
			return nil, errors.Errorf("Field %s does not exist on struct", f.Name)
		}

		srcVal := fieldByIndex(prev, index)
		dstVal := fieldByIndex(next, index)

		if srcVal.IsValid() &&
			dstVal.IsValid() &&
			reflect.DeepEqual(srcVal.Interface(), dstVal.Interface()) {
			// No changes
			continue
		}

		t := next.Type().FieldByIndex(index).Type

		switch {
		case t.Kind() == reflect.Slice:
			fieldOps = append(fieldOps, sliceOps(srcVal, dstVal, f.Path)...)
		case t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Slice:
			fieldOps = append(fieldOps, sliceOps(srcVal.Elem(), dstVal.Elem(), f.Path)...)
		default:
			fieldOps = append(fieldOps, apigateway.PatchOperation{
				Op:    apigateway.OpReplace,
				Path:  strptr(f.Path),
				Value: opValue(dstVal),
			})
		}

		if len(fieldOps) == 0 {
			// No nested changes
			continue
		}

		if f.Modifier != nil {
			fieldOps = f.Modifier(fieldOps)
		}

		ops = append(ops, fieldOps...)
	}

	return ops, nil
}

// fieldByIndex returns a value at a nested index in a struct. It works the
// same way as reflect.Value.FieldByIndex, except it does not panic if a nil
// pointer is encountered.
//
// If the nested field is not found or a field on the path is nil, an empty
// reflect.Value is returned. To assert if a value was empty, use .IsValid() on
// the returned Value.
//
// Panics if v is not a struct.
func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for i, idx := range index {
		if i > 0 {
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					// nil pointer; return empty value
					return reflect.Value{}
				}
				v = v.Elem()
			}
		}
		if idx >= v.NumField() {
			return reflect.Value{}
		}
		v = v.Field(idx)
	}
	return v
}

// sliceOps resolves operations to apply on a slice. The order in the slices is
// not taken into account.
func sliceOps(src, dst reflect.Value, prefix string) []apigateway.PatchOperation {
	prev := sliceValues(src, prefix)
	next := sliceValues(dst, prefix)
	sort.Strings(prev)
	sort.Strings(next)

	n := len(prev)
	if n < len(next) {
		n = len(next)
	}

	var ops []apigateway.PatchOperation

	for i := 0; i < n; i++ {
		var a string
		if len(prev) > i {
			a = prev[i]
		}

		var b string
		if len(next) > i {
			b = next[i]
		}

		if a == b {
			// identical
			continue
		}

		// remove
		if a != "" && !contains(a, next) {
			ops = append(ops, apigateway.PatchOperation{
				Op:   apigateway.OpRemove,
				Path: strptr(a),
			})
		}

		// add
		if b != "" && !contains(b, prev) {
			ops = append(ops, apigateway.PatchOperation{
				Op:   apigateway.OpAdd,
				Path: strptr(b),
			})
		}
	}

	return ops
}

func sliceValues(slice reflect.Value, prefix string) []string {
	if !slice.IsValid() {
		return nil
	}
	vals := make([]string, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		v := slice.Index(i)
		str := fmt.Sprintf("%s/%s", prefix, *opValue(v))
		vals[i] = str
	}
	return vals
}

func contains(needle string, haystack []string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func jsonpath(path string) string {
	path = strings.Replace(path, "~", "~0", -1)
	path = strings.Replace(path, "/", "~1", -1)
	return path
}

func opValue(v reflect.Value) *string {
	switch v.Kind() {
	case reflect.String:
		str := v.String()
		if str == "" {
			return nil
		}
		str = jsonpath(str)
		return &str
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return opValue(v.Elem())
	default:
		str := fmt.Sprintf("%v", v.Interface())
		str = jsonpath(str)
		return &str
	}
}

// fieldIndices returns a map of field name to field index.
func fieldIndices(t reflect.Type, prefix ...string) map[string][]int {
	m := make(map[string][]int)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			// Unexported
			continue
		}
		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			for k, v := range fieldIndices(ft, append(prefix, f.Name)...) {
				m[k] = append(f.Index, v...)
			}
			continue
		}
		name := append(prefix, f.Name)
		s := strings.Join(name, ".")
		m[s] = f.Index
	}
	return m
}

func strptr(str string) *string { return &str }

package ctyext

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/zclconf/go-cty/cty"
)

// FieldNameFunc is called from FromCtyValue. The function should return the
// field name to read from the cty.Value for a given struct field when decoding
// an object. If the function returns an empty string, the field is ignored.
type FieldNameFunc = func(field reflect.StructField) string

// FromCtyValue assigns a cty.Value to an interface{}, which must be a pointer,
// using a fixed set of conversion rules.
//
// The implementation is largely inspired by gocty.FromCtyValue, with the
// addition of the fieldName function that allows configuring the field name to
// use when converting objects to structs.
//
// When setting fields in a struct, fields present in the cty.Value are set. If
// the corresponding field is not set in the cty.Value, it is ignored.
//
// In case an error occurs, a PathError is returned.
func FromCtyValue(val cty.Value, target interface{}, fieldName FieldNameFunc) error {
	tVal := reflect.ValueOf(target)
	if tVal.Kind() != reflect.Ptr {
		panic("target value is not a pointer")
	}
	if tVal.IsNil() {
		panic("target value is nil pointer")
	}
	return fromCtyValue(val, tVal, nil, fieldName)
}

func fromCtyValue(val cty.Value, target reflect.Value, path cty.Path, fieldName FieldNameFunc) error {
	ty := val.Type()

	for target.Kind() == reflect.Ptr {
		if val.IsNull() && !ty.IsListType() && !ty.IsMapType() {
			return nil
		}
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}

	switch ty {
	case cty.Bool:
		return setBool(val, target, path)
	case cty.Number:
		return setNumber(val, target, path)
	case cty.String:
		return setString(val, target, path)
	}

	switch {
	case ty.IsListType(), ty.IsSetType():
		return setList(val, target, path, fieldName)
	case ty.IsMapType():
		return setMap(val, target, path, fieldName)
	case ty.IsObjectType():
		return setObject(val, target, path, fieldName)
	case ty.IsTupleType():
		return setTuple(val, target, path, fieldName)
	case ty.IsCapsuleType():
		return PathError{Path: path, Err: fmt.Errorf("capsule types not supported")}
	}

	// We should never reach this point - all types should be covered above.
	return PathError{Path: path, Err: fmt.Errorf("unsupported source type %#v", val.Type())}
}

func setBool(val cty.Value, target reflect.Value, path cty.Path) error {
	if target.Kind() != reflect.Bool {
		return PathError{Path: path, Err: fmt.Errorf("target is not a boolean")}
	}
	target.SetBool(val.True())
	return nil
}

func setNumber(val cty.Value, target reflect.Value, path cty.Path) error {
	bf := val.AsBigFloat()

	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i64, _ := bf.Int64()
		target.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u64, _ := bf.Uint64()
		target.SetUint(u64)
		return nil
	case reflect.Float32, reflect.Float64:
		f64, _ := bf.Float64()
		target.SetFloat(f64)
		return nil
	case reflect.Struct:
		return setBigFloat(bf, target, path)
	default:
		return PathError{Path: path, Err: fmt.Errorf("number cannot be assigned to %s", target.Type().String())}
	}
}

var bigFloatType = reflect.TypeOf(big.Float{})
var bigIntType = reflect.TypeOf(big.Int{})

func setBigFloat(bf *big.Float, target reflect.Value, path cty.Path) error {
	switch {
	case bigFloatType.ConvertibleTo(target.Type()):
		target.Set(reflect.ValueOf(bf).Elem().Convert(target.Type()))
		return nil
	case bigIntType.ConvertibleTo(target.Type()):
		bi, accuracy := bf.Int(nil)
		if accuracy != big.Exact {
			return PathError{Path: path, Err: fmt.Errorf("value must be a whole number")}
		}
		target.Set(reflect.ValueOf(bi).Elem().Convert(target.Type()))
		return nil
	}
	return PathError{Path: path, Err: fmt.Errorf("big float cannot be assigned to %s", target.Type().String())}
}

func setString(val cty.Value, target reflect.Value, path cty.Path) error {
	if target.Kind() != reflect.String {
		return PathError{Path: path, Err: fmt.Errorf("target is %s, not string", target.Kind())}
	}
	target.SetString(val.AsString())
	return nil
}

func setObject(val cty.Value, target reflect.Value, path cty.Path, fieldName FieldNameFunc) error {
	if target.Kind() != reflect.Struct {
		return PathError{Path: path, Err: fmt.Errorf("target is %s, not struct", target.Kind())}
	}

	attrTypes := val.Type().AttributeTypes()

	ty := target.Type()
	targetFields := make(map[string]int)
	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)
		attrName := fieldName(field)
		if attrName != "" {
			targetFields[attrName] = i
		}
	}

	path = append(path, nil)

	for k := range attrTypes {
		path[len(path)-1] = cty.GetAttrStep{Name: k}

		fieldIdx, exists := targetFields[k]
		if !exists {
			return PathError{Path: path, Err: fmt.Errorf("unsupported attribute %q", k)}
		}

		ev := val.GetAttr(k)

		targetField := target.Field(fieldIdx)
		if err := fromCtyValue(ev, targetField, path, fieldName); err != nil {
			return err
		}
	}

	return nil
}

func setTuple(val cty.Value, target reflect.Value, path cty.Path, fieldName FieldNameFunc) error {
	if target.Kind() != reflect.Struct {
		return PathError{Path: path, Err: fmt.Errorf("target is %s, not struct", target.Kind())}
	}

	elemTypes := val.Type().TupleElementTypes()
	fieldCount := target.Type().NumField()

	if fieldCount != len(elemTypes) {
		return PathError{Path: path, Err: fmt.Errorf("a tuple of %d elements is required", fieldCount)}
	}

	path = append(path, nil)

	for i := range elemTypes {
		path[len(path)-1] = cty.IndexStep{Key: cty.NumberIntVal(int64(i))}

		ev := val.Index(cty.NumberIntVal(int64(i)))

		targetField := target.Field(i)
		err := fromCtyValue(ev, targetField, path, fieldName)
		if err != nil {
			return err
		}
	}

	return nil
}

func setMap(val cty.Value, target reflect.Value, path cty.Path, fieldName FieldNameFunc) error {
	if target.Kind() != reflect.Map {
		return PathError{Path: path, Err: fmt.Errorf("target is %s, not map", target.Kind())}
	}

	if val.IsNull() {
		target.Set(reflect.Zero(target.Type()))
		return nil
	}

	tv := reflect.MakeMap(target.Type())
	et := target.Type().Elem()

	path = append(path, nil)

	var err error
	val.ForEachElement(func(k cty.Value, v cty.Value) bool {
		path[len(path)-1] = cty.IndexStep{Key: k}
		mapVal := reflect.New(et)
		if err = fromCtyValue(v, mapVal, path, fieldName); err != nil {
			return true
		}
		key := k.AsString()
		tv.SetMapIndex(reflect.ValueOf(key), mapVal.Elem())
		return false
	})
	target.Set(tv)
	return err
}

func setList(val cty.Value, target reflect.Value, path cty.Path, fieldName FieldNameFunc) error {
	if target.Kind() != reflect.Slice && target.Kind() != reflect.Array {
		return PathError{Path: path, Err: fmt.Errorf("target is %s, not slice or array", target.Kind())}
	}

	i := 0
	switch target.Kind() {
	case reflect.Slice:
		if val.IsNull() {
			target.Set(reflect.Zero(target.Type()))
			return nil
		}
		n := val.LengthInt()
		tv := reflect.MakeSlice(target.Type(), n, n)
		path = append(path, nil)
		var err error
		val.ForEachElement(func(key cty.Value, val cty.Value) bool {
			path[len(path)-1] = cty.IndexStep{Key: cty.NumberIntVal(int64(i))}
			targetElem := tv.Index(i)
			if err := fromCtyValue(val, targetElem, path, fieldName); err != nil {
				return true
			}
			i++
			return false
		})
		target.Set(tv)
		return err
	case reflect.Array:
		if val.IsNull() {
			return PathError{Path: path, Err: fmt.Errorf("null value is not allowed")}
		}
		n := val.LengthInt()
		if n != target.Len() {
			return PathError{Path: path, Err: fmt.Errorf("must be a list of length %d", target.Len())}
		}
		path = append(path, nil)
		var err error
		val.ForEachElement(func(key cty.Value, val cty.Value) bool {
			path[len(path)-1] = cty.IndexStep{Key: cty.NumberIntVal(int64(i))}
			targetElem := target.Index(i)
			if err := fromCtyValue(val, targetElem, path, fieldName); err != nil {
				return true
			}
			i++
			return false
		})
		return err
	}
	return PathError{Path: path, Err: fmt.Errorf("target must be a slice or array")}
}

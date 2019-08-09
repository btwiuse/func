package ctyext

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/set"
)

// ToCtyValue produces a cty.Value from a Go value. The result will conform to
// the given type, or an error will be returned if this is not possible.
//
// The implementation is largely inspired by gocty.ToCtyValue, with the
// addition of the fieldName function that allows configuring the field name to
// use when converting structs to objects.
//
// The target type serves as a hint to resolve ambiguities in the mapping. For
// example, the Go type set.Set tells us that the value is a set but does not
// describe the set's element type.
func ToCtyValue(val interface{}, ty cty.Type, fieldName FieldNameFunc) (cty.Value, error) {
	return toCtyValue(reflect.ValueOf(val), ty, nil, fieldName)
}

func toCtyValue(val reflect.Value, ty cty.Type, path cty.Path, fieldName FieldNameFunc) (cty.Value, error) {
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	if !val.IsValid() {
		return cty.NullVal(ty), nil
	}

	switch ty {
	case cty.Bool:
		return getBool(val, path)
	case cty.String:
		return getString(val, path)
	case cty.Number:
		return getNumber(val, path)
	case cty.DynamicPseudoType:
		return cty.NilVal, PathError{Path: path, Err: fmt.Errorf("dynamic types not supported")}
	}

	switch {
	case ty.IsListType():
		return getList(val, ty.ElementType(), path, fieldName)
	case ty.IsMapType():
		return getMap(val, ty.ElementType(), path, fieldName)
	case ty.IsObjectType():
		return getObject(val, ty.AttributeTypes(), path, fieldName)
	case ty.IsSetType():
		return getSet(val, ty.ElementType(), path, fieldName)
	case ty.IsTupleType():
		return getTuple(val, ty.TupleElementTypes(), path, fieldName)
	case ty.IsCapsuleType():
		return cty.NilVal, PathError{Path: path, Err: fmt.Errorf("capsule types not supported")}
	}

	// We should never fall out here
	return cty.NilVal, PathError{Path: path, Err: fmt.Errorf("unsupported target type %#v", ty)}
}

func getBool(val reflect.Value, path cty.Path) (cty.Value, error) {
	if val.Kind() != reflect.Bool {
		return cty.NilVal, PathError{Path: path, Err: fmt.Errorf("value is %s, not bool", val.Kind())}
	}
	return cty.BoolVal(val.Bool()), nil
}

func getString(val reflect.Value, path cty.Path) (cty.Value, error) {
	if val.Kind() != reflect.String {
		return cty.NilVal, PathError{Path: path, Err: fmt.Errorf("value is %s, not string", val.Kind())}
	}
	return cty.StringVal(val.String()), nil
}

func getNumber(val reflect.Value, path cty.Path) (cty.Value, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cty.NumberIntVal(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cty.NumberUIntVal(val.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return cty.NumberFloatVal(val.Float()), nil
	case reflect.Struct:
		if val.Type().AssignableTo(bigIntType) {
			bigInt := val.Interface().(big.Int)
			bigFloat := (&big.Float{}).SetInt(&bigInt)
			val = reflect.ValueOf(*bigFloat)
		}

		if val.Type().AssignableTo(bigFloatType) {
			bigFloat := val.Interface().(big.Float)
			return cty.NumberVal(&bigFloat), nil
		}
	}
	return cty.NilVal, path.NewErrorf("can't convert Go %s to number", val.Kind())
}

func getList(val reflect.Value, ety cty.Type, path cty.Path, fieldName FieldNameFunc) (cty.Value, error) {
	switch val.Kind() {
	case reflect.Slice:
		if val.IsNil() {
			return cty.NullVal(cty.List(ety)), nil
		}
		fallthrough
	case reflect.Array:
		if val.Len() == 0 {
			return cty.ListValEmpty(ety), nil
		}

		path = append(path, nil)

		vals := make([]cty.Value, val.Len())
		for i := range vals {
			var err error
			path[len(path)-1] = cty.IndexStep{
				Key: cty.NumberIntVal(int64(i)),
			}
			vals[i], err = toCtyValue(val.Index(i), ety, path, fieldName)
			if err != nil {
				return cty.NilVal, err
			}
		}
		return cty.ListVal(vals), nil
	}
	return cty.NilVal, path.NewErrorf("can't convert Go %s to %s", val.Kind(), cty.List(ety).FriendlyName())
}

func getMap(val reflect.Value, ety cty.Type, path cty.Path, fieldName FieldNameFunc) (cty.Value, error) {
	if val.Kind() != reflect.Map {
		return cty.NilVal, PathError{Path: path, Err: fmt.Errorf("value is %s, not map", val.Kind())}
	}
	if val.Len() == 0 {
		return cty.MapValEmpty(ety), nil
	}

	keyType := val.Type().Key()
	if keyType.Kind() != reflect.String {
		return cty.NilVal, path.NewErrorf("can't convert Go map with key type %s; key type must be string", keyType)
	}

	path = append(path, nil)

	vals := make(map[string]cty.Value, val.Len())
	for _, kv := range val.MapKeys() {
		k := kv.String()
		var err error
		path[len(path)-1] = cty.IndexStep{
			Key: cty.StringVal(k),
		}
		vals[k], err = toCtyValue(val.MapIndex(reflect.ValueOf(k)), ety, path, fieldName)
		if err != nil {
			return cty.NilVal, err
		}
	}

	return cty.MapVal(vals), nil
}

func getObject(val reflect.Value, attr map[string]cty.Type, path cty.Path, fieldName FieldNameFunc) (cty.Value, error) {
	if len(attr) == 0 {
		return cty.EmptyObjectVal, nil
	}
	path = append(path, nil)

	switch val.Kind() {
	case reflect.Map:
		keyType := val.Type().Key()
		if keyType.Kind() != reflect.String {
			return cty.NilVal, path.NewErrorf("can't convert Go map with key type %s; key type must be string", keyType)
		}

		haveKeys := make(map[string]struct{}, val.Len())
		for _, kv := range val.MapKeys() {
			haveKeys[kv.String()] = struct{}{}
		}

		vals := make(map[string]cty.Value, len(attr))
		for k, at := range attr {
			var err error
			path[len(path)-1] = cty.GetAttrStep{Name: k}

			if _, have := haveKeys[k]; !have {
				vals[k] = cty.NullVal(at)
				continue
			}

			vals[k], err = toCtyValue(val.MapIndex(reflect.ValueOf(k)), at, path, fieldName)
			if err != nil {
				return cty.NilVal, err
			}
		}
		return cty.ObjectVal(vals), nil
	case reflect.Struct:
		ty := val.Type()
		attrFields := make(map[string]int)
		for i := 0; i < ty.NumField(); i++ {
			field := ty.Field(i)
			attrName := fieldName(field)
			if attrName != "" {
				attrFields[attrName] = i
			}
		}

		vals := make(map[string]cty.Value, len(attr))
		for k, at := range attr {
			path[len(path)-1] = cty.GetAttrStep{
				Name: k,
			}

			if fieldIdx, have := attrFields[k]; have {
				var err error
				vals[k], err = toCtyValue(val.Field(fieldIdx), at, path, fieldName)
				if err != nil {
					return cty.NilVal, err
				}
			} else {
				vals[k] = cty.NullVal(at)
			}
		}

		return cty.ObjectVal(vals), nil
	}
	return cty.NilVal, path.NewErrorf("can't convert Go %s to %s", val.Kind(), cty.Object(attr).FriendlyName())
}

func getSet(val reflect.Value, ety cty.Type, path cty.Path, fieldName FieldNameFunc) (cty.Value, error) {
	var vals []cty.Value

	switch val.Kind() {
	case reflect.Slice:
		if val.IsNil() {
			return cty.NullVal(cty.Set(ety)), nil
		}
		fallthrough
	case reflect.Array:
		if val.Len() == 0 {
			return cty.SetValEmpty(ety), nil
		}

		vals = make([]cty.Value, val.Len())
		for i := range vals {
			var err error
			vals[i], err = toCtyValue(val.Index(i), ety, path, fieldName)
			if err != nil {
				return cty.NilVal, err
			}
		}
		return cty.SetVal(vals), nil
	case reflect.Struct:
		rawSet := val.Interface().(set.Set)
		inVals := rawSet.Values()

		if len(inVals) == 0 {
			return cty.SetValEmpty(ety), nil
		}

		vals = make([]cty.Value, len(inVals))
		for i := range inVals {
			var err error
			vals[i], err = toCtyValue(reflect.ValueOf(inVals[i]), ety, path, fieldName)
			if err != nil {
				return cty.NilVal, err
			}
		}
		return cty.SetVal(vals), nil
	}
	return cty.NilVal, path.NewErrorf("can't convert Go %s to %s", val.Kind(), cty.Set(ety).FriendlyName())
}

func getTuple(val reflect.Value, elemTypes []cty.Type, path cty.Path, fieldName FieldNameFunc) (cty.Value, error) {
	path = append(path, nil)
	switch val.Kind() {
	case reflect.Slice:
		if val.IsNil() {
			return cty.NullVal(cty.Tuple(elemTypes)), nil
		}

		if val.Len() != len(elemTypes) {
			return cty.NilVal, path.NewErrorf("wrong number of elements %d; need %d", val.Len(), len(elemTypes))
		}

		if len(elemTypes) == 0 {
			return cty.EmptyTupleVal, nil
		}

		vals := make([]cty.Value, len(elemTypes))
		for i, ety := range elemTypes {
			var err error

			path[len(path)-1] = cty.IndexStep{Key: cty.NumberIntVal(int64(i))}

			vals[i], err = toCtyValue(val.Index(i), ety, path, fieldName)
			if err != nil {
				return cty.NilVal, err
			}
		}
		return cty.TupleVal(vals), nil
	case reflect.Struct:
		fieldCount := val.Type().NumField()
		if fieldCount != len(elemTypes) {
			return cty.NilVal, path.NewErrorf("wrong number of struct fields %d; need %d", fieldCount, len(elemTypes))
		}

		if len(elemTypes) == 0 {
			return cty.EmptyTupleVal, nil
		}

		vals := make([]cty.Value, len(elemTypes))
		for i, ety := range elemTypes {
			var err error

			path[len(path)-1] = cty.IndexStep{
				Key: cty.NumberIntVal(int64(i)),
			}

			vals[i], err = toCtyValue(val.Field(i), ety, path, fieldName)
			if err != nil {
				return cty.NilVal, err
			}
		}
		return cty.TupleVal(vals), nil
	}
	return cty.NilVal, path.NewErrorf("can't convert Go %s to %s", val.Kind(), cty.Tuple(elemTypes).FriendlyName())
}

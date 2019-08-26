package attr

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/func/func/resource"
	"github.com/zclconf/go-cty/cty"
)

// NotSetError is returned when trying to read a value and the corresponding
// field is not set in the DynamoDB attribute value.
type NotSetError struct {
	Target cty.Type
	Key    string
}

func (e NotSetError) Error() string {
	return fmt.Sprintf("cannot convert to %s: key %s not set", e.Target.FriendlyName(), e.Key)
}

// FromBool creates a boolean attribute.
func FromBool(b bool) dynamodb.AttributeValue {
	return dynamodb.AttributeValue{BOOL: &b}
}

// ToBool returns the boolean value from an attribute.
func ToBool(attr dynamodb.AttributeValue) (bool, error) {
	b := attr.BOOL
	if b == nil {
		return false, fmt.Errorf("boolean value not set")
	}
	return *b, nil
}

// FromString creates a string attribute.
func FromString(str string) dynamodb.AttributeValue {
	return dynamodb.AttributeValue{S: &str}
}

// ToString returns the string from an attribute.
func ToString(attr dynamodb.AttributeValue) (string, error) {
	str := attr.S
	if str == nil {
		return "", fmt.Errorf("string value not set")
	}
	return *str, nil
}

// FromInt64 creates a numeric attribute from a 64 bit integer.
func FromInt64(v int64) dynamodb.AttributeValue {
	str := strconv.Itoa(int(v))
	return dynamodb.AttributeValue{N: &str}
}

// ToInt64 parses a numeric attribute as a 64 bit integer.
func ToInt64(attr dynamodb.AttributeValue) (int64, error) {
	n := attr.N
	if n == nil {
		return 0, fmt.Errorf("number value not set")
	}
	v, err := strconv.ParseInt(*n, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// FromFloat64 creates a numeric attribute from a 64 bit float.
func FromFloat64(v float64) dynamodb.AttributeValue {
	str := fmt.Sprintf("%v", v) // use %v instead of %f to exclude unnecessary 0's at the end
	return dynamodb.AttributeValue{N: &str}
}

// ToFloat64 parses a numeric attribute as a 64 bit float.
func ToFloat64(attr dynamodb.AttributeValue) (float64, error) {
	n := attr.N
	if n == nil {
		return 0, fmt.Errorf("number value not set")
	}
	f, err := strconv.ParseFloat(*n, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}

// FromStringSlice creates an attribute with a sorted list of strings.
func FromStringSlice(list []string) dynamodb.AttributeValue {
	values := make([]dynamodb.AttributeValue, len(list))
	for i, v := range list {
		values[i] = dynamodb.AttributeValue{S: aws.String(v)}
	}
	return dynamodb.AttributeValue{L: values}
}

// ToStringSlice returns a sorted list of strings.
func ToStringSlice(attr dynamodb.AttributeValue) ([]string, error) {
	if len(attr.L) == 0 {
		return nil, nil
	}
	values := make([]string, len(attr.L))
	for i, a := range attr.L {
		str := a.S
		if str == nil {
			return nil, fmt.Errorf("value %d is not a string", i)
		}
		values[i] = *a.S
	}
	return values, nil
}

// FromStringSet creates an unsorted set of strings.
//
// The values are set in the same order as the incoming list, but values may be
// returned from DynamoDB in another order.
func FromStringSet(list []string) dynamodb.AttributeValue {
	if len(list) == 0 {
		panic("String set cannot be empty")
	}
	return dynamodb.AttributeValue{SS: list}
}

// ToStringSet returns a set of strings from an attribute.
//
// As the set is not sorted, the order is not deterministic.
func ToStringSet(attr dynamodb.AttributeValue) []string {
	return attr.SS
}

// FromCtyValue encodes a value from the cty type system to a DynamoDB attribute.
//
// In case the value contains nested objects such as objects, they are nested
// into the returned attribute.
//
// Unknown, dynamic and capsule values are not supported.
func FromCtyValue(v cty.Value) dynamodb.AttributeValue {
	if v.IsNull() {
		return dynamodb.AttributeValue{NULL: aws.Bool(true)}
	}
	if !v.IsKnown() {
		panic("Value is not known")
	}

	ty := v.Type()
	switch ty {
	case cty.Bool:
		return FromBool(v.True())
	case cty.Number:
		bf := v.AsBigFloat()
		if bf.IsInt() {
			i64, _ := bf.Int64()
			return FromInt64(i64)
		}
		f64, _ := bf.Float64()
		return FromFloat64(f64)
	case cty.String:
		return FromString(v.AsString())
	case cty.DynamicPseudoType:
		panic("Dynamic types are not supported")
	}

	switch {
	case ty.IsListType(), ty.IsTupleType():
		list := make([]dynamodb.AttributeValue, 0, v.LengthInt())
		v.ForEachElement(func(_ cty.Value, elem cty.Value) bool {
			ev := FromCtyValue(elem)
			list = append(list, ev)
			return false
		})
		return dynamodb.AttributeValue{L: list}
	case ty.IsSetType():
		if ty.ElementType() == cty.String {
			// DynamoDB has native support for unsorted sets of strings.
			list := make([]string, 0, v.LengthInt())
			v.ForEachElement(func(_ cty.Value, elem cty.Value) bool {
				list = append(list, elem.AsString())
				return false
			})
			return dynamodb.AttributeValue{SS: list}
		}
		if ty.ElementType() == cty.Number {
			// DynamoDB has native support for unsorted sets of numbers.
			list := make([]string, 0, v.LengthInt())
			v.ForEachElement(func(_ cty.Value, elem cty.Value) bool {
				bf := elem.AsBigFloat()
				list = append(list, bf.String())
				return false
			})
			return dynamodb.AttributeValue{NS: list}
		}
		// Sets of other types cannot be represented in DynamoDB. Return list instead.
		list := make([]dynamodb.AttributeValue, 0, v.LengthInt())
		v.ForEachElement(func(_ cty.Value, elem cty.Value) bool {
			ev := FromCtyValue(elem)
			list = append(list, ev)
			return false
		})
		return dynamodb.AttributeValue{L: list}
	case ty.IsMapType(), ty.IsObjectType():
		m := make(map[string]dynamodb.AttributeValue, v.LengthInt())
		v.ForEachElement(func(key cty.Value, val cty.Value) bool {
			m[key.AsString()] = FromCtyValue(val)
			return false
		})
		return dynamodb.AttributeValue{M: m}
	case ty.IsCapsuleType():
		panic("Capsule types not supported")
	}

	// All cases should be covered above.
	panic(fmt.Sprintf("%s not supported", ty.FriendlyName()))
}

// ToCtyValue decodes an attribute to a value in the cty type system.
//
// The given type is used to determine how values are decoded. An error is
// returned if the type does not match.
func ToCtyValue(attr dynamodb.AttributeValue, ty cty.Type) (cty.Value, error) { // nolint: gocyclo
	if attr.NULL != nil && *attr.NULL {
		return cty.NullVal(ty), nil

	}
	switch ty {
	case cty.Bool:
		v := attr.BOOL
		if v == nil {
			return cty.NilVal, NotSetError{ty, "B"}
		}
		return cty.BoolVal(*v), nil
	case cty.String:
		v := attr.S
		if v == nil {
			return cty.NilVal, NotSetError{ty, "S"}
		}
		return cty.StringVal(*v), nil
	case cty.Number:
		v := attr.N
		if v == nil {
			return cty.NilVal, NotSetError{ty, "N"}
		}
		return cty.ParseNumberVal(*v)
	}

	switch {
	case ty.IsListType():
		et := ty.ElementType()
		switch {
		case len(attr.L) > 0:
			vals := make([]cty.Value, len(attr.L))
			for i, v := range attr.L {
				ev, err := ToCtyValue(v, et)
				if err != nil {
					return cty.NilVal, fmt.Errorf("element %d: %v", i, err)
				}
				vals[i] = ev
			}
			return cty.ListVal(vals), nil
		case len(attr.NS) > 0:
			if et != cty.Number {
				return cty.NilVal, fmt.Errorf("set of numbers cannot be assigned to %s", ty)
			}
			vals := make([]cty.Value, len(attr.NS))
			for i, v := range attr.NS {
				ev, err := cty.ParseNumberVal(v)
				if err != nil {
					return cty.NilVal, fmt.Errorf("element %d: %v", i, err)
				}
				vals[i] = ev
			}
			return cty.ListVal(vals), nil
		case len(attr.SS) > 0:
			if et != cty.String {
				return cty.NilVal, fmt.Errorf("set of strings cannot be assigned to %s", ty)
			}
			vals := make([]cty.Value, len(attr.SS))
			for i, v := range attr.SS {
				vals[i] = cty.StringVal(v)
			}
			return cty.ListVal(vals), nil
		default:
			return cty.ListValEmpty(et), nil
		}
	case ty.IsSetType():
		et := ty.ElementType()
		switch {
		case len(attr.NS) > 0:
			if et != cty.Number {
				return cty.NilVal, fmt.Errorf("set of numbers cannot be assigned to %s", ty)
			}
			vals := make([]cty.Value, len(attr.NS))
			for i, v := range attr.NS {
				ev, err := cty.ParseNumberVal(v)
				if err != nil {
					return cty.NilVal, fmt.Errorf("element %d: %v", i, err)
				}
				vals[i] = ev
			}
			return cty.SetVal(vals), nil
		case len(attr.SS) > 0:
			if et != cty.String {
				return cty.NilVal, fmt.Errorf("set of strings cannot be assigned to %s", ty)
			}
			vals := make([]cty.Value, len(attr.SS))
			for i, v := range attr.SS {
				vals[i] = cty.StringVal(v)
			}
			return cty.SetVal(vals), nil
		default:
			return cty.SetValEmpty(et), nil
		}
	case ty.IsTupleType():
		vals := make([]cty.Value, len(attr.L))
		types := ty.TupleElementTypes()
		for i, v := range attr.L {
			ev, err := ToCtyValue(v, types[i])
			if err != nil {
				return cty.NilVal, fmt.Errorf("element %d: %v", i, err)
			}
			vals[i] = ev
		}
		return cty.TupleVal(vals), nil
	case ty.IsMapType():
		et := ty.ElementType()
		if len(attr.M) == 0 {
			return cty.MapValEmpty(et), nil
		}
		vals := make(map[string]cty.Value, len(attr.M))
		for k, v := range attr.M {
			ev, err := ToCtyValue(v, et)
			if err != nil {
				return cty.NilVal, fmt.Errorf("element %s: %v", k, err)
			}
			vals[k] = ev
		}
		return cty.MapVal(vals), nil
	case ty.IsObjectType():
		if len(attr.M) == 0 {
			return cty.EmptyObjectVal, nil
		}
		vals := make(map[string]cty.Value, len(attr.M))
		types := ty.AttributeTypes()
		for k, v := range attr.M {
			et := types[k]
			ev, err := ToCtyValue(v, et)
			if err != nil {
				return cty.NilVal, fmt.Errorf("element %s: %v", k, err)
			}
			vals[k] = ev
		}
		return cty.ObjectVal(vals), nil
	}

	panic(fmt.Sprintf("Not supported: %s", ty.FriendlyName()))
}

// FromCtyPath encodes a cty path to an attribute.
func FromCtyPath(path cty.Path) dynamodb.AttributeValue {
	parts := make([]dynamodb.AttributeValue, len(path))
	for i, p := range path {
		switch v := p.(type) {
		case cty.GetAttrStep:
			parts[i] = dynamodb.AttributeValue{
				M: map[string]dynamodb.AttributeValue{
					"Attr": {S: aws.String(v.Name)},
				},
			}
		case cty.IndexStep:
			parts[i] = dynamodb.AttributeValue{
				M: map[string]dynamodb.AttributeValue{
					"Index": FromCtyValue(v.Key),
				},
			}
		default:
			// Should not happen, a path can only consist of Attr and Index steps.
			panic(fmt.Sprintf("Unknown path type %T", v))
		}
	}
	return dynamodb.AttributeValue{L: parts}
}

// ToCtyPath decodes an attribute to a cty path.
func ToCtyPath(attr dynamodb.AttributeValue) (cty.Path, error) {
	if attr.L == nil {
		return nil, fmt.Errorf("attribute list not set")
	}
	if len(attr.L) == 0 {
		return nil, nil
	}
	path := make(cty.Path, len(attr.L))
	for i, p := range attr.L {
		if p.M == nil {
			return nil, fmt.Errorf("list does not contain maps")
		}
		if attr, ok := p.M["Attr"]; ok {
			if attr.S == nil {
				return nil, fmt.Errorf("%d: attribute name not set", i)
			}
			path[i] = cty.GetAttrStep{Name: *attr.S}
			continue
		}
		if index, ok := p.M["Index"]; ok {
			if index.N != nil {
				// Numeric index
				k, err := cty.ParseNumberVal(*index.N)
				if err != nil {
					return nil, fmt.Errorf("%d: %v", i, err)
				}
				path[i] = cty.IndexStep{Key: k}
				continue
			}
			if index.S != nil {
				// String index
				path[i] = cty.IndexStep{Key: cty.StringVal(*index.S)}
				continue
			}
			return nil, fmt.Errorf("%d: index number or name must be set", i)
		}
		return nil, fmt.Errorf("%d: Attr or Index must be set", i)
	}
	return path, nil
}

// FromExpression encodes a graph expression into an attribute.
func FromExpression(expression resource.Expression) dynamodb.AttributeValue {
	expr := make([]dynamodb.AttributeValue, len(expression))
	for i, p := range expression {
		switch v := p.(type) {
		case resource.ExprLiteral:
			expr[i] = dynamodb.AttributeValue{M: map[string]dynamodb.AttributeValue{
				"Literal": FromCtyValue(v.Value),
			}}
		case resource.ExprReference:
			expr[i] = dynamodb.AttributeValue{M: map[string]dynamodb.AttributeValue{
				"Reference": FromCtyPath(v.Path),
			}}
		default:
			// This should not happen, an expression can only consist of
			// literals and expressions.
			panic(fmt.Sprintf("Unsupported type %T at %d", v, i))
		}
	}
	return dynamodb.AttributeValue{L: expr}
}

// ToExpression decodes an attribute to a graph expression.
func ToExpression(attr dynamodb.AttributeValue) (resource.Expression, error) {
	if len(attr.L) == 0 {
		return nil, nil
	}
	expr := make(resource.Expression, len(attr.L))
	for i, p := range attr.L {
		if p.M == nil {
			return nil, fmt.Errorf("list does not contain maps")
		}
		if lit, ok := p.M["Literal"]; ok {
			if lit.S == nil {
				return nil, fmt.Errorf("%d: literal string not set", i)
			}
			expr[i] = resource.ExprLiteral{Value: cty.StringVal(*lit.S)}
			continue
		}
		if ref, ok := p.M["Reference"]; ok {
			p, err := ToCtyPath(ref)
			if err != nil {
				return nil, fmt.Errorf("%d: parse reference: %v", i, err)
			}
			if len(p) == 0 {
				return nil, fmt.Errorf("%d: reference path is empty", i)
			}
			expr[i] = resource.ExprReference{Path: p}
			continue
		}
		return nil, fmt.Errorf("%d: Literal or Reference must be set", i)
	}
	return expr, nil
}

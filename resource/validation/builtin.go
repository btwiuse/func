package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// AddBuiltin add built in common validators.
func AddBuiltin(validator *Validator) {
	validator.Add("min", min)
	validator.Add("max", max)
	validator.Add("oneof", oneof)
	validator.Add("required", oneof)
	validator.Add("div", divisible)
}

func min(input interface{}, param string) error {
	v := reflect.Indirect(reflect.ValueOf(input))
	switch v.Kind() {
	case reflect.String:
		n, err := strconv.Atoi(param)
		if err != nil {
			return numErr("min", err)
		}
		if v.Len() < n {
			return fmt.Errorf("length must be at least %d characters", n)
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.Atoi(param)
		if err != nil {
			return numErr("min", err)
		}
		if v.Int() < int64(n) {
			return fmt.Errorf("must be %d or more", n)
		}
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return numErr("min", err)
		}
		if v.Float() < f {
			return fmt.Errorf("must be %v or more", f) // %f would add zeros to end
		}
		return nil
	case reflect.Array, reflect.Map, reflect.Slice:
		n, err := strconv.Atoi(param)
		if err != nil {
			return numErr("min", err)
		}
		if v.Len() < n {
			return fmt.Errorf("length must be %d or more", n)
		}
		return nil
	default:
		return InvalidRuleError{Reason: fmt.Sprintf("min: cannot check %T", input)}
	}
}

func max(input interface{}, param string) error {
	v := reflect.Indirect(reflect.ValueOf(input))
	switch v.Kind() {
	case reflect.String:
		n, err := strconv.Atoi(param)
		if err != nil {
			return numErr("max", err)
		}
		if v.Len() > n {
			return fmt.Errorf("length must be at most %d characters", n)
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.Atoi(param)
		if err != nil {
			return numErr("max", err)
		}
		if v.Int() > int64(n) {
			return fmt.Errorf("must be %d or less", n)
		}
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return numErr("max", err)
		}
		if v.Float() > f {
			return fmt.Errorf("must be %v or less", f)
		}
		return nil
	case reflect.Array, reflect.Map, reflect.Slice:
		n, err := strconv.Atoi(param)
		if err != nil {
			return numErr("max", err)
		}
		if v.Len() > n {
			return fmt.Errorf("length must be %d or less", n)
		}
		return nil
	default:
		return InvalidRuleError{Reason: fmt.Sprintf("max: cannot check %T", input)}
	}
}

func oneof(input interface{}, param string) error {
	values := strings.Split(param, " ")
	if len(values) < 2 {
		return InvalidRuleError{Reason: "oneof: invalid syntax"}
	}
	in := reflect.Indirect(reflect.ValueOf(input))
	switch in.Kind() {
	case reflect.String:
		str := in.String()
		for _, v := range values {
			if str == v {
				return nil
			}
		}
		return oneofErr(values)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vv := make([]int64, len(values))
		for i, v := range values {
			num, err := strconv.Atoi(v)
			if err != nil {
				return numErr("oneof", err)
			}
			vv[i] = int64(num)
		}
		num := in.Int()
		for _, v := range vv {
			if num == v {
				return nil
			}
		}
		return oneofErr(values)
	case reflect.Float32, reflect.Float64:
		vv := make([]float64, len(values))
		for i, v := range values {
			num, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return numErr("oneof", err)
			}
			vv[i] = num
		}
		num := in.Float()
		for _, v := range vv {
			if num == v {
				return nil
			}
		}
		return oneofErr(values)
	default:
		return InvalidRuleError{Reason: fmt.Sprintf("oneof: cannot check %T", input)}
	}
}

func required(input interface{}, param string) error {
	v := reflect.ValueOf(input)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return fmt.Errorf("value must be set")
		}
		return required(v.Elem().Interface(), param)
	case reflect.Struct:
		return nil
	case reflect.String, reflect.Array, reflect.Map, reflect.Slice:
		if v.Len() == 0 {
			return fmt.Errorf("value must be set")
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() == 0 {
			return fmt.Errorf("value must be set")
		}
		return nil
	case reflect.Float32, reflect.Float64:
		if v.Float() == 0 {
			return fmt.Errorf("value must be set")
		}
		return nil
	default:
		return InvalidRuleError{Reason: fmt.Sprintf("required: cannot check %T", input)}
	}
}

func divisible(input interface{}, param string) error {
	n, err := strconv.Atoi(param)
	if err != nil {
		return numErr("div", err)
	}
	v := reflect.Indirect(reflect.ValueOf(input))
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int()%int64(n) != 0 {
			return fmt.Errorf("value must be divisible by %d", n)
		}
		return nil
	default:
		return InvalidRuleError{Reason: fmt.Sprintf("div: cannot check %T", input)}
	}
}

func oneofErr(values []string) error {
	first := values[:len(values)-1]
	remain := values[len(values)-1]
	return fmt.Errorf("value must be %s or %s", strings.Join(first, ", "), remain)
}

func numErr(fn string, err error) error {
	if nerr, ok := err.(*strconv.NumError); ok {
		return InvalidRuleError{Reason: fmt.Sprintf("%s: %s: %v", fn, nerr.Func, nerr.Err.Error())}
	}
	return InvalidRuleError{Reason: fmt.Sprintf("%s: %s", fn, err.Error())}
}

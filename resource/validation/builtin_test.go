package validation

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestBuiltin(t *testing.T) {
	type test struct {
		param string
		input interface{}
		want  error
	}

	tests := []struct {
		validator func(input interface{}, param string) error
		inputs    []test
	}{
		{
			min,
			[]test{
				{"3", "a", fmt.Errorf("length must be at least 3 characters")},
				{"3", "abc", nil},
				{"3", "abcdef", nil},
				{"3", []string{"a"}, fmt.Errorf("length must be 3 or more")},
				{"3", []string{"a", "b", "c"}, nil},
				{"3", []string{"a", "b", "c", "d", "e"}, nil},
				{"3", map[string]string{"a": "A"}, fmt.Errorf("length must be 3 or more")},
				{"3", map[string]string{"a": "A", "b": "B", "c": "C"}, nil},
				{"3", map[string]string{"a": "A", "b": "B", "c": "C", "d": "D", "e": "E"}, nil},
				{"3", 0, fmt.Errorf("must be 3 or more")},
				{"3", 3, nil},
				{"3", 5, nil},
				{"3", float64(0.12), fmt.Errorf("must be 3 or more")},
				{"3", float64(3.45), nil},
				{"3", float64(5.67), nil},
				{"3.14", float32(0.12), fmt.Errorf("must be 3.14 or more")},
				{"3.14", float32(3.45), nil},
				{"3.14", float32(5.67), nil},
				{"", "a", InvalidRuleError{Reason: "min: Atoi: invalid syntax"}},
				{"", []string{"a"}, InvalidRuleError{Reason: "min: Atoi: invalid syntax"}},
				{"", 3, InvalidRuleError{Reason: "min: Atoi: invalid syntax"}},
				{"", 3.14, InvalidRuleError{Reason: "min: ParseFloat: invalid syntax"}},
				{"3", struct{}{}, InvalidRuleError{Reason: "min: cannot check struct {}"}},
			},
		},
		{
			max,
			[]test{
				{"3", "a", nil},
				{"3", "abc", nil},
				{"3", "abcdef", fmt.Errorf("length must be at most 3 characters")},
				{"3", []string{"a"}, nil},
				{"3", []string{"a", "b", "c"}, nil},
				{"3", []string{"a", "b", "c", "d", "e"}, fmt.Errorf("length must be 3 or less")},
				{"3", map[string]string{"a": "A"}, nil},
				{"3", map[string]string{"a": "A", "b": "B", "c": "C"}, nil},
				{"3", map[string]string{"a": "A", "b": "B", "c": "C", "d": "D", "e": "E"}, fmt.Errorf("length must be 3 or less")},
				{"3", 0, nil},
				{"3", 3, nil},
				{"3", 5, fmt.Errorf("must be 3 or less")},
				{"3", float64(0.12), nil},
				{"3", float64(3.45), fmt.Errorf("must be 3 or less")},
				{"3", float64(5.67), fmt.Errorf("must be 3 or less")},
				{"3.14", float32(0.12), nil},
				{"3.14", float32(3.45), fmt.Errorf("must be 3.14 or less")},
				{"3.14", float32(5.67), fmt.Errorf("must be 3.14 or less")},
				{"", "a", InvalidRuleError{Reason: "max: Atoi: invalid syntax"}},
				{"", []string{"a"}, InvalidRuleError{Reason: "max: Atoi: invalid syntax"}},
				{"", 3, InvalidRuleError{Reason: "max: Atoi: invalid syntax"}},
				{"", 3.14, InvalidRuleError{Reason: "max: ParseFloat: invalid syntax"}},
				{"3", struct{}{}, InvalidRuleError{Reason: "max: cannot check struct {}"}},
			},
		},
		{
			oneof,
			[]test{
				{"a b", "a", nil},
				{"a b", "x", fmt.Errorf("value must be a or b")},
				{"a b c", "x", fmt.Errorf("value must be a, b or c")},
				{"1 2", 1, nil},
				{"1 2", 9, fmt.Errorf("value must be 1 or 2")},
				{"1 2 3", 9, fmt.Errorf("value must be 1, 2 or 3")},
				{"1.23 2.34", 1.23, nil},
				{"1.23 2.34", 9.99, fmt.Errorf("value must be 1.23 or 2.34")},
				{"1.23 2.34 3.45", 9.99, fmt.Errorf("value must be 1.23, 2.34 or 3.45")},
				{"", 1, InvalidRuleError{Reason: "oneof: invalid syntax"}},
				{"a b", []string{}, InvalidRuleError{Reason: "oneof: cannot check []string"}},
				{"a b", struct{}{}, InvalidRuleError{Reason: "oneof: cannot check struct {}"}},
			},
		},
		{
			required,
			[]test{
				{"", "a", nil},
				{"", "", fmt.Errorf("value must be set")},
				{"", 1, nil},
				{"", 0, fmt.Errorf("value must be set")},
				{"", 1.00, nil},
				{"", 0.00, fmt.Errorf("value must be set")},
				{"", struct{}{}, nil},
				{"", (*struct{})(nil), fmt.Errorf("value must be set")},
				{"", &struct{}{}, nil},
			},
		},
		{
			divisible,
			[]test{
				{"64", 64, nil},
				{"99", 99, nil},
				{"32", 1024, nil},
				{"64", 65, fmt.Errorf("value must be divisible by 64")},
				{"16", 1.23, InvalidRuleError{Reason: "div: cannot check float64"}},
				{"16", struct{}{}, InvalidRuleError{Reason: "div: cannot check struct {}"}},
			},
		},
	}

	for _, tt := range tests {
		for _, tc := range tt.inputs {
			in := strings.ReplaceAll(fmt.Sprintf("%#v", tc.input), " ", "")
			t.Run(fmt.Sprintf("%s(%s,%q)", fnName(tt.validator), in, tc.param), func(t *testing.T) {
				got := tt.validator(tc.input, tc.param)
				gotStr := fmt.Sprintf("%v", got)
				wantStr := fmt.Sprintf("%v", tc.want)
				if gotStr != wantStr {
					t.Errorf("Got %q, Want: %q", gotStr, wantStr)
				}
			})
		}
	}
}

// fnName returns the name of the function
func fnName(fn interface{}) string {
	n := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	i := strings.LastIndex(n, ".")
	return n[i+1:]
}

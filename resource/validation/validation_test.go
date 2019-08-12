package validation_test

import (
	"fmt"

	"github.com/func/func/resource/validation"
)

func Example_builtin() {
	v := validation.New()
	validation.AddBuiltin(v)

	tag := "min=5,max=6"

	fmt.Println(v.Validate("bar", tag))
	fmt.Println(v.Validate("foobar", tag))
	fmt.Println(v.Validate("foobarbaz", tag))
	// Output:
	// length must be at least 5 characters
	// <nil>
	// length must be at most 6 characters
}

func Example_customFunc() {
	v := validation.New()

	v.Add("eq", func(value interface{}, param string) error {
		str := fmt.Sprintf("%v", value)
		if str != param {
			return fmt.Errorf("value must be %q", param)
		}
		return nil
	})

	fmt.Println(v.Validate("bar", "eq=foo"))
	fmt.Println(v.Validate("foo", "eq=foo"))
	// Output:
	// value must be "foo"
	// <nil>
}

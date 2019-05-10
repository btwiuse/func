package schema

import (
	"fmt"
	"reflect"
	"strings"
)

// An InputField is a single field marked with a func:"input" struct tag.
//
// Required
//
// Required fields can be marked with func:"input,required".
type InputField struct {
	Index    int
	Required bool
	Type     reflect.Type

	validation string
}

// An OutputField is a single field marked with a func:"output" struct tag.
type OutputField struct {
	Index int
	Type  reflect.Type
}

// Inputs extracts fields from target that have a func:"input" struct tag set.
//
// Panics if target is not a struct.
func Inputs(target reflect.Type) map[string]InputField {
	if target.Kind() != reflect.Struct {
		panic(fmt.Sprintf("Target must be a struct, not %s", target.Kind()))
	}
	fields := make(map[string]InputField)
	for i := 0; i < target.NumField(); i++ {
		f := target.Field(i)
		tag, ok := f.Tag.Lookup("func")
		if !ok {
			continue
		}
		if !strings.HasPrefix(tag, "input") {
			continue
		}
		field := InputField{
			Type:       f.Type,
			Required:   false,
			Index:      i,
			validation: f.Tag.Get("validate"),
		}
		if comma := strings.Index(tag, ","); comma >= 0 {
			attr := tag[comma+1:]
			if attr == "required" {
				field.Required = true
			} else {
				panic(fmt.Sprintf("Unsupported attribute %q set on %s", attr, f.Name))
			}
		}
		if f.PkgPath != "" {
			panic(fmt.Sprintf("Unexporeted field %q set as input", f.Name))
		}
		name := fieldName(f)
		fields[name] = field
	}
	return fields
}

// Outputs extracts fields from target that have a func:"output" struct tag set.
//
// Panics if target is not a struct.
func Outputs(target reflect.Type) map[string]OutputField {
	if target.Kind() != reflect.Struct {
		panic(fmt.Sprintf("Target must be a struct, not %s", target.Kind()))
	}
	fields := make(map[string]OutputField)
	for i := 0; i < target.NumField(); i++ {
		f := target.Field(i)
		tag, ok := f.Tag.Lookup("func")
		if !ok {
			continue
		}
		if !strings.HasPrefix(tag, "output") {
			continue
		}
		field := OutputField{
			Type:  f.Type,
			Index: i,
		}
		if comma := strings.Index(tag, ","); comma >= 0 {
			attr := tag[comma+1:]
			panic(fmt.Sprintf("Unsupported attribute %q set on %s", attr, f.Name))
		}
		if f.PkgPath != "" {
			panic(fmt.Sprintf("Unexporeted field %q set as output", f.Name))
		}
		name := fieldName(f)
		fields[name] = field
	}
	return fields
}

// Validate validates that the given value is a valid, according to a validate
// struct tag.
//
//   type Example struct {
//       Name int    `func:"input" validate:"gte=5,lt10"`
//       ARN  string `func:"input" validate:"arn"`
//   }
//
// Fields are validated using https://gopkg.in/go-playground/validator.v9.
//
// Additional custom validation rules are added:
//
//   arn    validate that the value is a valid AWS ARN.
//   div    validate that field is divisible by the given value.
//
// Type validation is not performed, validation may panic if an invalid
// validation rule is set based on type (for example, arn on a int64 field).
func (f InputField) Validate(value interface{}) error {
	return validate(value, f.validation)
}

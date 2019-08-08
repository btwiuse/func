package schema

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/zclconf/go-cty/cty"
)

// A Field represents an extracted field from a struct.
type Field struct {
	Index int               // The field's index, relative to the parent struct.
	Type  reflect.Type      // The field's type.
	Tags  map[string]string // Struct tags set on the field, excluding func and name tags.

	functag string // value for func:""
}

// A FieldSet contains extracted schema fields.
type FieldSet map[string]Field

// Inputs filters the FieldSet and returns all fields that are marked as an
// input, based on the func:"input" struct tag.
func (ff FieldSet) Inputs() FieldSet {
	out := make(FieldSet, len(ff))
	for k, v := range ff {
		if v.functag == "input" {
			out[k] = v
		}
	}
	return out
}

// Outputs filters the FieldSet and returns all fields that are marked as an
// output, based on the func:"output" struct tag.
func (ff FieldSet) Outputs() FieldSet {
	out := make(FieldSet, len(ff))
	for k, v := range ff {
		if v.functag == "output" {
			out[k] = v
		}
	}
	return out
}

// CtyType converts the FieldSet to a cty object type.
//
// The type is processed deeply, nested structs or pointers to structs are
// included.
//
// Fields that have interface types are not included as they cannot be
// represented in the cty type system.
//
// Panics if a field cannot be converted. See ImpliedType() for details.
func (ff FieldSet) CtyType() cty.Type {
	obj := make(map[string]cty.Type, len(ff))
	for k, v := range ff {
		if v.Type.Kind() == reflect.Interface {
			continue
		}
		obj[k] = ImpliedType(v.Type)
	}
	return cty.Object(obj)
}

// Fields extracts fields from target. Unexported fields are ignored.
//
// All fields are extracted, regardless if they are marked as an input, output
// or neither. The returned FieldSet may be further filtered to get the desired
// fields. The func struct tag is excluded from the Tags in the returned
// fields.
//
// The name of the field is derived from the struct field name. For example,
// ExampleField becomes example_field. This can be overridden by setting a
// `name:"<override>"` tag.
//
// Panics if target is not a struct or a pointer to a struct.
func Fields(target reflect.Type) FieldSet {
	t := target
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("Target must be a struct or pointer to struct, not %s", target.Kind()))
	}
	fields := make(FieldSet, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		field := Field{
			Type:  f.Type,
			Index: i,
		}
		tag := parseTag(f.Tag)
		var name string
		if n, ok := tag["name"]; ok {
			name = n
			delete(tag, "name")
		} else {
			name = fieldName(f)
		}
		field.functag = tag["func"]
		delete(tag, "func")
		field.Tags = tag
		fields[name] = field
	}
	return fields
}

// parseTag parses a struct tag string into a map where the key is the key of
// the struct tag and the value is the entire quoted value.
//
//   `example:"foo,bar"`
//   ->
//   map[string]string{"example": "foo,bar"}
//
// The code is mostly copied from go strlib reflect/type.go.
func parseTag(tag reflect.StructTag) map[string]string {
	tags := make(map[string]string)

	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		value, err := strconv.Unquote(qvalue)
		if err != nil {
			panic(fmt.Sprintf("unquote structtag value: %v", err))
		}
		tags[name] = value
	}

	return tags
}

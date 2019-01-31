package resource

import (
	"fmt"
	"reflect"
	"strings"
)

// FieldDirection describes the direction (input and/or output) of a field in a
// resource.
type FieldDirection byte

// Field directions
const (
	Input  FieldDirection = 1 << 0         // Input
	Output                = 1 << 1         // Output
	IO                    = Input | Output // Input or Output
)

// Input returns true of the field direction includes input.
func (f FieldDirection) Input() bool { return f&Input == Input }

// Output returns true if the field direction includes output.
func (f FieldDirection) Output() bool { return f&Output == Output }

// String implements fmt.Stringer
func (f FieldDirection) String() string {
	var io []string
	if f.Input() {
		io = append(io, "input")
	}
	if f.Output() {
		io = append(io, "output")
	}
	if len(io) == 0 {
		return "unknown"
	}
	return strings.Join(io, " / ")
}

// A Field is an input or output field in a struct.
type Field struct {
	Dir   FieldDirection // Field Input/Output direction.
	Name  string         // Struct tag name, 'foo' in input:"foo"
	Attr  string         // Extra info after comma, 'extra' in input:"foo,extra"
	Index int            // Field index.
	Type  reflect.Type   // Field type.
}

const inputStructTag = "input"
const outputStructTag = "output"

// Fields returns the fields from t.
//
// Fields are declared by struct tags using `input` and `output`:
//
//   type Resource struct {
//       Input string `input:"in"`
//       Output string `input:"out"`
//   }
//
// Panics if:
//  Target is not a struct.
//  Input AND output tag set on same field.
//  Input or output tag set on unexported field.
func Fields(target reflect.Type, dir FieldDirection) []Field {
	if target.Kind() != reflect.Struct {
		panic(fmt.Sprintf("Target must be a struct, not %s", target.Kind()))
	}
	var fields []Field
	for i := 0; i < target.NumField(); i++ {
		f := target.Field(i)
		it, input := f.Tag.Lookup(inputStructTag)
		ot, output := f.Tag.Lookup(outputStructTag)
		if input && output {
			// Both
			panic(fmt.Sprintf("Field %q is marked as an input and output; only one is allowed", f.Name))
		}
		if !input && !output {
			// Neither
			continue
		}
		if f.PkgPath != "" {
			dir := inputStructTag
			if output {
				dir = outputStructTag
			}
			panic(fmt.Sprintf("Unexporeted field %q set as %s", f.Name, dir))
		}
		if input && !dir.Input() {
			continue
		}
		if output && !dir.Output() {
			continue
		}
		tag := it
		dir := Input
		if output {
			tag = ot
			dir = Output
		}
		res := Field{Dir: dir, Name: tag, Index: i, Type: f.Type}
		if comma := strings.Index(tag, ","); comma >= 0 {
			res.Name = tag[:comma]
			res.Attr = tag[comma+1:]
		}
		fields = append(fields, res)
	}
	return fields
}

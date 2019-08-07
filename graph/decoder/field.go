package decoder

import (
	"reflect"

	"github.com/func/func/resource"
	"github.com/func/func/resource/schema"
)

// a field is a single parsed input or output field within a resource
// definition.
type field struct {
	def   resource.Definition
	index int

	// Only set for inputs
	input *schema.Field
	expr  *expression
}

// value returns the Value for the definition's struct field.
func (f field) value() reflect.Value {
	return reflect.Indirect(reflect.ValueOf(f.def)).Field(f.index)
}

func (f field) output() bool {
	return f.expr == nil
}

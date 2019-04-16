package decoder

import (
	"reflect"

	"github.com/func/func/resource"
)

// a field is a single parsed input or output field within a resource
// definition.
type field struct {
	def  resource.Definition
	info resource.Field
	expr *expression // nil if field is for an input
}

// value returns the Value for the definition's struct field.
func (f field) value() reflect.Value {
	return reflect.Indirect(reflect.ValueOf(f.def)).Field(f.info.Index)
}

func (f field) output() bool {
	return f.info.Dir == resource.Output
}

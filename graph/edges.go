package graph

import (
	"reflect"

	"gonum.org/v1/gonum/graph"
)

type ref struct {
	graph.Line
	Reference
}

// A Reference describes a dependency relationship for a single field between
// two resources.
type Reference struct {
	Source Field
	Target Field
}

// A Field is a single field in a resource definition struct.
type Field struct {
	Resource *Resource
	Index    []int
}

// Value returns the underlying value from the resource definition the Field
// points to.
func (f Field) Value() reflect.Value {
	val := reflect.Indirect(reflect.ValueOf(f.Resource.Definition))
	return val.FieldByIndex(f.Index)
}

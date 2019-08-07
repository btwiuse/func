package reconciler

import (
	"fmt"
	"reflect"

	"github.com/func/func/graph"
	"github.com/func/func/resource/schema"
	"github.com/pkg/errors"
)

// containingField finds the field in the given resources that matches the
// desired field.
// Returns nil if no such field was found.
func containingField(resources []*graph.Resource, field graph.Field) *graph.Resource {
	for _, r := range resources {
		if r.Config.Name == field.Name {
			return r
		}
	}
	return nil
}

// evalDependency evaluates the dependency value, collecting all the parent
// values of the expression, evaluating the expression, and inserting the value
// into the target field.
//
// All parent fields must be resolved before evaluation.
func evalDependency(dep *graph.Dependency) error {
	// Collect parent data.
	data := make(map[graph.Field]interface{})
	for _, f := range dep.Expr.Fields() {
		parent := containingField(dep.Parents(), f)
		if parent == nil {
			return errors.Errorf("no such field resource: %s", f.Name)
		}
		v := reflect.Indirect(reflect.ValueOf(parent.Config.Def))
		outputs := schema.Fields(v.Type()).Outputs()
		field, ok := outputs[f.Field]
		if !ok {
			return fmt.Errorf("%s does not have an output field %q", parent.Config.Name, f.Field)
		}
		srcVal := v.Field(field.Index)
		data[f] = reflect.Indirect(srcVal).Interface()
	}

	// Get target field.
	v := reflect.Indirect(reflect.ValueOf(dep.Child().Config.Def))
	inputs := schema.Fields(v.Type()).Inputs()
	field, ok := inputs[dep.Target.Field]
	if !ok {
		return fmt.Errorf("%s does not have an input field %q", dep.Child().Config.Name, dep.Target.Field)
	}
	dstVal := v.Field(field.Index)

	// Convert to/from pointer to match target value.
	var val reflect.Value
	isPtr := dstVal.Kind() == reflect.Ptr
	if isPtr {
		val = reflect.New(dstVal.Type().Elem())
	} else {
		val = reflect.New(dstVal.Type())
	}

	// Eval.
	if err := dep.Expr.Eval(data, val.Interface()); err != nil {
		return errors.Wrap(err, "evaluate expression")
	}

	// Set value.
	if isPtr {
		dstVal.Set(val)
	} else {
		dstVal.Set(val.Elem())
	}

	return nil
}

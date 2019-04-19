package reconciler

import (
	"fmt"
	"reflect"

	"github.com/func/func/graph"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// fieldValue returns returns the value of a certain field.
func fieldValue(res *graph.Resource, field graph.Field, dir resource.FieldDirection) (reflect.Value, error) {
	v := reflect.Indirect(reflect.ValueOf(res.Config.Def))
	for _, f := range resource.Fields(v.Type(), dir) {
		if f.Name == field.Field {
			return v.Field(f.Index), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("%s does not have an %s field %q", field.Type, dir, field.Field)
}

// containingField finds the field in the given resources that matches the
// desired field.
// Returns nil if no such field was found.
func containingField(resources []*graph.Resource, field graph.Field) *graph.Resource {
	for _, r := range resources {
		if r.Config.Def.Type() == field.Type && r.Config.Name == field.Name {
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
			return errors.Errorf("no such field resource: %s.%s", f.Type, f.Name)
		}
		srcVal, err := fieldValue(parent, f, resource.Output)
		if err != nil {
			return errors.Wrapf(err, "resolve parent field")
		}
		data[f] = reflect.Indirect(srcVal).Interface()
	}

	// Get target field.
	dstVal, err := fieldValue(dep.Child(), dep.Target, resource.Input)
	if err != nil {
		return errors.Wrap(err, "resolve target field")
	}

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

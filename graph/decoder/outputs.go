package decoder

import "github.com/zclconf/go-cty/cty"

func makeOutputs() outputs {
	return make(map[string]map[string]map[string]cty.Value)
}

// type -> name -> attribute -> value
type outputs map[string]map[string]map[string]cty.Value

func (o outputs) add(typename, name string, values map[string]cty.Value) {
	if o[typename] == nil {
		o[typename] = make(map[string]map[string]cty.Value)
	}
	o[typename][name] = values
}

func (o outputs) variables() map[string]cty.Value {
	m := make(map[string]cty.Value)
	for t, names := range o {
		byName := make(map[string]cty.Value)
		for name, outputs := range names {
			byName[name] = cty.ObjectVal(outputs)
		}
		m[t] = cty.ObjectVal(byName)
	}
	return m
}

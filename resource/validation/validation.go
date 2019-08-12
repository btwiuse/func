package validation

import (
	"fmt"
	"strings"

	"go.uber.org/multierr"
)

// A Validator maintains a list of registered validation rules.
type Validator struct {
	rules map[string]func(value interface{}, param string) error
}

// New creates a new empty validator. Rules should be added to the validator
// with Add().
func New() *Validator {
	return &Validator{
		rules: make(map[string]func(value interface{}, param string) error),
	}
}

// An InvalidRuleError is returned when the rule cannot be processed. This
// indicates a programmer error, rather than user error.
type InvalidRuleError struct {
	Reason string
}

func (e InvalidRuleError) Error() string {
	return fmt.Sprintf("invalid rule: %s", e.Reason)
}

// Add registers a new validation rule.
//
// Not safe for concurrent access.
//
// Panics if a validator with the same name has already been registered.
func (v *Validator) Add(name string, rule func(value interface{}, param string) error) {
	if _, ok := v.rules[name]; ok {
		panic(fmt.Sprintf("A checker with name %q has already been registered", name))
	}
	v.rules[name] = rule
}

// Validate validates the given input value against rules.
//
// Rules must be provided in a comma separated list (without space):
//   rule1,rule2
//
// Additional parameters can be provided to rules:
//   min=3,max=10
//
// If rules is empty, no validation is performed.
func (v *Validator) Validate(value interface{}, rules string) error {
	if rules == "" {
		return nil
	}
	parts := strings.Split(rules, ",")
	var err error
	for i, p := range parts {
		val := strings.SplitN(p, "=", 2)
		name := val[0]
		if name == "" {
			return InvalidRuleError{Reason: fmt.Sprintf("name not set for rule %d", i)}
		}
		param := ""
		if len(val) == 2 {
			param = val[1]
		}
		fn := v.rules[name]
		if fn == nil {
			return InvalidRuleError{Reason: fmt.Sprintf("no such rule: %q", name)}
		}
		err = multierr.Append(err, fn(value, param))
	}
	return err
}

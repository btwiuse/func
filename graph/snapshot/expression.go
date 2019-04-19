package snapshot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/func/func/graph"
	"github.com/pkg/errors"
)

// Expr is an expression in a snapshot. It is optimized for convenient use
// above anything else, as it is primarily meant to be used in tests.
type Expr string

var ident = "[a-z0-9_]+"
var exprRe = regexp.MustCompile(`\${(` + ident + `.` + ident + `.` + ident + `)}`)

// Fields returns the dynamic fields of an expression.
func (e Expr) Fields() []graph.Field {
	fields := exprRe.FindAllStringSubmatch(string(e), -1)
	if len(fields) == 0 {
		return nil
	}
	out := make([]graph.Field, len(fields))
	for i, f := range fields {
		parts := strings.Split(f[1], ".")
		out[i] = graph.Field{
			Type:  parts[0],
			Name:  parts[1],
			Field: parts[2],
		}
	}
	return out
}

// Eval evaluates the expression.
func (e Expr) Eval(data map[graph.Field]interface{}, target interface{}) error {
	var out *string
	switch t := target.(type) {
	case *string:
		out = t
	case *Expr:
		out = (*string)(t)
	default:
		return errors.Errorf("target must be *string, not %T", target)
	}
	val := string(e)
	for f, v := range data {
		k := string(ExprFrom(f))
		val = strings.Replace(val, k, fmt.Sprintf("%v", v), -1)
	}
	*out = val
	return nil
}

// ExprFrom creates an expression for a field.
func ExprFrom(field graph.Field) Expr {
	return Expr(fmt.Sprintf("${%s.%s.%s}", field.Type, field.Name, field.Field))
}

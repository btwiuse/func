package graph

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/func/func/ctyext"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// An Expression describes a value for a field.
//
// The Expression may consist of any combination of literals and references.
// The exprPart interface is closed, only ExprLiteral and ExprReference are
// allowed.
type Expression []exprPart

// exprPart is a part in an Expression. The interface is closed, only Parts
// declared in this package are allowed.
type exprPart interface{ isExpr() }

// ExprLiteral is a literal value in an expression.
type ExprLiteral struct {
	Value cty.Value
}

func (e ExprLiteral) isExpr() {}

// ExprReference is a part in an expression that has a reference to another
// field.
type ExprReference struct {
	Path cty.Path
}

func (e ExprReference) isExpr() {}

// References returns all referenced paths that are found in the expression.
//
// If the returned slice is empty, the expression contains no dynamic
// references. Such an expression can be evaluated with expr.Value(nil).
func (expr Expression) References() []cty.Path {
	var parts []cty.Path
	for _, e := range expr {
		if ref, ok := e.(ExprReference); ok {
			parts = append(parts, ref.Path)
		}
	}
	return parts
}

// An EvalContext provides context for evaluating an expression.
// See Value() for more details.
type EvalContext struct {
	// Variable values for resolving reference values in the expression.
	Variables map[string]cty.Value
}

// Value evaluates the expression value with the given variables.
//
// The following rules apply:
//
//   - If the expression contains a single literal value, it is returned.
//   - If the expression contains a single reference values, the referenced
//     value is extracted from vars and returned.
//   - If the expression contains a combination of values, they are
//     concatenated to a string value. Every value in vars must be convertible
//     to string.
//   - I an unknown value is encountered, an unknown value is returned.
//     If it was the only part in the expression, the type will match this part.
//     Otherwise, the returned value will be an unknown string.
//
// If the expression contains a reference to a variable that was not set in the
// ctx, an error is returned.
//
// A nil ctx is equivalent to an EvalContext with no variables, meaning only
// expressions with static literals can be evaluated.
func (expr Expression) Value(ctx *EvalContext) (cty.Value, error) {
	if ctx == nil {
		ctx = &EvalContext{}
	}
	vals := make([]cty.Value, len(expr))
	for i, e := range expr {
		switch p := e.(type) {
		case ExprLiteral:
			vals[i] = p.Value
		case ExprReference:
			val := cty.ObjectVal(ctx.Variables)
			for _, p := range p.Path {
				v, err := p.Apply(val)
				if err != nil {
					return cty.NilVal, err
				}
				val = v
			}
			vals[i] = val
		default:
			// This should not happen unless we add a new ExprPart that is not
			// supported here (always a bug).
			panic(fmt.Sprintf("Not supported: %T", p))
		}
	}
	if len(vals) == 0 {
		return cty.NilVal, nil
	}
	if len(vals) == 1 {
		return vals[0], nil
	}
	var buf bytes.Buffer
	for i, v := range vals {
		if !v.IsWhollyKnown() {
			return cty.UnknownVal(cty.String), nil
		}
		if conv := convert.GetConversion(v.Type(), cty.String); conv != nil {
			tmp, err := conv(v)
			if err != nil {
				return cty.NilVal, errors.Wrapf(err, "convert part %d: %v", i, err)
			}
			v = tmp
		}
		buf.WriteString(v.AsString())
	}
	return cty.StringVal(buf.String()), nil
}

// MergeLiterals merges consecutive literal values into a single literal. Parts
// of the expression that are not literals are returned in place as-is.
func (expr Expression) MergeLiterals() Expression {
	if len(expr) == 1 {
		return expr
	}

	join := func(expr Expression) Expression {
		if len(expr) == 0 {
			return nil
		}
		val, err := expr.Value(nil)
		if err != nil {
			// This should not happen as the expression is only constructed of
			// literal expressions that can be resolved without additional
			// variables.
			panic(err)
		}
		return Expression{ExprLiteral{Value: val}}
	}

	var out Expression // nolint: prealloc
	var pending Expression
	for _, e := range expr {
		if lit, ok := e.(ExprLiteral); ok {
			pending = append(pending, lit)
			continue
		}
		out = append(out, join(pending)...)
		pending = pending[:0]
		out = append(out, e)
	}
	out = append(out, join(pending)...)
	return out
}

// Equals returns true if the expression is equivalent to the other expression.
func (expr Expression) Equals(other Expression) bool {
	// NOTE(akupila): Ideally we wouldn't pull in cmp just for this.
	opts := []cmp.Option{
		cmp.Transformer("GoString", func(v cty.Value) string { return v.GoString() }),
		cmp.Transformer("Name", func(v cty.GetAttrStep) string { return v.Name }),
		cmp.Transformer("GoString", func(v cty.IndexStep) string { return v.GoString() }),
	}
	return cmp.Equal(expr, other, opts...)
}

type jsonExprPart map[jsonExprKey]json.RawMessage

type jsonExprKey string

const (
	jsonExprLit jsonExprKey = "lit"
	jsonExprRef jsonExprKey = "ref"
)

// MarshalJSON marshals an expression to json.
func (expr Expression) MarshalJSON() ([]byte, error) {
	parts := make([]jsonExprPart, len(expr))
	for i, e := range expr {
		switch v := e.(type) {
		case ExprLiteral:
			b, err := ctyjson.Marshal(v.Value, v.Value.Type())
			if err != nil {
				return nil, errors.Wrap(err, "marshal literal")
			}
			parts[i] = jsonExprPart{jsonExprLit: b}
		case ExprReference:
			str := ctyext.PathString(v.Path)
			parts[i] = jsonExprPart{jsonExprRef: []byte(fmt.Sprintf("%q", str))}
		default:
			return nil, errors.Errorf("unsupported type %T at %d", v, i)
		}
	}
	return json.Marshal(parts)
}

// UnmarshalJSON unmarshals an expression from json.
func (expr *Expression) UnmarshalJSON(b []byte) error {
	var parts []jsonExprPart
	if err := json.Unmarshal(b, &parts); err != nil {
		return errors.Wrap(err, "unmarshal expression parts")
	}
	ex := make(Expression, len(parts))
	for i, p := range parts {
		if lit, ok := p[jsonExprLit]; ok {
			var v ctyjson.SimpleJSONValue
			if err := v.UnmarshalJSON(lit); err != nil {
				return err
			}
			ex[i] = ExprLiteral{Value: v.Value}
			continue
		}
		if ref, ok := p[jsonExprRef]; ok {
			var str string
			if err := json.Unmarshal(ref, &str); err != nil {
				return err
			}
			path, err := ctyext.ParsePathString(str)
			if err != nil {
				return errors.Wrap(err, "parse path")
			}
			ex[i] = ExprReference{Path: path}
			continue
		}
		return errors.Errorf("unknown expression at %d: %v", i, p)
	}
	*expr = ex
	return nil
}

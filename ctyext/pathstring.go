package ctyext

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

// PathString returns a string of the given path.
func PathString(path cty.Path) string {
	var buf bytes.Buffer
	for i, p := range path {
		switch v := p.(type) {
		case cty.GetAttrStep:
			if i > 0 {
				buf.WriteByte('.')
			}
			buf.WriteString(v.Name)
		case cty.IndexStep:
			if v.Key.Type() == cty.Number {
				bf := v.Key.AsBigFloat()
				val, _ := bf.Int64()
				fmt.Fprintf(&buf, "[%d]", val)
				continue
			}
			fmt.Fprintf(&buf, "[%q]", v.Key.AsString())
		default:
			panic(fmt.Sprintf("Unknown path type %T", v))
		}
	}
	return buf.String()
}

// ParsePathString parses a string into a path.
func ParsePathString(str string) (cty.Path, error) {
	parts := strings.FieldsFunc(str, func(r rune) bool {
		return r == '.' || r == '[' || r == ']'
	})

	pp := make(cty.Path, len(parts))
	for i, p := range parts {
		if p[0] == '"' {
			// String index
			pp[i] = cty.IndexStep{
				Key: cty.StringVal(p[1 : len(p)-1]),
			}
			continue
		}
		if p[0] >= '0' && p[0] < '9' {
			// Numberic index
			n, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				return cty.Path{}, errors.Wrap(err, "parse index")
			}
			pp[i] = cty.IndexStep{
				Key: cty.NumberIntVal(n),
			}
			continue
		}
		pp[i] = cty.GetAttrStep{
			Name: p,
		}
	}

	return pp, nil
}

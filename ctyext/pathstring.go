package ctyext

import (
	"bytes"
	"fmt"

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

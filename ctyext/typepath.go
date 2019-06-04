package ctyext

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// ApplyTypePath applies the given path to a type and returns the type at that
// path. It is analogous to cty.Path.Apply() for values.
//
// The implementation may be incomplete; it assumes string based access is done
// with AttrSteps, while index based access is done with IndexSteps.
//
// If the path cannot be matched, an ApplyPathError is returned.
func ApplyTypePath(ty cty.Type, path cty.Path) (cty.Type, error) {
	for i, p := range path {
		switch e := p.(type) {
		case cty.GetAttrStep:
			switch {
			case ty.IsMapType():
				ty = ty.ElementType()
			case ty.IsObjectType():
				if !ty.HasAttribute(e.Name) {
					str := fmt.Sprintf("no attribute named %q", e.Name)
					if i > 0 {
						str += fmt.Sprintf(" in %s", PathString(path[:i]))
					}
					return cty.NilType, fmt.Errorf(str)
				}
				ty = ty.AttributeType(e.Name)
			default:
				str := fmt.Sprintf("cannot access nested type %q", e.Name)
				if i > 0 {
					str += fmt.Sprintf(", %s is a %s", PathString(path[:i]), ty.FriendlyNameForConstraint())
				} else {
					str += fmt.Sprintf(" in %s", ty.FriendlyNameForConstraint())
				}
				return cty.NilType, fmt.Errorf(str)
			}
		case cty.IndexStep:
			if ty.IsCollectionType() {
				ty = ty.ElementType()
				continue
			}
			str := fmt.Sprintf("cannot access indexed type from %s", ty.FriendlyName())
			if i > 0 {
				str += fmt.Sprintf(" in %s", PathString(path[:i]))
			}
			return cty.NilType, fmt.Errorf(str)
		}
	}
	return ty, nil
}

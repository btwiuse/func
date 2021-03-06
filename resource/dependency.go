package resource

import "github.com/zclconf/go-cty/cty"

// A Dependency is a dependency for a single field between two resources.
type Dependency struct {
	// Child is the name of the child resource with the dependency.
	Child string

	// Field is the path to the field within the child resource. The Field is
	// relative to the resource's Data.
	Field cty.Path

	// Expression is the expression value to resolve. The expression may refer
	// to multiple parent resources.
	Expression Expression
}

// Parents returns the names of the parent resources in the dependency's
// expression.
func (d Dependency) Parents() []string {
	refs := d.Expression.References()
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = ref[0].(cty.GetAttrStep).Name
	}
	return names
}

// Equals returns true if two expression are equal.
func (d Dependency) Equals(other Dependency) bool {
	return d.Field.Equals(other.Field) && d.Expression.Equals(other.Expression)
}

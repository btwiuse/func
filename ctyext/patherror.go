package ctyext

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// PathError is an error with an associated path.
type PathError struct {
	Path cty.Path
	Err  error
}

// Error formats the error message with a string path.
func (e PathError) Error() string {
	if len(e.Path) == 0 {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", PathString(e.Path), e.Err)
}

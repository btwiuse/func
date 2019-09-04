package api

import (
	"bytes"

	"github.com/hashicorp/hcl2/hcl"
)

// An ErrorCode indicates the type of error that occurred.
type ErrorCode string

// Valid error codes:
const (
	ValidationError ErrorCode = "validation"
	Unavailable     ErrorCode = "unavailable"
)

// An Error is a known error returned from the API.
type Error struct {
	Code        ErrorCode
	Message     string
	Diagnostics hcl.Diagnostics
}

func (e *Error) Error() string {
	if e.Diagnostics != nil {
		return e.Diagnostics.Error()
	}

	var buf bytes.Buffer
	buf.WriteString(string(e.Code))

	if e.Message != "" {
		buf.WriteString(": ")
		buf.WriteString(e.Message)
	}

	return buf.String()
}

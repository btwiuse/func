package api

import "github.com/hashicorp/hcl2/hcl"

// A RetryableError is returned when an api call can be retried.
type RetryableError interface {
	CanRetry() bool
}

type retryErr struct {
	err error
}

func (e retryErr) Error() string  { return e.err.Error() }
func (e retryErr) CanRetry() bool { return true }

// NewRetryableError creates a new retryable error.
func NewRetryableError(err error) error {
	return retryErr{err: err}
}

// A DiagnosticsError is returned when the error contains diagnostics.
// This is always a user error.
type DiagnosticsError interface {
	Diagnostics() hcl.Diagnostics
}

type diagsErr struct {
	diags hcl.Diagnostics
}

// Diagnostics returns embedded diagnostics.
func (e *diagsErr) Diagnostics() hcl.Diagnostics { return e.diags }
func (e *diagsErr) Error() string                { return e.diags.Error() }

// NewDiagnosticsError creates a new error that contains diagnostics.
func NewDiagnosticsError(diagnostics hcl.Diagnostics) error {
	return &diagsErr{diags: diagnostics}
}

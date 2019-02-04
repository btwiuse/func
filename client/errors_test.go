package client

import (
	"log"
	"os"
	"testing"

	"github.com/func/func/config"
)

func parseInvalidFile() error {
	l := &config.Loader{}
	_, diags := l.Load("testdata/invalid")
	return &DiagnosticsError{loader: l, Diagnostics: diags}
}

func ExampleDiagnosticsError_PrintDiagnostics() {
	err := parseInvalidFile()
	derr, ok := err.(*DiagnosticsError)
	if !ok {
		log.Fatal(err)
	}
	_ = derr.PrintDiagnostics(os.Stdout)
	// Output:
	// Error: Missing newline after block definition
	//
	//   on testdata/invalid/invalid.hcl line 6:
	//    4: resource "invalid" "syntax" {
	//    5:   # too many closing braces
	//    6: } }
	//
	// A block definition must end with a newline.
}

// Prevent Example from becoming full file example
func TestNoop(t *testing.T) {}

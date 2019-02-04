package client

import (
	"io"

	"github.com/hashicorp/hcl2/hcl"
	"golang.org/x/crypto/ssh/terminal"
)

// A DiagnosticsError is returned when the error originated from hcl diagnostics.
type DiagnosticsError struct {
	loader ConfigLoader
	hcl.Diagnostics
}

// PrintDiagnostics prints diagnostics to the given writer.
//
// If a TTY is attached, the output will be colorized and wrap at the terminal
// width. Otherwise, wrap will occur at 78 characters and output won't contain
// ANSI escape characters.
func (d *DiagnosticsError) PrintDiagnostics(w io.Writer) error {
	files := d.loader.Files()
	cols, _, err := terminal.GetSize(0)
	if err != nil {
		cols = 78
	}
	color := terminal.IsTerminal(0)
	wr := hcl.NewDiagnosticTextWriter(w, files, uint(cols), color)
	return wr.WriteDiagnostics(d.Diagnostics)
}

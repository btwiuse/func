package rpc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl2/hcl"
)

var (
	diagHCL = hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Test summary",
			Detail:   "Test detail",
			Subject: &hcl.Range{
				Filename: "test",
				Start:    hcl.Pos{Line: 1, Column: 2, Byte: 3},
				End:      hcl.Pos{Line: 4, Column: 5, Byte: 6},
			},
			Context: &hcl.Range{
				Filename: "test",
				Start:    hcl.Pos{Line: 7, Column: 8, Byte: 9},
				End:      hcl.Pos{Line: 10, Column: 11, Byte: 12},
			},
		},
		{
			Severity: hcl.DiagWarning,
			Summary:  "Warning",
		},
	}
	diagRPC = []*Diagnostic{
		{
			Error:   true,
			Summary: "Test summary",
			Detail:  "Test detail",
			Subject: &Range{
				Filename: "test",
				Start:    &Pos{Line: 1, Column: 2, Byte: 3},
				End:      &Pos{Line: 4, Column: 5, Byte: 6},
			},
			Context: &Range{
				Filename: "test",
				Start:    &Pos{Line: 7, Column: 8, Byte: 9},
				End:      &Pos{Line: 10, Column: 11, Byte: 12},
			},
		},
		{
			Error:   false,
			Summary: "Warning",
		},
	}
)

func TestDiagsFromHCL(t *testing.T) {
	got := DiagsFromHCL(diagHCL)
	want := diagRPC
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func TestDiagsToHCL(t *testing.T) {
	got := DiagsToHCL(diagRPC)
	want := diagHCL
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

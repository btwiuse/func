package httpapi

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
	diagHTTP = []*diagnostic{
		{
			Error:   true,
			Summary: "Test summary",
			Detail:  "Test detail",
			Subject: &diagrange{
				Filename: "test",
				Start:    pos{Line: 1, Column: 2, Byte: 3},
				End:      pos{Line: 4, Column: 5, Byte: 6},
			},
			Context: &diagrange{
				Filename: "test",
				Start:    pos{Line: 7, Column: 8, Byte: 9},
				End:      pos{Line: 10, Column: 11, Byte: 12},
			},
		},
		{
			Error:   false,
			Summary: "Warning",
		},
	}
)

func TestDiagsFromHCL(t *testing.T) {
	got := diagsFromHCL(diagHCL)
	want := diagHTTP
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

func TestDiagsToHCL(t *testing.T) {
	got := diagsToHCL(diagHTTP)
	want := diagHCL
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}

package rpc

import (
	"github.com/hashicorp/hcl2/hcl"
)

// DiagsFromHCL convets hcl diagnostics to rpc diagnostics.
//
// Expressions that may be part of the hcl diagnostics are ignored.
func DiagsFromHCL(diags hcl.Diagnostics) []*Diagnostic {
	dd := make([]*Diagnostic, len(diags))
	for i, d := range diags {
		dd[i] = &Diagnostic{
			Error:   d.Severity == hcl.DiagError,
			Summary: d.Summary,
			Detail:  d.Detail,
			Subject: RangeFromHCL(d.Subject),
			Context: RangeFromHCL(d.Context),
		}
	}
	return dd
}

// RangeFromHCL converts a hcl range to rpc range. Returns nil if rng is nil.
func RangeFromHCL(rng *hcl.Range) *Range {
	if rng == nil {
		return nil
	}
	return &Range{
		Filename: rng.Filename,
		Start:    PosFromHCL(rng.Start),
		End:      PosFromHCL(rng.End),
	}
}

// PosFromHCL converts a hcl position to rpc position.
func PosFromHCL(pos hcl.Pos) *Pos {
	return &Pos{
		Line:   int64(pos.Line),
		Column: int64(pos.Column),
		Byte:   int64(pos.Byte),
	}
}

// DiagsToHCL converts rpc diagnostics to hcl diagnostics.
func DiagsToHCL(diags []*Diagnostic) hcl.Diagnostics {
	dd := make([]*hcl.Diagnostic, len(diags))
	for i, d := range diags {
		sev := hcl.DiagWarning
		if d.Error {
			sev = hcl.DiagError
		}
		dd[i] = &hcl.Diagnostic{
			Severity: sev,
			Summary:  d.Summary,
			Detail:   d.Detail,
			Subject:  RangeToHCL(d.Subject),
			Context:  RangeToHCL(d.Context),
		}
	}
	return dd
}

// RangeToHCL converts an rpc range to hcl range.
func RangeToHCL(rng *Range) *hcl.Range {
	if rng == nil {
		return nil
	}
	return &hcl.Range{
		Filename: rng.Filename,
		Start:    PosToHCL(rng.Start),
		End:      PosToHCL(rng.End),
	}
}

// PosToHCL converts an rpc position to hcl position.
func PosToHCL(pos *Pos) hcl.Pos {
	return hcl.Pos{
		Line:   int(pos.Line),
		Column: int(pos.Column),
		Byte:   int(pos.Byte),
	}
}

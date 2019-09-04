package httpapi

import (
	"github.com/hashicorp/hcl2/hcl"
)

type diagnostic struct {
	Error   bool       `json:"err"`
	Summary string     `json:"sum"`
	Detail  string     `json:"det"`
	Subject *diagrange `json:"sub,omitempty"`
	Context *diagrange `json:"ctx,omitempty"`
}

type diagrange struct {
	Filename string `json:"file"`
	Start    pos    `json:"start"`
	End      pos    `json:"end"`
}

type pos struct {
	Line   int `json:"line"`
	Column int `json:"col"`
	Byte   int `json:"byte"`
}

func diagsFromHCL(diags hcl.Diagnostics) []*diagnostic {
	if len(diags) == 0 {
		return nil
	}
	out := make([]*diagnostic, len(diags))
	for i, d := range diags {
		out[i] = &diagnostic{
			Error:   d.Severity == hcl.DiagError,
			Summary: d.Summary,
			Detail:  d.Detail,
			Subject: rangeFromHCL(d.Subject),
			Context: rangeFromHCL(d.Context),
		}
	}
	return out
}

func rangeFromHCL(rng *hcl.Range) *diagrange {
	if rng == nil {
		return nil
	}
	return &diagrange{
		Filename: rng.Filename,
		Start:    posFromHCL(rng.Start),
		End:      posFromHCL(rng.End),
	}
}

func posFromHCL(p hcl.Pos) pos {
	return pos{
		Line:   p.Line,
		Column: p.Column,
		Byte:   p.Byte,
	}
}

func diagsToHCL(diags []*diagnostic) hcl.Diagnostics {
	if len(diags) == 0 {
		return nil
	}
	out := make(hcl.Diagnostics, len(diags))
	for i, d := range diags {
		sev := hcl.DiagWarning
		if d.Error {
			sev = hcl.DiagError
		}
		out[i] = &hcl.Diagnostic{
			Severity: sev,
			Summary:  d.Summary,
			Detail:   d.Detail,
			Subject:  rangeToHCL(d.Subject),
			Context:  rangeToHCL(d.Context),
		}
	}
	return out
}

func rangeToHCL(rng *diagrange) *hcl.Range {
	if rng == nil {
		return nil
	}
	return &hcl.Range{
		Filename: rng.Filename,
		Start:    posToHCL(rng.Start),
		End:      posToHCL(rng.End),
	}
}

func posToHCL(p pos) hcl.Pos {
	return hcl.Pos{
		Line:   p.Line,
		Column: p.Column,
		Byte:   p.Byte,
	}
}

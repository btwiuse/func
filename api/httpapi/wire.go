package httpapi

import "github.com/hashicorp/hcl2/hclpack"

type applyRequest struct {
	Project string        `json:"proj"`
	Config  *hclpack.Body `json:"cfg"`
}

type applyResponse struct {
	SourcesRequired []*sourceRequest `json:"srcs,omitempty"`
	Diagnostics     []*diagnostic    `json:"diags,omitempty"`
}

type sourceRequest struct {
	Key     string            `json:"key"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

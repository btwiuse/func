package api

import "github.com/hashicorp/hcl2/hcl"

// ApplyRequest is the request to send to Apply.
type ApplyRequest struct {
	// Namespace is the namespace to apply resource changes to.
	Namespace string

	// Config is the configuration to apply.
	Config hcl.Body
}

// ApplyResponse is returned from applying resources.
//
// The response may contain an UploadRequest, in which case the apply is
// rejected due to missing source code. The source code should be uploaded and
// then Apply should be retried.
type ApplyResponse struct {
	SourcesRequired []SourceRequest
}

// A SourceRequest is a request for source code.
//
// Source requests are returned from Apply when source code is required.
type SourceRequest struct {
	// Key is the key for the source code to upload.
	// The key maps back to a resource in the ApplyRequest config.
	Key string

	// Url is the destination URL to upload to.
	// The request should be done as a HTTP PUT.
	URL string

	// Headers include the additional headers that must be set on the upload.
	Headers map[string]string
}

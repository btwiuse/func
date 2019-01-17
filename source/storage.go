package source

import (
	"context"
	"io"
)

// UploadConfig is the configuration to pass to NewUpload when creating a new
// upload url.
type UploadConfig struct {
	Filename      string
	ContentMD5    string
	ContentLength int
}

// An UploadURL is returned from NewUpload.
type UploadURL struct {
	URL     string
	Headers map[string]string
}

// Storage provides source code storage.
type Storage interface {
	// Has returns true if the source storage has a file with a certain name.
	Has(ctx context.Context, filename string) (bool, error)

	// Get returns the given file from storage. The caller is responsible for
	// closing the file when done.
	Get(ctx context.Context, filename string) (io.ReadCloser, error)

	// NewUpload creates a new url that will accept uploads.
	NewUpload(cfg UploadConfig) (*UploadURL, error)
}

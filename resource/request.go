package resource

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// An AuthProvider provides authentication information for provisioning a
// resource.
type AuthProvider interface {
	AWS() (aws.CredentialsProvider, error)
}

// SourceCode contains one set of source code, matching a single source entry
// for the resource.
type SourceCode interface {
	// Digest returns a hash digest.
	Key() string

	// Reader returns a reader to the source tarball.
	Reader(ctx context.Context) (targz io.ReadCloser, err error)
}

// A CreateRequest is passed to a resource's Create method when a new resource
// is being created.
type CreateRequest struct {
	Auth   AuthProvider
	Source []SourceCode
}

// An UpdateRequest is passed to a resource's Update method when a new resource
// is being updated.
//
// Previous contains the previous version of the resource. The type for
// Previous will match the resource type.
type UpdateRequest struct {
	Auth     AuthProvider
	Source   []SourceCode
	Previous Definition

	SourceChanged bool
	ConfigChanged bool
}

// CreateRequest converts the update to a Create Request.
func (r *UpdateRequest) CreateRequest() *CreateRequest {
	return &CreateRequest{
		Auth:   r.Auth,
		Source: r.Source,
	}
}

// DeleteRequest converts the update to a Delete Request.
func (r *UpdateRequest) DeleteRequest() *DeleteRequest {
	return &DeleteRequest{
		Auth: r.Auth,
	}
}

// A DeleteRequest is passed to a resource when it is being deleted.
type DeleteRequest struct {
	Auth AuthProvider
}

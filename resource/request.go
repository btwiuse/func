package resource

import (
	"github.com/aws/aws-sdk-go-v2/aws"
)

// An AuthProvider provides authentication information for provisioning a
// resource.
type AuthProvider interface {
	AWS() (aws.CredentialsProvider, error)
}

// A CreateRequest is passed to a resource's Create method when a new resource
// is being created.
type CreateRequest struct {
	Auth AuthProvider
}

// An UpdateRequest is passed to a resource's Update method when a new resource
// is being updated.
//
// Previous contains the previous version of the resource. The type for
// Previous will match the resource type.
type UpdateRequest struct {
	Auth     AuthProvider
	Previous interface{}
}

// A DeleteRequest is passed to a resource when it is being deleted.
type DeleteRequest struct {
	Auth AuthProvider
}

package resource

import (
	"context"
)

// A Definition describes a resource.
//
// All resources must implement this interface.
type Definition interface {
	Create(ctx context.Context, req *CreateRequest) error
	Update(ctx context.Context, req *UpdateRequest) error
	Delete(ctx context.Context, req *DeleteRequest) error
}

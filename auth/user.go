package auth

import (
	"context"
	"fmt"
)

type ctxKey string

const userCtxKey ctxKey = "user"

// A User is an authorized user.
type User struct {
	ID string
}

// ContextWithUser sets a user on the context. If a user is already set, it is
// overwritten. No checks are done on the user, including it being nil.
//
// The user can be retrieved using UserFromContext.
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}

// UserFromContext retrieves a user from context. An error is returned if no
// user is set.
//
// The user can be set with ContextWithUser.
func UserFromContext(ctx context.Context) (*User, error) {
	u, ok := ctx.Value(userCtxKey).(*User)
	if !ok {
		return nil, fmt.Errorf("user not set")
	}
	return u, nil
}

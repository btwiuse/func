package storage

import "github.com/pkg/errors"

// ErrNotFound is returned when attempting to get or delete an item that does
// not exist.
var ErrNotFound = errors.New("not found")

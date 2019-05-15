package reconciler

import (
	"context"
	"io"
)

type source struct {
	key     string
	storage SourceStorage
}

func (s *source) Key() string { return s.key }
func (s *source) Reader(ctx context.Context) (targz io.ReadCloser, err error) {
	return s.storage.Get(ctx, s.key)
}

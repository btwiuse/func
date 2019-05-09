package reconciler

import (
	"context"
	"io"

	"github.com/func/func/config"
)

type source struct {
	info    config.SourceInfo
	storage SourceStorage
}

func (s *source) Key() string { return s.info.Key }
func (s *source) Size() int   { return s.info.Len }
func (s *source) Reader(ctx context.Context) (targz io.ReadCloser, err error) {
	return s.storage.Get(ctx, s.info.Key)
}

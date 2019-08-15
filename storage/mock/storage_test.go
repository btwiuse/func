package mock_test

import (
	"reflect"
	"testing"

	"github.com/func/func/storage/mock"
	"github.com/func/func/storage/testsuite"
)

func TestMock(t *testing.T) {
	testsuite.Run(t, testsuite.Config{
		New: func(t *testing.T, types map[string]reflect.Type) (testsuite.Target, func()) {
			return &mock.Storage{}, func() {}
		},
	})
}

package disk

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/func/func/source"
	"github.com/pkg/errors"
)

// Storage is a local source repository that stores files on disk.
type Storage struct {
	Dir string // Directory to store files in.

	server *http.Server
	addr   string

	once  sync.Once
	ready chan struct{} // closed when server is listening
}

var _ source.Storage = (*Storage)(nil)

func (s *Storage) init() {
	s.once.Do(func() {
		s.ready = make(chan struct{})
	})
}

// ListenAndServe starts a HTTP server on an OS-assigned port and starts
// accepting uploads.
func (s *Storage) ListenAndServe() error {
	s.init()
	s.server = &http.Server{Handler: http.HandlerFunc(s.handleRequest)}
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		return errors.Wrap(err, "create listener")
	}
	s.addr = lis.Addr().String()
	close(s.ready)
	return s.server.Serve(lis)
}

func (s *Storage) handleRequest(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		http.Error(w, "Filename not provided", http.StatusBadRequest)
		return
	}
	filename := filepath.Join(s.Dir, name)
	f, err := os.Create(filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not create file: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(f, r.Body); err != nil {
		http.Error(w, fmt.Sprintf("Could not write file: %v", err), http.StatusInternalServerError)
		return
	}
	if err := f.Close(); err != nil {
		http.Error(w, fmt.Sprintf("Could not close file: %v", err), http.StatusInternalServerError)
	}
}

// NewUploadURL creates a new upload url that the local source repo will
// accept.
func (s *Storage) NewUpload(cfg source.UploadConfig) (*source.UploadURL, error) {
	s.init()
	<-s.ready

	url := &url.URL{
		Scheme: "http",
		Host:   s.addr,
		Path:   cfg.Filename,
	}

	return &source.UploadURL{
		URL: url.String(),
		Headers: map[string]string{
			"Content-MD5": cfg.ContentMD5,
		},
	}, nil
}

// Has returns true if the given filename exists in the source storage.
func (s *Storage) Has(ctx context.Context, filename string) (bool, error) {
	files, err := ioutil.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "read files")
	}
	for _, f := range files {
		if f.Name() == filename {
			return true, nil
		}
	}
	return false, nil
}

// Get returns a source file. The caller is responsible for closing the file.
func (s *Storage) Get(ctx context.Context, filename string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.Dir, filename))
}

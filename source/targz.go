package source

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// TarGZ compresses source files to a .tar.gz archive.
type TarGZ struct{}

// Compress compresses the given files into a tarball that is written into w.
//
// The file paths will be relative to the given directory.
func (TarGZ) Compress(w io.Writer, dir string) error {
	dir = filepath.Clean(dir)
	gz := gzip.NewWriter(w)
	tf := tar.NewWriter(gz)

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if path == dir {
			// Skip self
			return nil
		}
		if err != nil {
			return errors.WithStack(err)
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return errors.WithStack(err)
		}
		hdr.Name = strings.TrimPrefix(path, dir+string(filepath.Separator))
		if err = tf.WriteHeader(hdr); err != nil {
			return errors.WithStack(err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return errors.WithStack(err)
		}
		if _, err := io.Copy(tf, f); err != nil {
			return errors.WithStack(err)
		}
		if err := f.Close(); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}); err != nil {
		return errors.WithStack(err)
	}

	if err := tf.Close(); err != nil {
		return errors.WithStack(err)
	}
	if err := gz.Close(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

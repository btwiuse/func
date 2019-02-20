package convert

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"

	"github.com/pkg/errors"
)

// Zip converts a given .tar.gz stream to a zip.
func Zip(w io.Writer, targz io.Reader) error {
	gz, err := gzip.NewReader(targz)
	if err != nil {
		return errors.Wrap(err, "create gzip reader")
	}
	tf := tar.NewReader(gz)
	z := zip.NewWriter(w)

	for {
		h, err := tf.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "read tar")
		}

		info := h.FileInfo()
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return errors.Wrap(err, "get zip header")
		}

		if info.IsDir() {
			header.Name += "/"
			header.Method = zip.Store
		}

		writer, err := z.CreateHeader(header)
		if err != nil {
			return errors.Wrap(err, "create zip header")
		}

		if info.IsDir() {
			continue
		}

		if header.Mode().IsRegular() {
			_, err := io.CopyN(writer, tf, info.Size())
			if err != nil {
				return errors.Wrap(err, "copy file")
			}
		}
	}

	if err := z.Close(); err != nil {
		return errors.Wrap(err, "close zip writer")
	}

	return nil
}

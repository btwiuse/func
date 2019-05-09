package source_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"testing"

	"github.com/func/func/source"
	"github.com/google/go-cmp/cmp"
)

func TestCompress(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		check   func(t *testing.T, buf *bytes.Buffer)
		wantErr bool
	}{
		{
			"TarGZ",
			"testdata/compress",
			func(t *testing.T, buf *bytes.Buffer) {
				want := map[string][]byte{
					"a.txt":     []byte("aaa\n"),
					"sub/b.txt": []byte("bbb\n"),
				}
				got := filesInGzip(t, buf)
				if diff := cmp.Diff(got, want); diff != "" {
					t.Errorf("Files do not match (-got, +want)\n%s", diff)
				}
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tgz := &source.TarGZ{}
			err := tgz.Compress(&buf, tt.dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Compress() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.check(t, &buf)
		})
	}
}

func filesInGzip(t *testing.T, r io.Reader) map[string][]byte {
	t.Helper()
	gzr, err := gzip.NewReader(r)
	if err != nil {
		t.Fatalf("Could not create gzip reader: %v", err)
	}
	files := make(map[string][]byte)
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		data, err := ioutil.ReadAll(tr)
		if err != nil {
			t.Fatalf("Could not read file in tar: %v", err)
		}
		files[hdr.Name] = data
	}
	return files
}

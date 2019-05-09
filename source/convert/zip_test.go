package convert_test

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/func/func/source"
	"github.com/func/func/source/convert"
	"github.com/google/go-cmp/cmp"
)

func TestZip(t *testing.T) {
	targz := source.TarGZ{}
	var tarBuf bytes.Buffer
	if err := targz.Compress(&tarBuf, "testdata"); err != nil {
		t.Fatalf("Compress test tar.gz: %v", err)
	}

	var gotZip bytes.Buffer
	err := convert.Zip(&gotZip, &tarBuf)
	if err != nil {
		t.Fatalf("Zip() error = %v", err)
	}

	got := filesInZip(t, &gotZip)
	want := map[string][]byte{
		"a.txt":     []byte("aaa\n"),
		"sub/b.txt": []byte("bbb\n"),
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Files do not match (-got, +want)\n%s", diff)
	}
}

func filesInZip(t *testing.T, buf *bytes.Buffer) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string][]byte)
	dir := ""
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			dir = zf.Name
			continue
		}
		f, err := zf.Open()
		if err != nil {
			t.Errorf("Open %s in zip: %v", zf.Name, err)
		}
		buf, err := ioutil.ReadAll(f)
		if err != nil {
			t.Errorf("Read %s: %v", zf.Name, err)
		}
		_ = f.Close()
		name := filepath.Join(dir, zf.Name)
		got[name] = buf
	}
	return got
}

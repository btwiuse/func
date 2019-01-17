package disk_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"testing"

	"github.com/func/func/source"
	"github.com/func/func/source/disk"
)

func TestStorage(t *testing.T) {
	dir, done := mktemp(t)
	defer done()

	s := &disk.Storage{Dir: dir}
	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			panic(err)
		}
	}()

	ctx := context.Background()

	name := "file.txt"
	data := []byte("foo")

	h := md5.Sum(data)
	md5 := base64.StdEncoding.EncodeToString(h[:])

	// File does not exist
	has, err := s.Has(ctx, name)
	if err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if has {
		t.Fatalf("Has() got = %t, want = %t", has, false)
	}

	// Create upload
	u, err := s.NewUpload(source.UploadConfig{
		Filename:      name,
		ContentMD5:    md5,
		ContentLength: len(data),
	})
	if err != nil {
		t.Fatalf("NewUpload() error = %v", err)
	}

	// Upload data
	req, err := http.NewRequest(http.MethodPut, u.URL, strings.NewReader("foo"))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	for k, v := range u.Headers {
		req.Header.Add(k, v)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		if dumped, err := httputil.DumpResponse(resp, true); err == nil {
			t.Log(string(dumped))
		}
		t.Fatalf("StatusCode = %d", resp.StatusCode)
	}

	// File should now exist
	has, err = s.Has(ctx, name)
	if err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if !has {
		t.Fatalf("Has() got = %t, want = %t", has, true)
	}

	// Read back file
	f, err := s.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	got, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	want := []byte("foo")
	if !bytes.Equal(got, want) {
		t.Errorf("Stored data does not match\nGot\n%s\nWant\n%s", hex.Dump(got), hex.Dump(want))
	}
}

func mktemp(t *testing.T) (string, func()) {
	dir, err := ioutil.TempDir("", "func-storage")
	if err != nil {
		t.Fatal(err)
	}
	return dir, func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	}
}

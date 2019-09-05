package auth_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/func/func/auth"
	"github.com/google/go-cmp/cmp"
)

func TestTokenFilename(t *testing.T) {
	c1 := "abc"
	c2 := "def"
	f1 := auth.TokenFilename(c1)
	f2 := auth.TokenFilename(c2)
	if f1 == f2 {
		t.Errorf("Tokens with different client ids must have different filenames\n%s -> %s\n%s -> %s", c1, f1, c2, f2)
	}
}

func TestTokenFilename_noClientID(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("Want panic with no client id")
		}
	}()
	auth.TokenFilename("")
}

func TestTokenFromFile_NotExist(t *testing.T) {
	got, err := auth.TokenFromFile("nonexisting")
	// Should not return an error
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("Returned token is not nil, got %#v", got)
	}
}

func TestToken_DiskIO(t *testing.T) {
	before := &auth.Token{
		ClientID:    "abc",
		AccessToken: "123",
	}
	tmp, done := tempdir(t)
	defer done()

	filename := filepath.Join(tmp, ".func", "credentials", "abc")
	if err := before.SaveToFile(filename); err != nil {
		t.Fatalf("SaveToFile() err = %v", err)
	}

	after, err := auth.TokenFromFile(filename)
	if err != nil {
		t.Fatalf("TokenFromFile() err = %v", err)
	}

	if diff := cmp.Diff(before, after); diff != "" {
		t.Errorf("Diff (-before +after)\n%s", diff)
	}
}

func TestToken_WriteTo(t *testing.T) {
	tok := &auth.Token{
		ClientID:     "abc",
		AccessToken:  "def",
		Type:         "ghi",
		RefreshToken: "jkl",
		Expiry:       time.Now(),
	}
	var buf bytes.Buffer
	n, err := tok.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(buf.Len()) {
		t.Errorf("Written bytes do not match; got %d, want %d", n, buf.Len())
	}
}

func TestToken_ReadFrom(t *testing.T) {
	b, _ := json.Marshal(auth.Token{
		ClientID:     "abc",
		AccessToken:  "def",
		Type:         "ghi",
		RefreshToken: "jkl",
		Expiry:       time.Now(),
	})

	tok := &auth.Token{}
	n, err := tok.ReadFrom(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(b)) {
		t.Errorf("Read bytes do not match; got %d, want %d", n, len(b))
	}
}

func tempdir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	return dir, func() {
		_ = os.RemoveAll(dir)
	}
}

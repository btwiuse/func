package auth

import (
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gopkg.in/square/go-jose.v2"
)

func TestKeyProviderStatic(t *testing.T) {
	key := []byte("secret")
	got, err := KeyProviderStatic(key).Key()
	if err != nil {
		t.Fatal(err)
	}
	gotb := got.([]byte)
	if !bytes.Equal(gotb, key) {
		t.Errorf("Key does not match\nGot:\n%s\nWant:\n%s", hex.Dump(gotb), hex.Dump(key))
	}
}

func TestKeyProviderFile(t *testing.T) {
	key := []byte("secret")
	file, done := tempFile(t, []byte("secret"))
	defer done()
	got, err := KeyProviderFile(file).Key()
	if err != nil {
		t.Fatal(err)
	}
	gotb := got.([]byte)
	if !bytes.Equal(gotb, key) {
		t.Errorf("Key does not match\nGot:\n%s\nWant:\n%s", hex.Dump(gotb), hex.Dump(key))
	}
}

func TestKeyProviderJWKS(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	jwks := &KeyProviderJWKS{Endpoint: ts.URL + "/jwks.json"}
	got, err := jwks.Key()
	if err != nil {
		t.Fatal(err)
	}
	set, ok := got.(*jose.JSONWebKeySet)
	if !ok {
		t.Fatalf("Returned type is %T, want *jose.JSONWebKeySet", got)
	}
	if len(set.Keys) != 1 {
		t.Fatalf("Got %d keys, want 1 key", len(set.Keys))
	}
}

func TestKeyProviderJWKS_cache(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.FileServer(http.Dir("testdata")).ServeHTTP(w, r)
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	jwks := KeyProviderJWKS{Endpoint: ts.URL + "/jwks.json"}

	_, err := jwks.Key()
	if err != nil {
		t.Fatal(err)
	}
	_, err = jwks.Key() // Cached keys
	if err != nil {
		t.Fatal(err)
	}

	if calls != 1 {
		t.Errorf("%d calls were made to jwks endpoind, want 1", calls)
	}
}

func tempFile(t *testing.T, data []byte) (name string, done func()) {
	t.Helper()
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	name = f.Name()
	return name, func() {
		_ = os.RemoveAll(name)
	}
}

package auth_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/func/func/auth"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2"
)

func TestTokenFilename_diff(t *testing.T) {
	aa := auth.TokenFilename("a", "a")
	ab := auth.TokenFilename("a", "b")
	ba := auth.TokenFilename("b", "a")
	if aa == ab {
		t.Errorf("Tokens with different client id must have different filenames\n%q == %q", aa, ab)
	}
	if aa == ab {
		t.Errorf("Tokens with different auduence must have different filenames\n%q == %q", aa, ba)
	}
}

func TestToken_io(t *testing.T) {
	tok := &auth.Token{
		OAuth2Token: &oauth2.Token{AccessToken: "abc"},
		IDClaims:    map[string]interface{}{"name": "tester"},
	}
	dir, done := tempdir(t)
	defer done()
	err := tok.Persist(dir, "a", "b")
	if err != nil {
		t.Fatal(err)
	}

	got, err := auth.LoadToken(dir, "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(oauth2.Token{}),
	}
	if diff := cmp.Diff(got, tok, opts...); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
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

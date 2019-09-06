package auth

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/func/func/auth/testserver"
	"gopkg.in/oauth2.v3/models"
)

func TestClient_Authorize_open(t *testing.T) {
	ts := &testserver.OIDC{}
	issuer, done := ts.Start()
	defer done()

	scope := []string{"offline_access"}

	ctx, cancel := context.WithCancel(context.Background())
	cli := &Client{
		Issuer: issuer,

		open: func(authorize string) {
			u, err := url.Parse(authorize)
			if err != nil {
				t.Fatalf("Parse authorize url: %v", err)
			}

			if !strings.HasPrefix(u.String(), issuer) {
				t.Errorf("Url does not start with issuer\nGot  http://%s\nWant %s", u.Host, issuer)
			}

			query := u.Query()
			checkValue(t, query, "response_type", "code")
			checkValue(t, query, "scope", "openid offline_access")

			if query.Get("state") == "" {
				t.Errorf("state not set")
			}

			// Cleanup
			cancel()
		},
	}
	_, err := cli.Authorize(ctx, "https://test.com/", scope...)
	if err == context.Canceled {
		// This is expected
		return
	}
	t.Fatal(err)
}

func TestClient_Authorize_callback(t *testing.T) {
	clientID := "testclient"
	ts := &testserver.OIDC{
		Client: &models.Client{
			ID:     clientID,
			UserID: "user123",
			Domain: Callback.String(),
		},
		UserClaims: map[string]interface{}{
			"nickname": "tester",
		},
	}
	issuer, done := ts.Start()
	defer done()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cli := &Client{
		Issuer:   issuer,
		ClientID: clientID,
		open: func(authorize string) {
			resp, err := http.Get(authorize)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK {
				b, _ := httputil.DumpResponse(resp, true)
				t.Error(string(b))
				cancel()
			}
			_ = resp.Body.Close()
		},
	}
	scope := []string{"profile", "offline_access"}
	got, err := cli.Authorize(ctx, "https://example.com/", scope...)
	if err != nil {
		t.Fatalf("Authorize() err = %v", err)
	}
	if got == nil {
		t.Fatalf("Returned token is nil")
	}
	nick, _ := got.IDClaims["nickname"].(string)
	if nick != "tester" {
		t.Errorf("Nickname in id token does not match; got %q, want %q", nick, "tester")
	}
}

func checkValue(t *testing.T, vals url.Values, name string, want string) {
	t.Helper()
	got := vals.Get(name)
	if got != want {
		t.Errorf("Value for %q does not match; got %q, want %q", name, got, want)
	}
}

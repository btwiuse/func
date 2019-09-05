package auth

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/func/func/auth/internal/testserver"
	"golang.org/x/oauth2"
	"gopkg.in/oauth2.v3/models"
)

func TestClient_Authorize_open(t *testing.T) {
	clientID := "testclient"

	lis, issuer := listener(t)
	done := serve(lis, testserver.OAuth2(
		&models.Client{
			ID:     clientID,
			UserID: "user123",
			Domain: DefaultCallback.String(),
		},
	))
	defer done()

	scope := []string{"openid", "profile", "offline_access"}

	ctx, cancel := context.WithCancel(context.Background())
	cli := &Client{
		Endpoint: oauth2.Endpoint{
			AuthURL:  issuer + "/authorize",
			TokenURL: issuer + "/oauth/token",
		},
		ClientID: clientID,
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
			checkValue(t, query, "scope", strings.Join(scope, " "))
			checkValueSet(t, query, "state")

			// Cleanup
			cancel()
		},
	}
	_, _ = cli.Authorize(ctx, scope) // error is expected because context is cancelled
}

func TestClient_Authorize_callback(t *testing.T) {
	clientID := "testclient"

	lis, issuer := listener(t)
	done := serve(lis, testserver.OAuth2(
		&models.Client{
			ID:     clientID,
			UserID: "user123",
			Domain: DefaultCallback.String(),
		},
	))
	defer done()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cli := &Client{
		Endpoint: oauth2.Endpoint{
			AuthURL:  issuer + "/authorize",
			TokenURL: issuer + "/oauth/token",
		},
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
	scope := []string{"openid", "profile", "offline_access"}
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("audience", "https://example.com/"),
	}
	got, err := cli.Authorize(ctx, scope, opts...)
	if err != nil {
		t.Fatalf("Authorize() err = %v", err)
	}
	if got == nil {
		t.Error("Returned token is nil")
	}
}

func checkValue(t *testing.T, vals url.Values, name string, want string) {
	t.Helper()
	got := vals.Get(name)
	if got != want {
		t.Errorf("Value for %q does not match; got %q, want %q", name, got, want)
	}
}

func checkValueSet(t *testing.T, vals url.Values, name string) {
	t.Helper()
	got := vals.Get(name)
	if got == "" {
		t.Errorf("Value for %q not set", name)
	}
}

func listener(t *testing.T) (lis net.Listener, url string) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	url = "http://" + lis.Addr().String()
	return lis, url
}

func serve(lis net.Listener, handler http.Handler) (done func()) {
	srv := &http.Server{Handler: handler}
	go func() {
		if err := srv.Serve(lis); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			panic(err)
		}
	}()
	done = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			panic(err)
		}
	}
	return done
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// Callback is the callback that is called when authorization succeeds or
// fails.
var Callback = &url.URL{
	Scheme: "http",
	Host:   "127.0.0.1:30428",
	Path:   "/callback",
}

// Client is an OpenID Connect client.
type Client struct {
	// Issuer is the domain name of the OpenID Connect issuer.
	Issuer string

	// ClientID is the client id of the application.
	ClientID string

	// open opens the given url.
	// If not set, an attempt to automatically open the url will be made and if
	// that fails, the user is prompted to manually open the url in their
	// browser. Can be set in tests to capture the generated url.
	open func(url string)
}

// Authorize authorizes a user using OAuth2.
//
// This consists of the following steps:
//   1. Get endpoints from <issuer>/.well-known/openid-configuration.
//   2. Build authorization request.
//   3. Open a browser with the authorize endpoint.
//   4. Start a local webserver to handle callback
//   5. On callback, verify the response and POST to the token endpoint with
//      the received authorization code.
//   6. Extract response and create credentials.
//
// Cancelling the context will terminate the flow and shut down the webserver.
//
// The scope will always include "openid".
//
// The token exchange is done by passing the PKCE challenge:
// https://www.oauth.com/oauth2-servers/pkce/
func (c *Client) Authorize(ctx context.Context, audience string, scope ...string) (*Token, error) {
	provider, err := oidc.NewProvider(ctx, c.Issuer)
	if err != nil {
		return nil, fmt.Errorf("create oidc provider: %v", err)
	}
	oidcConfig := &oidc.Config{
		ClientID: c.ClientID,
	}
	verifier := provider.Verifier(oidcConfig)

	config := oauth2.Config{
		ClientID:    c.ClientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: Callback.String(),
		Scopes:      append([]string{oidc.ScopeOpenID}, scope...),
	}

	codeVerifier, codeChallenge := randVerifier()
	state := randState()

	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("audience", audience),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	authURL := config.AuthCodeURL(state, opts...)

	// Start listener.
	lis, err := net.Listen("tcp", Callback.Host)
	if err != nil {
		return nil, fmt.Errorf("start listener: %v", err)
	}

	// Open url.
	open := c.open
	if open == nil {
		open = systemOpen
	}
	go open(authURL)

	tokenc := make(chan *Token) // Received token
	errc := make(chan error)    // Error respons

	// Start webserver for capturing callback response.
	handler := http.NewServeMux()
	handler.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if e := q.Get("error"); e != "" {
			desc := q.Get("error_description")
			http.Error(w, desc, http.StatusBadRequest)
			errc <- fmt.Errorf(desc)
			return
		}

		// Verify state
		if q.Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			errc <- fmt.Errorf("unexpected state response")
			return
		}

		code := q.Get("code")
		opts := []oauth2.AuthCodeOption{
			oauth2.SetAuthURLParam("code_verifier", codeVerifier),
		}
		oauth2Token, err := config.Exchange(ctx, code, opts...)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusUnauthorized)
			errc <- fmt.Errorf("could not exchange token: %v", err)
			return
		}

		token := &Token{
			OAuth2Token: oauth2Token,
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
			errc <- fmt.Errorf("no id_token field in oauth2 token")
			return
		}
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			errc <- fmt.Errorf("failed to verify id token: %v", err)
			return
		}
		if err := idToken.Claims(&token.IDClaims); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			errc <- fmt.Errorf("failed to extract id claims: %v", err)
			return
		}

		tokenc <- token
		fmt.Fprintln(w, "Logged in. You may now close this tab.")
	})

	go func() {
		errc <- http.Serve(lis, handler)
	}()
	defer func() {
		_ = lis.Close()
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errc:
		return nil, err
	case token := <-tokenc:
		return token, nil
	}
}

func randVerifier() (verifier string, challenge string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	verifier = encode(b)

	h := sha256.New()
	if _, err := h.Write([]byte(verifier)); err != nil {
		panic(err)
	}
	challenge = encode(h.Sum(nil))

	return verifier, challenge
}

func randState() (state string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return encode(b)
}

func encode(b []byte) string {
	v := base64.StdEncoding.EncodeToString(b)
	v = strings.ReplaceAll(v, "+", "-")
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "=", "")
	return v
}

// systemOpen opens the given url with the system browser.
//
// If the url cannot be opened, a message is printed to stderr prompting the
// user to manually open the url.
func systemOpen(url string) {
	fmt.Fprintln(os.Stderr, "Logging you in using your browser")
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "darwin", "windows":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Open the following url in your browser to log in:")
		fmt.Fprintln(os.Stderr, url)
	}
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// Authorize authorizes a user using OpenID Connect PKCE.
func Authorize(ctx context.Context, audience string) (*Credentials, error) {
	provider, err := oidc.NewProvider(ctx, Issuer)
	if err != nil {
		return nil, err
	}
	oidcConfig := &oidc.Config{
		ClientID: ClientID,
	}
	verifier := provider.Verifier(oidcConfig)

	config := oauth2.Config{
		ClientID:    ClientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: "http://127.0.0.1:30428/callback",
		Scopes:      []string{oidc.ScopeOpenID, oidc.ScopeOfflineAccess, "profile", "email"},
	}

	codeVerifier := encode(randBytes(32))
	codeChallenge := encode(s256(codeVerifier))
	state := encode(randBytes(32))

	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("audience", audience),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	u := config.AuthCodeURL(state, opts...)
	open(u)

	credc := make(chan *Credentials)
	errc := make(chan error)

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			errc <- fmt.Errorf("state did not match")
			return
		}

		code := r.URL.Query().Get("code")
		opts := []oauth2.AuthCodeOption{
			oauth2.SetAuthURLParam("code_verifier", codeVerifier),
		}
		oauth2Token, err := config.Exchange(ctx, code, opts...)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			errc <- fmt.Errorf("could not exchange token")
			return
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

		creds := &Credentials{
			ClientID:    ClientID,
			OAuth2Token: oauth2Token,
		}
		if err := idToken.Claims(&creds.IDTokenClaims); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			errc <- err
			return
		}

		credc <- creds
		close(errc)
		close(credc)
		fmt.Fprintln(w, "Logged in. You may now close this tab.")
	})

	go func() {
		errc <- http.ListenAndServe("127.0.0.1:30428", nil)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errc:
		return nil, err
	case creds := <-credc:
		return creds, nil
	}
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

func s256(value string) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(value))
	return h.Sum(nil)
}

func encode(b []byte) string {
	v := base64.StdEncoding.EncodeToString(b)
	v = strings.ReplaceAll(v, "+", "-")
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "=", "")
	return v
}

func open(url string) {
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
		fmt.Fprintln(os.Stderr, "Opening the following url in your browser to log in:")
		fmt.Fprintln(os.Stderr, url)
	}
}

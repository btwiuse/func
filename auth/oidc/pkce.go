package oidc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type PKCE struct {
	Endpoint string
	ClientID string
	Audience string
	Scope    []string

	verifier string
	state    string
}

func (pkce *PKCE) AuthorizeURL(callback *url.URL) (*url.URL, error) {
	verifier := make([]byte, 32)
	if _, err := rand.Read(verifier); err != nil {
		return nil, err
	}
	pkce.verifier = encode(verifier)

	h := sha256.New()
	if _, err := h.Write([]byte(pkce.verifier)); err != nil {
		return nil, err
	}
	s256 := h.Sum(nil)
	challenge := encode(s256)

	state := make([]byte, 32)
	if _, err := rand.Read(state); err != nil {
		return nil, err
	}
	pkce.state = encode(state)

	query := url.Values{}
	query.Add("audience", pkce.Audience)
	query.Add("client_id", pkce.ClientID)
	query.Add("code_challenge", challenge)
	query.Add("code_challenge_method", "S256")
	query.Add("redirect_uri", callback.String())
	query.Add("response_type", "code")
	query.Add("scope", strings.Join(pkce.Scope, " "))
	query.Add("state", pkce.state)

	u, err := url.Parse(pkce.Endpoint)
	if err != nil {
		return nil, err
	}
	u.RawQuery = query.Encode()
	u.Path = "/authorize"

	return u, nil
}

func (pkce *PKCE) HandleCallback(r *http.Request) (*Credentials, error) {
	q := r.URL.Query()
	if errCode := q.Get("error"); errCode != "" {
		reason := q.Get("error_description")
		return nil, fmt.Errorf(reason)
	}

	if q.Get("state") != pkce.state {
		return nil, fmt.Errorf("state does not match")
	}

	body := url.Values{}
	body.Add("grant_type", "authorization_code")
	body.Add("client_id", pkce.ClientID)
	body.Add("code_verifier", pkce.verifier)
	body.Add("code", q.Get("code"))
	body.Add("redirect_uri", "http://localhost:30428/callback")

	resp, err := http.PostForm(pkce.Endpoint+"/oauth/token", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(os.Stderr, resp.Body)
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var token struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		Scope        string `json:"scope"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	// TODO: validate id token?

	return &Credentials{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      token.IDToken,
		Scope:        token.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
		TokenType:    token.TokenType,
	}, nil

}

func open(url string) {
}

func encode(b []byte) string {
	v := base64.StdEncoding.EncodeToString(b)
	v = strings.ReplaceAll(v, "+", "-")
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "=", "")
	return v
}

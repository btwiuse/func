package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/oauth2"
)

// Credentials contain access credentials for an authenticated user.
type Credentials struct {
	API           string                 `json:"api"`
	OAuth2Token   *oauth2.Token          `json:"oauth2_token"`
	IDTokenClaims map[string]interface{} `json:"id_claims"`
}

// Name resolves a user facing name for identifying credentials. The value
// returned is the first available id claim:
//
//   1. nickname
//   2. email
//   3. name
//
// If none of the above id claims have been set, <unknown> is returned.
func (c *Credentials) Name() string {
	if v, ok := c.IDTokenClaims["nickname"].(string); ok {
		return v
	}
	if v, ok := c.IDTokenClaims["email"].(string); ok {
		return v
	}
	if v, ok := c.IDTokenClaims["name"].(string); ok {
		return v
	}
	return "<unknown>"
}

// SetAuthHeader attaches an Authentication header to a request.
func (c *Credentials) SetAuthHeader(r *http.Request) error {
	if c == nil || c.OAuth2Token == nil {
		return fmt.Errorf("token not set")
	}
	c.OAuth2Token.SetAuthHeader(r)
	return nil
}

var nameRe = regexp.MustCompile("[^a-zA-z]+")

func (c *Credentials) filename() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if c.API == "" {
		panic("API not set")
	}
	u, err := url.Parse(c.API)
	if err != nil {
		panic(err)
	}
	name := nameRe.ReplaceAllString(u.Host, "-")
	return filepath.Join(home, ".func", "credentials", name)
}

// Save saves credentials to disk. Overwrites any previous credentials.
func (c *Credentials) Save() error {
	filename := c.filename()
	if err := os.MkdirAll(filepath.Dir(filename), 0744); err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(c)
}

// LoadCredentials loads credentials from disk. Returns nil if no credentials
// exist.
func LoadCredentials() (*Credentials, error) {
	creds := &Credentials{
		API: Audience,
	}
	f, err := os.Open(creds.filename())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if err := json.NewDecoder(f).Decode(creds); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return creds, nil
}

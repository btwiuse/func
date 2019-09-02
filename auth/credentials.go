package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// Credentials contain access credentials for an authenticated user.
type Credentials struct {
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

func (c *Credentials) filename() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return filepath.Join(home, ".func", "credentials")
}

// Save saves credentials to disk. Overwrites any previous credentials.
func (c *Credentials) Save() error {
	f, err := os.Create(c.filename())
	if err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(c)
}

// LoadCredentials loads credentials from disk. Returns nil if no credentials
// exist.
func LoadCredentials() (*Credentials, error) {
	creds := &Credentials{}
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

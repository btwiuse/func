package jwks

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type key struct {
	Alg string   `json:"alg"` // algorithm for the key
	Kty string   `json:"kty"` // key type
	Use string   `json:"use"` // how the key was meant to be used
	X5C []string `json:"x5c"` // x509 certificate chain
	E   string   `json:"e"`   // exponent for a standard pem
	N   string   `json:"n"`   // moduluos for a standard pem
	Kid string   `json:"kid"` // unique identifier for the key
	X5T string   `json:"x5t"` // thumbprint of the x.509 cert (SHA-1 thumbprint)
}

type Doer interface {
	Do(r *http.Request) (*http.Response, error)
}

// JWKS is a JSON Web Key Set
type JWKS struct {
	URL    string
	Client Doer

	keys []key
}

func (j *JWKS) Fetch() error {
	cli := j.Client
	if cli == nil {
		cli = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodGet, j.URL, nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var got struct {
		Keys []key `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		return err
	}
	j.keys = make([]key, 0, len(got.Keys))
	for _, k := range got.Keys {
		if k.Use != "sig" {
			// Not used for signing
			continue
		}
		if k.Kty != "RSA" {
			// Only RSA is supported
			continue
		}
		if k.Kid != "" {
			// No Key id
			continue
		}
		if len(k.X5C) == 0 || k.N == "" || k.E == "" {
			// No valid public key
			continue
		}
		j.keys = append(j.keys, k)
	}
	return nil
}

func (j *JWKS) Cert(kid string) (string, error) {
	if len(j.keys) == 0 {
		if err := j.Fetch(); err != nil {
			return "", fmt.Errorf("fetch: %v", err)
		}
	}
	for _, k := range j.keys {
		if k.Kid == kid {
			cert := "-----BEGIN CERTIFICATE-----\n" + k.X5C[0] + "\n-----END CERTIFICATE-----"
			return cert, nil
		}
	}
	return "", fmt.Errorf("key not found")
}

package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gopkg.in/square/go-jose.v2"
)

// A KeyProvider provides a key for verifying a token.
type KeyProvider interface {
	Key() (interface{}, error)
}

// KeyProviderStatic is a hardcoded key.
type KeyProviderStatic []byte

// Key returns the hardcoded static key.
func (k KeyProviderStatic) Key() (interface{}, error) {
	return []byte(k), nil
}

// KeyProviderFile is a key on disk.
type KeyProviderFile string

// Key reads the contents of the file and returns a key.
func (k KeyProviderFile) Key() (interface{}, error) {
	return ioutil.ReadFile(string(k))
}

// KeyProviderJWKS provides keys from a JSON Web Key Set endpoint.
//
// Keys are fetched as needed and cached.
type KeyProviderJWKS struct {
	// Endpoint to fetch keys from, such as
	// https://<issuer.com>/.well-known/jwks.json
	Endpoint string

	// Expire sets how long keys are valid in the cache. Keys are re-fetched if
	// more than this amount of time passes.
	Expire time.Duration

	updated time.Time
	keyset  *jose.JSONWebKeySet
}

// Key returns a JSON Web Key set. The keys are fetched from the URL if they
// don't exist or if the keys are older than the expire time. Fetched keys are
// cached for a subsequent request.
//
// If expiry is not set, keys expire after 10 minutes.
func (k *KeyProviderJWKS) Key() (interface{}, error) {
	// Get keys if needed
	expiry := k.Expire
	if expiry == 0 {
		expiry = 10 * time.Minute
	}
	if k.keyset == nil || len(k.keyset.Keys) == 0 || k.updated.Before(time.Now().Add(-expiry)) {
		// No suitable keys, fetch keys first.
		if err := k.Update(); err != nil {
			return nil, fmt.Errorf("update keys: %v", err)
		}
	}
	return k.keyset, nil
}

// Update forces a key update even if keys are already present and have not expired.
// Fetched keys are cached.
func (k *KeyProviderJWKS) Update() error {
	resp, err := http.Get(k.Endpoint)
	if err != nil {
		return fmt.Errorf("get keys: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	keyset := &jose.JSONWebKeySet{}
	if err := json.NewDecoder(resp.Body).Decode(keyset); err != nil {
		return fmt.Errorf("decode: %v", err)
	}

	k.updated = time.Now()
	k.keyset = keyset

	return nil
}

package auth

import (
	"fmt"

	"gopkg.in/square/go-jose.v2/jwt"
)

// Claims deserializes a jwt using the provided key.
func Claims(token string, keyProvider KeyProvider, expected jwt.Expected) (jwt.Claims, error) {
	tok, err := jwt.ParseSigned(token)
	if err != nil {
		return jwt.Claims{}, fmt.Errorf("parse token: %v", err)
	}

	key, err := keyProvider.Key()
	if err != nil {
		return jwt.Claims{}, fmt.Errorf("get key: %v", err)
	}

	claims := jwt.Claims{}
	if err := tok.Claims(key, &claims); err != nil {
		return jwt.Claims{}, fmt.Errorf("get claims: %v", err)
	}

	if err := claims.Validate(expected); err != nil {
		return jwt.Claims{}, fmt.Errorf("validate claims: %v", err)
	}

	return claims, nil
}

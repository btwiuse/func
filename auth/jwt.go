package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/func/func/auth/permission"
)

type ctxUserKey int

const (
	userKey ctxUserKey = iota
)

type User struct {
	id          string
	permissions []permission.Permission
}

func (u *User) CheckPermissions(required ...permission.Permission) error {
	var missing []permission.Permission
	for _, p := range required {
		if !u.HasPermission(p) {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing permissions: %v", missing)
	}
	return nil
}

func (u *User) HasPermission(want permission.Permission) bool {
	for _, g := range u.permissions {
		if g == want {
			return true
		}
	}
	return false
}

func ContextWithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func UserFromContext(ctx context.Context) (*User, error) {
	u, ok := ctx.Value(userKey).(*User)
	if !ok {
		return nil, fmt.Errorf("user not set")
	}
	return u, nil
}

// JWTMiddleware extracts and validates a JWT token that's attached to the request as an Authorization header.
//
// If the token is not set or invalid, http.StatusUnauthorized is returned.
// If the token is valid, the token is attached to the request context.
func JWTMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "No Authorization token", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}

		var claims struct {
			jwt.StandardClaims
			Audience    []string                `json:"aud"`
			Issuer      string                  `json:"iss"`
			Permissions []permission.Permission `json:"permissions"`
		}

		_, err := jwt.ParseWithClaims(parts[1], &claims, verify)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if ok := validAudience(claims.Audience); !ok {
			http.Error(w, "Invalid audience", http.StatusUnauthorized)
			return
		}
		if ok := validIssuer(claims.Issuer); !ok {
			http.Error(w, "Invalid issuer", http.StatusUnauthorized)
			return
		}

		u := &User{
			id:          claims.Subject,
			permissions: claims.Permissions,
		}

		ctx := ContextWithUser(r.Context(), u)
		handler(w, r.WithContext(ctx))
	}
}

func validAudience(aud []string) bool {
	for _, aud := range aud {
		if strings.EqualFold(aud, Audience) {
			return true
		}
	}
	return false
}

func validIssuer(iss string) bool {
	if strings.EqualFold(iss, Issuer) {
		return true
	}
	return false
}

func verify(token *jwt.Token) (interface{}, error) {
	cert, err := getPemCert(token)
	if err != nil {
		panic(err.Error())
	}

	result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
	return result, nil
}

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

func getPemCert(token *jwt.Token) (string, error) {
	resp, err := http.Get(Issuer + ".well-known/jwks.json")
	if err != nil {
		return "", err
	}
	var got struct {
		Keys []key `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		return "", err
	}
	for _, k := range got.Keys {
		if k.Kid == token.Header["kid"] {
			cert := "-----BEGIN CERTIFICATE-----\n" + k.X5C[0] + "\n-----END CERTIFICATE-----"
			return cert, nil
		}
	}
	return "", fmt.Errorf("key not found")
}

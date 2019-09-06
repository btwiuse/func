package httpapi

import (
	"net/http"
	"strings"

	"github.com/func/func/auth"
	"gopkg.in/square/go-jose.v2/jwt"
)

// authMiddleware provides middleware for checking an incoming request for a
// valid jwt token.
//
//   1. Token is extracted from the Authorization header
//   2. Token is parsed
//   3. Claims are validated to match
//
// In case the signing key does not match or the claims do not match the
// expected claims, 401 is returned and the wrapped handler func is not called.
//
// When claims are valid, the user is attached on the context. The user can be
// retrieved using UserFromContext.
func authMiddleware(keyProvider auth.KeyProvider, expected jwt.Expected) func(n http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, "No auth header", http.StatusUnauthorized)
				return
			}

			authParts := strings.Split(header, " ")
			if len(authParts) != 2 || !strings.EqualFold(authParts[0], "Bearer") {
				http.Error(w, "Invalid auth header", http.StatusUnauthorized)
				return
			}
			token := authParts[1]

			claims, err := auth.Claims(token, keyProvider, expected)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user := &auth.User{
				ID: claims.Subject,
			}

			ctx := auth.ContextWithUser(r.Context(), user)
			r = r.WithContext(ctx)

			next(w, r)
		}
	}
}

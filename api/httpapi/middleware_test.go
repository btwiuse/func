package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/func/func/auth"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestAuthMiddleware_invalidAuthHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"NoHeader", ""},
		{"InvalidHeader", "foo bar baz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorized := authMiddleware(nil, jwt.Expected{})
			handler := authorized(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("next should not be called")
			})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, authReq(tt.header))

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Status code does not match; got %d, want %d", w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	userID := "testuser"
	key := []byte("secret")
	sig := hs256Signer(t, key)
	token := serializeJWT(t, sig, jwt.Claims{
		Subject:  userID,
		Issuer:   "testissuer",
		Audience: jwt.Audience{"test"},
	})

	authorized := authMiddleware(
		auth.KeyProviderStatic(key),
		jwt.Expected{
			Audience: jwt.Audience([]string{"test"}),
			Issuer:   "testissuer",
		},
	)
	handler := authorized(func(w http.ResponseWriter, r *http.Request) {
		u, err := auth.UserFromContext(r.Context())
		if err != nil {
			t.Errorf("Get user err = %v", err)
		}
		if id := u.ID; id != userID {
			t.Errorf("User id does not match; got %q, want %q", id, userID)
		}
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, authReq("Bearer "+token))

	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
}

func hs256Signer(t *testing.T, key []byte) jose.Signer {
	t.Helper()
	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return sig
}

func serializeJWT(t *testing.T, sig jose.Signer, claims jwt.Claims) string {
	t.Helper()
	token, err := jwt.Signed(sig).Claims(claims).CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func authReq(header string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if header != "" {
		req.Header.Add("Authorization", header)
	}
	return req
}

package testserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/generates"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/models"
	"gopkg.in/oauth2.v3/server"
	"gopkg.in/oauth2.v3/store"
)

// OAuth2 creates a new OAuth2 test server.
func OAuth2(clients ...*models.Client) *http.ServeMux {
	manager := manage.NewDefaultManager()
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	clientStore := store.NewClientStore()
	for _, c := range clients {
		if err := clientStore.Set(c.ID, c); err != nil {
			panic(err)
		}
	}
	manager.MapClientStorage(clientStore)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	keyBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	gen := generates.NewJWTAccessGenerate(keyBytes, jwt.SigningMethodRS256)
	manager.MapAccessGenerate(gen)

	srv := server.NewDefaultServer(manager)
	srv.SetUserAuthorizationHandler(func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		clientID := r.URL.Query().Get("client_id")
		cli, err := clientStore.GetByID(clientID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Client ID %q not found", clientID), http.StatusUnauthorized)
			return
		}
		return cli.GetUserID(), nil
	})

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Printf("testserver.oauth2: Internal Error: %v\n", err)
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Printf("testserver.oauth2: Response Error: %v\n", re.Error)
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		_ = srv.HandleAuthorizeRequest(w, r)
	})

	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = srv.HandleTokenRequest(w, r)
	})

	return mux
}

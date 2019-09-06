package testserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/dgrijalva/jwt-go"
	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/generates"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/models"
	"gopkg.in/oauth2.v3/server"
	"gopkg.in/oauth2.v3/store"
	"gopkg.in/square/go-jose.v2"
)

// OIDC is a test OpenID Connect server.
type OIDC struct {
	Client     *models.Client
	UserClaims map[string]interface{}
}

// Start starts the server. The server is started in a goroutine and can be
// stopped by calling the returned done function.
func (o *OIDC) Start() (issuer string, done func()) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	issuer = "http://" + lis.Addr().String()

	manager := manage.NewDefaultManager()
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	clientStore := store.NewClientStore()
	if o.Client != nil {
		if err := clientStore.Set(o.Client.GetID(), o.Client); err != nil {
			panic(err)
		}
	}
	manager.MapClientStorage(clientStore)

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	keyBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})
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

	srv.SetInternalErrorHandler(func(err error) *errors.Response {
		log.Printf("testserver.oauth2: Internal Error: %v\n", err)
		return nil
	})
	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Printf("testserver.oauth2: Response Error: %v\n", re.Error)
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		_ = srv.HandleAuthorizeRequest(w, r)
	})

	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		// Capture response
		rec := httptest.NewRecorder()
		if err := srv.HandleTokenRequest(rec, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Unmarshal token
		token := make(map[string]interface{})
		if err := json.Unmarshal(rec.Body.Bytes(), &token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add id_token
		idTokenClaims := map[string]interface{}{
			"iss": issuer,
			"aud": o.Client.GetID(),
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		for k, v := range o.UserClaims {
			idTokenClaims[k] = v
		}
		idToken, _ := json.Marshal(idTokenClaims)
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privKey}, nil)
		if err != nil {
			panic(err)
		}
		object, err := signer.Sign(idToken)
		if err != nil {
			panic(err)
		}
		serialized, err := object.CompactSerialize()
		if err != nil {
			panic(err)
		}
		token["id_token"] = serialized

		// Copy headers
		for k, vv := range rec.Header() {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}

		// Output token with id_token
		_ = json.NewEncoder(w).Encode(token)
	})

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		config := struct {
			Issuer        string `json:"issuer"`
			AuthEndpoint  string `json:"authorization_endpoint"`
			TokenEndpoint string `json:"token_endpoint"`
			JWKS          string `json:"jwks_uri"`
		}{
			Issuer:        issuer,
			AuthEndpoint:  issuer + "/authorize",
			TokenEndpoint: issuer + "/oauth/token",
			JWKS:          issuer + "/.well-known/jwks.json",
		}
		_ = json.NewEncoder(w).Encode(config)
	})

	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		set := jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{
				{
					Key:       privKey.Public(),
					KeyID:     "0",
					Algorithm: "RS256",
					Use:       "sig",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(set)
	})

	server := &http.Server{
		Handler: mux,
	}
	go func() {
		_ = server.Serve(lis)
	}()

	done = func() {
		_ = server.Close()
	}

	return issuer, done
}

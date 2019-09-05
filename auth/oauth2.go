package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/oauth2"
)

// DefaultCallback is the callback that is used if no callback has been
// specified.
var DefaultCallback = &url.URL{
	Scheme: "http",
	Host:   "127.0.0.1:30428",
	Path:   "/callback",
}

// Client is an OAuth2 client. The client can authorize a user by providing a
// temporary local webserver for callback.
type Client struct {
	// Enpoint contains the OAuth2 endpoints.
	Endpoint oauth2.Endpoint

	// ClientID to pass to the endpoint on authorization.
	ClientID string

	// Callback is the callback url. The local webserver will start listen on
	// this url. The host must be 127.0.0.1 or localhost.
	// Optional: If callback is not set; DefaultCallback is used.
	Callback *url.URL

	// open opens the url. If nil, the url will automatically be opened.
	// Can be used to mock this behavior in tests.
	open func(url string)
}

// Authorize authorizes a user using OAuth2.
//
// This consists of the following steps:
//   1. Get endpoints from <endpoint>/.well-known/openid-configuration.
//   2. Open a browser with the authorize endpoint.
//   3. Start a local webserver to handle callback
//   4. On callback, verify the response and POST to the token endpoint with
//      the received authorization code.
//   5. Extract response and create credentials.
//
// Cancelling the context will terminate the flow and shut down the webserver.
// Realistically, the roundtrip will take a couple seconds. If the response is
// not received within a short time, something is likely wrong and the callback
// was never called. Passing in a context with a timeout is likely a good idea.
//
// The token exchange is done by passing the PKCE challenge:
// https://www.oauth.com/oauth2-servers/pkce/
func (c *Client) Authorize(ctx context.Context, scopes []string, opts ...oauth2.AuthCodeOption) (*Token, error) {
	cb := c.Callback
	if cb == nil {
		cb = DefaultCallback
	}
	if cb.Hostname() != "127.0.0.1" && cb.Hostname() != "localhost" {
		return nil, fmt.Errorf("callback host must be 127.0.0.1 or localhost")
	}

	config := oauth2.Config{
		ClientID:    c.ClientID,
		Endpoint:    c.Endpoint,
		RedirectURL: cb.String(),
		Scopes:      scopes,
	}

	codeVerifier, codeChallenge := randVerifier()
	state := randState()

	authOpts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	authOpts = append(authOpts, opts...)

	u := config.AuthCodeURL(state, authOpts...)

	// Start listener.
	lis, err := net.Listen("tcp", cb.Host)
	if err != nil {
		return nil, fmt.Errorf("start listener: %v", err)
	}

	// Open url.
	open := c.open
	if open == nil {
		open = systemOpen
	}
	go open(u)

	tokenc := make(chan *Token) // Received token
	errc := make(chan error)    // Error response

	// Start webserver for capturing callback response.
	handler := http.NewServeMux()
	handler.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if e := q.Get("error"); e != "" {
			desc := q.Get("error_description")
			http.Error(w, desc, http.StatusBadRequest)
			errc <- fmt.Errorf(desc)
			return
		}

		// Verify state
		if q.Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			errc <- fmt.Errorf("unexpected state response")
			return
		}

		code := q.Get("code")
		opts := []oauth2.AuthCodeOption{
			oauth2.SetAuthURLParam("code_verifier", codeVerifier),
		}
		oauth2Token, err := config.Exchange(ctx, code, opts...)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusUnauthorized)
			errc <- fmt.Errorf("could not exchange token: %v", err)
			return
		}

		tokenc <- &Token{
			ClientID:     c.ClientID,
			AccessToken:  oauth2Token.AccessToken,
			RefreshToken: oauth2Token.RefreshToken,
			Type:         oauth2Token.TokenType,
			Expiry:       oauth2Token.Expiry,
		}
		fmt.Fprintln(w, "Logged in. You may now close this tab.")
	})

	go func() {
		errc <- http.Serve(lis, handler)
	}()
	defer func() {
		_ = lis.Close()
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errc:
		return nil, err
	case token := <-tokenc:
		return token, nil
	}
}

func randVerifier() (verifier string, challenge string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	verifier = encode(b)

	h := sha256.New()
	if _, err := h.Write([]byte(verifier)); err != nil {
		panic(err)
	}
	challenge = encode(h.Sum(nil))

	return verifier, challenge
}

func randState() (state string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return encode(b)
}

func encode(b []byte) string {
	v := base64.StdEncoding.EncodeToString(b)
	v = strings.ReplaceAll(v, "+", "-")
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "=", "")
	return v
}

// systemOpen opens the given url with the system browser.
//
// If the url cannot be opened, a message is printed to stderr prompting the
// user to manually open the url.
func systemOpen(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "darwin", "windows":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Open the following url in your browser to log in:")
		fmt.Fprintln(os.Stderr, url)
	}
}

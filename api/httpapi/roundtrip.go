package httpapi

import (
	"net/http"

	"github.com/func/func/auth"
)

// AuthRoundTripper is a http.RoundTripper that attaches the token as an
// authorization header to every outgoing request.
type AuthRoundTripper struct {
	// Token is the token to set on the request.
	Token *auth.Token

	// RountTripper is the transport to call with the request. If nil,
	// http.DefaultTransport is used.
	RoundTripper http.RoundTripper
}

// RoundTrip sets the authorization header on the request and calls the proxied
// RoundTripper. Uses http.DefaultTransport if the RoundTripper is not set.
func (rt *AuthRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Add("Authorization", rt.Token.Type+" "+rt.Token.AccessToken)
	proxy := rt.RoundTripper
	if proxy == nil {
		proxy = http.DefaultTransport
	}
	return proxy.RoundTrip(r)
}

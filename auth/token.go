package auth

import "time"

// Token is an OAuth2 token for an authorized user.
type Token struct {
	// ClientID is the client id that the token was assigned to.
	ClientID string

	// AccessToken is the token that authorizes and authenticates
	// the requests.
	AccessToken string

	// Type is the type of token.
	// The Type method returns either this or "Bearer", the default.
	Type string

	// RefreshToken is a token that's used by the application
	// (as opposed to the user) to refresh the access token
	// if it expires.
	RefreshToken string

	// Expiry is the optional expiration time of the access token.
	//
	// If zero, TokenSource implementations will reuse the same
	// token forever and RefreshToken or equivalent
	// mechanisms for that TokenSource will not be used.
	Expiry time.Time
}

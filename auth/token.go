package auth

import (
	"golang.org/x/oauth2"
)

// Token is an token for an authorized user.
type Token struct {
	OAuth2Token *oauth2.Token
	IDClaims    map[string]interface{}
}

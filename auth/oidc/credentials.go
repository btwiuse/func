package oidc

import (
	"net/http"
	"time"
)

type Credentials struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	Scope        string
	ExpiresAt    time.Time
	TokenType    string
}

func (c *Credentials) Attach(r *http.Request) error {
	r.Header.Add("Authorization", c.TokenType+" "+c.AccessToken)
	return nil
}

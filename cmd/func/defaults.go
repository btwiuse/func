package main

// Default flag values.
// Set in build using -ldflags "-X main.<name>=<value>"
var (
	DefaultEndpoint = "" // API Endpoint
	DefaultIssuer   = "" // OpenID Connect endpoint
	DefaultClientID = "" // OpenID Connect client id
)

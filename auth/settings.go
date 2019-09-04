package auth

// The following variables should be set when compiling:
//
//   go build -ldflags="-X auth.<name>=<value>"
var (
	// Issuer for auth tokens.
	Issuer = ""

	// ClientID for authorization.
	ClientID = ""

	// Audience is the target audience.
	Audience = ""
)

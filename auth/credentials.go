package auth

import "net/http"

type Credentials interface {
	Attach(req *http.Request) error
}

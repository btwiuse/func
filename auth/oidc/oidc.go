package oidc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
)

type Flow interface {
	AuthorizeURL(callback *url.URL) (*url.URL, error)
	HandleCallback(r *http.Request) (*Credentials, error)
}

type Opener interface {
	Open(url *url.URL)
}

type Authorizer struct {
	Flow   Flow
	Opener Opener

	LocalPort int
	LocalPath string

	credsc chan Credentials
	errc   chan error
}

func NewAuthorizer(flow Flow) *Authorizer {
	return &Authorizer{
		Flow:   flow,
		Opener: DefaultOpener,

		LocalPort: 30428,
		LocalPath: "/callback",
	}
}

func (a *Authorizer) Authorize(ctx context.Context) (*Credentials, error) {
	callback := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", a.LocalPort),
		Path:   a.LocalPath,
	}

	u, err := a.Flow.AuthorizeURL(callback)
	if err != nil {
		return nil, err
	}

	a.Opener.Open(u)

	credc := make(chan *Credentials)
	errc := make(chan error)

	go func() {
		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			creds, err := a.Flow.HandleCallback(r)
			if err != nil {
				http.Error(w, "Auth failed", http.StatusInternalServerError)
			}
			credc <- creds

			close(errc)
		})

		errc <- http.ListenAndServe(fmt.Sprintf(":%d", a.LocalPort), nil)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errc:
		return nil, err
	case creds := <-credc:
		return creds, nil
	}
}

type OpenFunc func(url *url.URL)

func (fn OpenFunc) Open(url *url.URL) { fn(url) }

var DefaultOpener = OpenFunc(func(url *url.URL) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url.String()).Start()
	case "darwin", "windows":
		err = exec.Command("open", url.String()).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Opening the following url in your browser to log in:")
		fmt.Fprintln(os.Stderr, url.String())
	}
})

package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

type Auth struct {
	Endpoint string

	ctx         context.Context
	redirectURI string
	server      *http.Server

	err chan error
}

func (a *Auth) Login(ctx context.Context) error {
	a.ctx = ctx
	a.err = make(chan error)

	port := "30428"
	server := &http.Server{
		Addr:    ":" + port,
		Handler: http.HandlerFunc(a.handleRequest),
	}
	a.server = server
	a.redirectURI = fmt.Sprintf("http://localhost:%s/callback", port)

	if err := a.authorize(); err != nil {
		return err
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			a.err <- err
		}
	}()

	select {
	case err := <-a.err:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *Auth) authorize() error {
	fmt.Fprintln(os.Stderr, "Logging you in via the browser")

	params := url.Values{}
	params.Add("redirect_uri", a.redirectURI)
	params.Add("response_mode", "form_post")
	endpoint := fmt.Sprintf("%s/authorize?%s", a.Endpoint, params.Encode())
	if err := exec.Command("xdg-open", endpoint).Run(); err != nil {
		return err
	}
	return nil
}

func (a *Auth) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		a.err <- fmt.Errorf("not post")
		return
	}
	if r.URL.Path != "/callback" {
		http.NotFound(w, r)
		a.err <- fmt.Errorf("not callback")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not parse form data", http.StatusBadRequest)
		a.err <- err
		return
	}

	if errCode := r.Form.Get("error"); errCode != "" {
		errReason := r.Form.Get("error_description")
		fmt.Fprintln(w, "login cancelled")
		a.err <- fmt.Errorf(errReason)
		return
	}

	code := r.Form.Get("code")
	if code == "" {
		http.Error(w, "Code not set", http.StatusBadRequest)
		a.err <- fmt.Errorf("code not set")
		return
	}

	go a.exchangeToken(code)

	fmt.Fprint(w, "ok")
}

func (a *Auth) shutdownServer() {
	ctx := context.Background()
	_ = a.server.Shutdown(ctx)
}

func (a *Auth) exchangeToken(code string) {
	if a.ctx.Err() != nil {
		return
	}

	params := url.Values{}
	params.Add("code", code)
	params.Add("redirect_uri", a.redirectURI)
	resp, err := http.Get(fmt.Sprintf("%s/token?%s", a.Endpoint, params.Encode()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}
	_, _ = io.Copy(os.Stdout, resp.Body)
	os.Exit(0)
}

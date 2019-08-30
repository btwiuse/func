package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

type Auth struct {
	Endpoint string

	callbackURI string
	err         chan error
}

func (a *Auth) Authorize() error {
	a.err = make(chan error)

	server := &http.Server{
		Addr:    ":30428",
		Handler: http.HandlerFunc(a.handleRequest),
	}
	a.callbackURI = fmt.Sprintf("http://localhost%s/callback", server.Addr)

	fmt.Fprintln(os.Stderr, "Logging you in via the browser")

	params := url.Values{}
	params.Add("redirect_uri", a.callbackURI)
	params.Add("response_mode", "form_post")
	endpoint := fmt.Sprintf("%s/authorize?%s", a.Endpoint, params.Encode())
	if err := exec.Command("xdg-open", endpoint).Run(); err != nil {
		return err
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			a.err <- err
		}
	}()

	if err := <-a.err; err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Logged in")

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

	fmt.Fprint(w, "ok")

	go func() {
		if err := a.exchangeToken(code); err != nil {
			a.err <- err
		}
		close(a.err)
	}()
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (a *Auth) exchangeToken(code string) error {
	params := url.Values{}
	params.Add("code", code)
	params.Add("redirect_uri", a.callbackURI)
	resp, err := http.Get(fmt.Sprintf("%s/token?%s", a.Endpoint, params.Encode()))
	if err != nil {
		return fmt.Errorf("get token: %v", err)
	}

	var tokens tokenResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&tokens); err != nil {
		return fmt.Errorf("decode tokens: %v", err)
	}
	_ = resp.Body.Close()

	return a.saveTokens(tokens)
}

func (a *Auth) saveTokens(tokens tokenResponse) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %v", err)
	}

	dir := filepath.Join(home, ".func")
	if err := os.MkdirAll(dir, 0744); err != nil {
		return fmt.Errorf("create .func dir: %v", err)
	}

	b, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("marshal tokens: %v", err)
	}

	if err := ioutil.WriteFile(filepath.Join(dir, "credentials"), b, 0644); err != nil {
		return fmt.Errorf("write credentials: %v", err)
	}

	return nil
}

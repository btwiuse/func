package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// ErrtokenNotFound is returned when attempting to read a token from disk that
// does not exist.
var ErrTokenNotFound = errors.New("token not found")

// Token is an token for an authorized user.
type Token struct {
	TokenEndpoint string                 `json:"token_endpoint"`
	OAuth2Token   *oauth2.Token          `json:"oauth2_token"`
	IDClaims      map[string]interface{} `json:"id_claims"`
}

func (t *Token) Persist(clientID string) error {
	file := tokenFilename(clientID)
	if err := os.MkdirAll(filepath.Dir(file), 0744); err != nil {
		return err
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
	}()
	_, err = t.WriteTo(f)
	return err
}

// request will fail if auth token does not exist
func HTTPClient(clientID string) (*http.Client, error) {
	file := tokenFilename(clientID)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	tok := &Token{}
	if err := json.Unmarshal(b, &tok); err != nil {
		return nil, err
	}
	cfg := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			TokenURL: tok.TokenEndpoint,
		},
	}
	ctx := context.Background()
	src := cfg.TokenSource(ctx, tok.OAuth2Token)
	return oauth2.NewClient(ctx, src), nil
}

func tokenFilename(clientID string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	dir := filepath.Join(home, ".func", "credentials")
	return filepath.Join(dir, clientID)
}

// WriteTo writes the token to the given writer.
// The method implements io.WriterTo but always returns 0 as number of bytes
// written.
func (t *Token) WriteTo(w io.Writer) (int64, error) {
	wc := &writeCounter{w: w}
	err := json.NewEncoder(wc).Encode(t)
	return int64(wc.n), err
}

// ReadFrom reads a token from a reader.
func (t *Token) ReadFrom(r io.Reader) (int64, error) {
	rc := &readCounter{r: r}
	err := json.NewDecoder(rc).Decode(t)
	return int64(rc.n), err
}

// writeCounter is an io.Writer that keeps track of number of bytes written.
type writeCounter struct {
	w io.Writer
	n int
}

func (c *writeCounter) Write(b []byte) (int, error) {
	n, err := c.w.Write(b)
	if err != nil {
		return n, err
	}
	c.n += n
	return n, nil
}

// readCounter is an io.Writer that keeps track of number of bytes read.
type readCounter struct {
	r io.Reader
	n int
}

func (c *readCounter) Read(b []byte) (int, error) {
	n, err := c.r.Read(b)
	if err != nil {
		return n, err
	}
	c.n += n
	return n, nil
}

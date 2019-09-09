package auth

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// Token is an token for an authorized user.
type Token struct {
	OAuth2Token *oauth2.Token          `json:"oauth2_token"`
	IDClaims    map[string]interface{} `json:"id_claims"`
}

func (t *Token) Persist(dir, audience, clientID string) error {
	file := filepath.Join(dir, TokenFilename(audience, clientID))
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

func TokenFilename(audience, clientID string) string {
	u, err := url.Parse(audience)
	if err != nil {
		panic(err)
	}
	return filepath.Join(u.Host, clientID)
}

// LoadToken loads a token from disk for a given audience and client id.
// Returns an error only if the token could not be read; returns nil if a token
// for the given audience / client id does not exist
func LoadToken(dir, audience, clientID string) (*Token, error) {
	file := filepath.Join(dir, TokenFilename(audience, clientID))
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() {
		err = f.Close()
	}()
	t := &Token{}
	if _, err := t.ReadFrom(f); err != nil {
		return nil, err
	}
	return t, nil
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

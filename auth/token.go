package auth

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Token is an OAuth2 token for an authorized user.
type Token struct {
	// ClientID is the client id that the token was assigned to.
	ClientID string

	// AccessToken is the token that authorizes and authenticates
	// the requests.
	AccessToken string

	// Type is the type of token.
	// The Type method returns either this or "Bearer", the default.
	Type string

	// RefreshToken is a token that's used by the application
	// (as opposed to the user) to refresh the access token
	// if it expires.
	RefreshToken string

	// Expiry is the optional expiration time of the access token.
	//
	// If zero, TokenSource implementations will reuse the same
	// token forever and RefreshToken or equivalent
	// mechanisms for that TokenSource will not be used.
	Expiry time.Time
}

// TokenFilename returns the filename on disk for a token. Panics if the
// clientID is not set, or if the user's homedir cannot be resolved.
func TokenFilename(clientID string) string {
	if clientID == "" {
		panic("ClientID not set")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, ".func", "credentials", clientID)
}

// TokenFromFile loads a token from disk for a given client id. Returns an
// error only if the token could not be read; returns nil if a token for the
// given client id does not exist
func TokenFromFile(filename string) (tok *Token, err error) {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() {
		err = f.Close()
	}()
	tok = &Token{}
	if _, err := tok.ReadFrom(f); err != nil {
		return nil, err
	}
	return tok, nil
}

// SaveToFile saves a token to the given file name.
// The file and any required directories are created if necessary. The file is
// overwritten if it already exists.
func (t *Token) SaveToFile(filename string) (err error) {
	if err := os.MkdirAll(filepath.Dir(filename), 0744); err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
	}()
	_, err = t.WriteTo(f)
	return err
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

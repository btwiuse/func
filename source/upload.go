package source

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Upload uploads data to the given url using HTTP PUT. Any headers provided
// will be added to the request. If data is a *bytes.Buffer, the Content-Length
// header is automatically set.
//
// If the server does not support Transfer-Encoding: chunked, data must be a
// *bytes.Buffer, *bytes.Reader or *strings.Reader.
//
// If client is nil, http.DefaultClient is used.
func Upload(ctx context.Context, client doer, url string, headers map[string]string, data io.Reader) error {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest(http.MethodPut, url, data)
	if err != nil {
		return errors.Wrap(err, "create request")
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return errors.Wrap(err, "upload")
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.Errorf("received unexpected status %v", resp.StatusCode)
		if body, berr := ioutil.ReadAll(resp.Body); berr == nil {
			err = errors.Errorf("%v: %s", err, string(bytes.TrimSpace(body)))
		}
		return err
	}

	if _, err = io.Copy(ioutil.Discard, resp.Body); err != nil {
		return errors.Wrap(err, "discard body")
	}
	if err := resp.Body.Close(); err != nil {
		return errors.Wrap(err, "close body")
	}

	return nil
}

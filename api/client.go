package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// SourceProvider provides source code for upload.
type SourceProvider interface {
	Source(sha string) *bytes.Buffer
}

// Client is a func api client.
type Client struct {
	API    API
	Logger *zap.Logger
	Source SourceProvider
}

// Apply applies the given hcl configuration.
//
// If source code is needed, source is uploaded. After upload, apply is
// retried.
func (c *Client) Apply(ctx context.Context, req *ApplyRequest) error {
	logger := c.Logger

	logger.Info("Apply")
	for {
		resp, err := c.API.Apply(ctx, req)
		if err != nil {
			return err
		}

		if len(resp.SourcesRequired) > 0 {
			logger.Debug(fmt.Sprintf("%d Sources required", len(resp.SourcesRequired)))

			if err := c.uploadSources(ctx, resp.SourcesRequired); err != nil {
				return errors.Wrap(err, "upload source")
			}

			// Retry after source files have been uploaded
			logger.Debug("Retry request with sources uploaded")
			continue
		}
		break
	}

	return nil
}

func (c *Client) uploadSources(ctx context.Context, srcs []*SourceRequest) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, src := range srcs {
		src := src
		g.Go(func() error {
			return c.uploadSource(ctx, src)
		})
	}
	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "upload source")
	}
	return nil
}

func (c *Client) uploadSource(ctx context.Context, src *SourceRequest) error {
	logger := c.Logger
	logger.Debug(fmt.Sprintf("Uploading %s", src.Key))

	data := c.Source.Source(src.Key)

	req, err := http.NewRequest(http.MethodPut, src.URL, data)
	if err != nil {
		return err
	}
	for k, v := range src.Headers {
		req.Header.Add(k, v)
	}

	start := time.Now()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 4 * time.Second,
			ResponseHeaderTimeout: 3 * time.Second,
		},
		// Prevent endless redirects
		Timeout: 10 * time.Minute,
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return errors.Wrap(err, "upload")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		logger.Error(
			"Received unexected status",
			zap.Int("code", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return errors.Errorf("received unexpected status %v", resp.StatusCode)
	}

	_, _ = io.Copy(ioutil.Discard, resp.Body)
	_ = resp.Body.Close()

	logger.Debug(fmt.Sprintf("Uploading %s completed after %s", src.Key, time.Since(start)))
	return nil
}

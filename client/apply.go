package client

import (
	"context"
	"net/http"

	"github.com/cenkalti/backoff"
	"github.com/func/func/api"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Apply applies changes to the resources in the given root directory and all
// child directories.
//
// The resources are loaded and applied to a namespace. In case source code is
// required, source code is collected and uploaded.
func (cli *Client) Apply(ctx context.Context, rootDir, namespace string) error {
	cli.once.Do(cli.init)

	body, diags := cli.Loader.Load(rootDir)
	if diags.HasErrors() {
		return cli.errDiagnostics(diags)
	}

	req := &api.ApplyRequest{Namespace: namespace, Config: body}

	op := func() error {
		resp, err := cli.API.Apply(ctx, req)
		if err != nil {
			if diags, ok := err.(hcl.Diagnostics); ok {
				return backoff.Permanent(cli.errDiagnostics(diags))
			}
			return errors.Wrap(err, "apply")
		}

		if len(resp.SourcesRequired) > 0 {
			g, uctx := errgroup.WithContext(ctx)
			for _, sr := range resp.SourcesRequired {
				sr := sr
				g.Go(func() error {
					src := cli.Loader.Source(sr.Key)
					err := source.Upload(
						uctx,
						http.DefaultClient,
						sr.URL,
						sr.Headers,
						src,
					)
					if err != nil {
						return errors.Wrapf(err, "Upload %s", sr.Key)
					}
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return errors.Wrap(err, "upload source")
			}
			return errors.New("retry after source")
		}

		return nil
	}

	algo := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)

	return backoff.Retry(op, algo)
}

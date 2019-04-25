package client

import (
	"context"
	"net/http"

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

	resp, err := cli.API.Apply(ctx, req)
	if err != nil {
		if diags, ok := err.(hcl.Diagnostics); ok {
			return cli.errDiagnostics(diags)
		}
		return errors.Wrap(err, "apply")
	}

	if len(resp.SourcesRequired) > 0 {
		g, ctx := errgroup.WithContext(ctx)
		for _, sr := range resp.SourcesRequired {
			sr := sr
			g.Go(func() error {
				src := cli.Loader.Source(sr.Digest)
				err := source.Upload(
					ctx,
					http.DefaultClient,
					sr.URL,
					sr.Headers,
					src,
				)
				if err != nil {
					return errors.Wrapf(err, "Upload %s", sr.Digest)
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return errors.Wrap(err, "upload source")
		}
		// Try again
		return cli.Apply(ctx, rootDir, namespace)
	}

	return nil
}

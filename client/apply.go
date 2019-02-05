package client

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/func/func/api"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/pkg/errors"
	"github.com/twitchtv/twirp"
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

	j, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req := &api.ApplyRequest{Namespace: namespace, Config: j}

	resp, err := cli.API.Apply(ctx, req)
	if err != nil {
		if twerr, ok := err.(twirp.Error); ok {
			if diagJSON := twerr.Meta("diagnostics"); diagJSON != "" {
				var diags hcl.Diagnostics
				if derr := json.Unmarshal([]byte(diagJSON), &diags); derr == nil {
					return cli.errDiagnostics(diags)
				}
			}
		}
		return err
	}

	if srcReq := resp.GetSourceRequest(); srcReq != nil {
		if err = cli.upload(ctx, srcReq); err != nil {
			return errors.Wrap(err, "upload source")
		}
		resp, err = cli.API.Apply(ctx, req)
		if err != nil {
			return err
		}
	}

	_ = resp

	return nil
}

// upload concurrently uploads all the requested sources.
func (cli *Client) upload(ctx context.Context, srcReq *api.SourceRequired) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, ur := range srcReq.GetUploads() {
		ur := ur
		g.Go(func() error {
			src := cli.Loader.Source(ur.GetDigest())
			err := source.Upload(
				ctx,
				http.DefaultClient,
				ur.GetUrl(),
				ur.GetHeaders(),
				src,
			)
			if err != nil {
				return errors.Wrap(err, ur.GetDigest())
			}
			return nil
		})
	}
	return g.Wait()
}

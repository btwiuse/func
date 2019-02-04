package client

import (
	"github.com/func/func/config"
	"github.com/hashicorp/hcl2/gohcl"
)

// ParseConfig finds all configuration files in the given root directory and
// parses them to a root configuration.
func (cli *Client) ParseConfig(rootDir string) (*config.Root, error) {
	cli.once.Do(cli.init)

	body, diags := cli.Loader.Load(rootDir)
	if diags.HasErrors() {
		return nil, cli.errDiagnostics(diags)
	}

	var cfgRoot config.Root
	diags = gohcl.DecodeBody(body, nil, &cfgRoot)
	if diags.HasErrors() {
		return nil, cli.errDiagnostics(diags)
	}

	return &cfgRoot, nil
}

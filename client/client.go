// Package client provides a concise interface to all client side logic.
package client

import (
	"bytes"
	"sync"

	"github.com/func/func/api"
	"github.com/func/func/config"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclpack"
)

// Client is a func client.
type Client struct {
	// API client to use.
	API api.Func

	// Loader allows overriding the configuration loader to use. Can be used to
	// replace the loader in tests, but otherwise should be left nil. If nil, a
	// default loader use used instead.
	Loader ConfigLoader

	// once is used to initialize default values, allowing the nil value to be
	// useful.
	once sync.Once
}

// ConfigLoader is used when loading configuration files from disk.
type ConfigLoader interface {
	Load(dir string) (*hclpack.Body, hcl.Diagnostics)
	Root(dir string) (string, hcl.Diagnostics)
	Source(sha string) *bytes.Buffer
	Files() map[string]*hcl.File
}

func (cli *Client) init() {
	if cli.Loader == nil {
		cli.Loader = &config.Loader{}
	}
}

func (cli *Client) errDiagnostics(diags hcl.Diagnostics) *DiagnosticsError {
	cli.once.Do(cli.init)
	return &DiagnosticsError{loader: cli.Loader, Diagnostics: diags}
}

package client

import (
	"path/filepath"
)

// FindRoot searches for the root directory in the given dir. If a root is not
// found, parent directories are traversed until a root directory is found.
func (cli *Client) FindRoot(dir string) (string, error) {
	cli.once.Do(cli.init)

	rootDir, diags := cli.Loader.Root(dir)
	if diags.HasErrors() {
		return "", cli.errDiagnostics(diags)
	}

	return filepath.Clean(rootDir), nil
}

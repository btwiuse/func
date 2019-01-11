package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/func/func/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var rootCommand = &cobra.Command{
	Use:   "root [dir]",
	Short: "Print project root directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			fmt.Fprintln(os.Stderr, cmd.UsageString())
			os.Exit(2)
		}
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		return runRoot(target)
	},
}

func init() {
	Func.AddCommand(rootCommand)
}

func runRoot(target string) error {
	l := &config.Loader{}
	root, err := l.Root(target)
	if err != nil {
		if errors.Cause(err) == io.EOF {
			return errors.Errorf("project not found in %s", target)
		}
		return err
	}
	fmt.Println(filepath.Clean(root))
	return nil
}

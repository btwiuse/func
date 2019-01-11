package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/func/func/config"
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
		return err
	}
	fmt.Println(filepath.Clean(root))
	return nil
}

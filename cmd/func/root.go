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
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) { // nolint: unparam
		if len(args) == 0 {
			args = []string{"."}
		}

		loader := &config.Loader{}

		root, err := loader.Root(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if abs, err := filepath.Abs(root); err == nil {
			root = abs
		}
		fmt.Println(root)
	},
}

func init() {
	Func.AddCommand(rootCommand)
}

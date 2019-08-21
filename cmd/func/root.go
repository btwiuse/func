package cmd

import (
	"fmt"
	"os"

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

		rootDir, err := loader.Root(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if rootDir == "" {
			fmt.Fprintln(os.Stderr, "Project not found")
			os.Exit(2)
		}
		fmt.Println(rootDir)
	},
}

func init() {
	Func.AddCommand(rootCommand)
}

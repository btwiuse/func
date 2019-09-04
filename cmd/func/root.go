package main

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

		p, err := config.FindProject(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if p == nil {
			fmt.Fprintln(os.Stderr, "Project not found")
			os.Exit(2)
		}
		fmt.Println(p.RootDir)
	},
}

func init() {
	cmd.AddCommand(rootCommand)
}

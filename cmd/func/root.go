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
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Fprintln(os.Stderr, cmd.UsageString())
			os.Exit(2)
		}
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		runRoot(target)
	},
}

func init() {
	Func.AddCommand(rootCommand)
}

func runRoot(target string) {
	l := &config.Loader{}
	rootDir, diags := l.Root(target)
	if diags.HasErrors() {
		l.PrintDiagnostics(os.Stderr, diags)
		os.Exit(1)
	}
	fmt.Println(filepath.Clean(rootDir))
}

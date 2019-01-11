package cmd

import (
	"fmt"
	"os"

	"github.com/func/func/config"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/spf13/cobra"
)

var listCommand = &cobra.Command{
	Use:     "list [dir]",
	Aliases: []string{"ls"},
	Short:   "List resources",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Fprintln(os.Stderr, cmd.UsageString())
			os.Exit(2)
		}
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		runList(target)
	},
}

func init() {
	Func.AddCommand(listCommand)
}

func runList(target string) {
	l := &config.Loader{}

	rootDir, diags := l.Root(target)
	if diags.HasErrors() {
		l.PrintDiagnostics(os.Stderr, diags)
		os.Exit(1)
	}

	body, diags := l.Load(rootDir)
	if diags.HasErrors() {
		l.PrintDiagnostics(os.Stderr, diags)
		os.Exit(1)
	}

	var root config.Root
	diags = gohcl.DecodeBody(body, nil, &root)
	if diags.HasErrors() {
		l.PrintDiagnostics(os.Stderr, diags)
		os.Exit(1)
	}
	fmt.Printf("Project %s\n", root.Project.Name)
	for _, r := range root.Resources {
		fmt.Printf("Resource %s %s\n", r.Type, r.Name)
	}
}

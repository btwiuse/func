package cmd

import (
	"fmt"
	"os"

	"github.com/func/func/config"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var listCommand = &cobra.Command{
	Use:     "list [dir]",
	Aliases: []string{"ls"},
	Short:   "List resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			fmt.Fprintln(os.Stderr, cmd.UsageString())
			os.Exit(2)
		}
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		return runList(target)
	},
}

func init() {
	Func.AddCommand(listCommand)
}

func runList(target string) error {
	l := &config.Loader{}
	rootDir, err := l.Root(target)
	if err != nil {
		return err
	}
	body, err := l.Load(rootDir)
	if err != nil {
		return err
	}
	var root config.Root
	diags := gohcl.DecodeBody(body, nil, &root)
	if diags.HasErrors() {
		cols, _, err := terminal.GetSize(0)
		if err != nil {
			cols = 80
		}
		wr := hcl.NewDiagnosticTextWriter(
			os.Stdout,
			l.Files(),
			uint(cols),
			true,
		)
		if err := wr.WriteDiagnostics(diags); err != nil {
			return err
		}
		os.Exit(1)
	}
	fmt.Printf("Project %s\n", root.Project.Name)
	for _, r := range root.Resources {
		fmt.Printf("Resource %s %s\n", r.Type, r.Name)
	}
	return nil
}

package cmd

import (
	"fmt"
	"os"

	"github.com/func/func/client"
	"github.com/spf13/cobra"
)

var listCommand = &cobra.Command{
	Use:     "list [dir]",
	Aliases: []string{"ls"},
	Short:   "List resources",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) { // nolint: unparam
		if len(args) == 0 {
			args = []string{"."}
		}

		cli := &client.Client{}

		rootDir, err := cli.FindRoot(args[0])
		if err != nil {
			fatal(err)
		}

		cfg, err := cli.ParseConfig(rootDir)
		if err != nil {
			fatal(err)
		}

		fmt.Fprintf(os.Stdout, "Project %s\n", cfg.Project.Name)
		for _, r := range cfg.Resources {
			fmt.Fprintf(os.Stdout, "Resource %s %s\n", r.Type, r.Name)
		}
	},
}

func init() {
	Func.AddCommand(listCommand)
}

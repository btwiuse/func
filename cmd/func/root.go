package cmd

import (
	"fmt"

	"github.com/func/func/client"
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

		cli := &client.Client{}
		root, err := cli.FindRoot(args[0])
		if err != nil {
			fatal(err)
		}
		fmt.Println(root)
	},
}

func init() {
	Func.AddCommand(rootCommand)
}

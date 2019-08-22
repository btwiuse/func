package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/func/func/config"
	"github.com/spf13/cobra"
)

var projectCommand = &cobra.Command{
	Use:   "project",
	Short: "Manage func project",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			args = []string{"."}
		}

		dir := args[0]
		project, err := config.FindProject(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if project == nil {
			green := color.New(color.FgGreen).SprintFunc()
			fmt.Fprintln(os.Stderr, "Project not found")
			fmt.Fprintf(os.Stderr, "Set up a new project with %s\n", green("func project new"))
			os.Exit(2)
			return
		}

		cyan := color.New(color.FgCyan).SprintFunc()
		faint := color.New(color.Faint).SprintFunc()

		fmt.Printf("Name: %s\n", cyan(project.Name))
		fmt.Printf("Root: %s\n", faint(project.RootDir))
	},
}

func init() {
	Func.AddCommand(projectCommand)
}

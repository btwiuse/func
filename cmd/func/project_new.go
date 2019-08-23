package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/func/func/config"
	"github.com/spf13/cobra"
)

var projectNewCommand = &cobra.Command{
	Use:   "new [dir]",
	Short: "Create a new project",
	Args:  cobra.MaximumNArgs(1),
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
		if project != nil {
			fmt.Fprintf(os.Stderr, "Project already found in %s\n", project.RootDir)
			os.Exit(2)
		}

		if dir == "." {
			d, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			dir = d
		}

		// Error is ignored, command can be triggered from apply that doesn't
		// specify the flag.
		name, _ := cmd.Flags().GetString("name")

		if name == "" {
			faint := color.New(color.Faint).SprintFunc()
			green := color.New(color.FgGreen).SprintFunc()
			cyan := color.New(color.FgCyan).SprintFunc()

			defaultName := filepath.Base(dir)
			reader := bufio.NewReader(os.Stdin)
			fmt.Fprint(os.Stderr, "Set up new project in "+green(dir)+"\n")
			fmt.Fprint(os.Stderr, faint("Cancel with ctrl-c\n\n"))
			fmt.Fprintf(os.Stderr, faint("› ")+"Project name [%s]: ", cyan(defaultName))
			name, _ = reader.ReadString('\n')
			name = strings.TrimSuffix(name, "\n")
			if name == "" {
				name = defaultName
			}
		}

		name = strings.TrimSpace(name)
		if len(name) == 0 {
			fmt.Fprintln(os.Stderr, "Project name must be set")
			os.Exit(1)
		}

		abs, err := filepath.Abs(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		project = &config.Project{
			Name:    name,
			RootDir: abs,
		}
		if err := project.Write(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	projectNewCommand.PersistentFlags().String("name", "", "New project name")
	projectCommand.AddCommand(projectNewCommand)
}

package main

import (
	"fmt"
	"os"

	"github.com/func/func/auth"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:           "func",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cmd.PersistentFlags().String("endpoint", auth.Audience, "Server endpoint")
}

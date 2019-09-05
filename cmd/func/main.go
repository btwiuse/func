package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// DefaultEndpoint is the default value for all api endpoint flags.
var DefaultEndpoint = "https://api.func.io"

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

package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version contains the current version.
	Version = "dev"

	// BuildDate contains a string with the build date.
	BuildDate = "unknown"
)

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("func\n")
		fmt.Printf("  Version:     %s\n", Version)
		fmt.Printf("  Built:       %s\n", BuildDate)
		fmt.Printf("  Go version:  %s\n", runtime.Version())
	},
}

func init() {
	cmd.AddCommand(versionCommand)
}

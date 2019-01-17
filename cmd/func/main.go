package cmd

import (
	"github.com/spf13/cobra"
)

// Func is the main binary entrypoint.
var Func = &cobra.Command{
	Use:           "func",
	SilenceErrors: true,
	SilenceUsage:  true,
}

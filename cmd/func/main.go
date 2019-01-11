package cmd

import (
	"github.com/spf13/cobra"
)

var Func = &cobra.Command{
	Use:           "func",
	SilenceErrors: true,
	SilenceUsage:  true,
}

package cmd

import (
	"log"
	"os"

	"github.com/func/func/client"
	"github.com/spf13/cobra"
)

// Func is the main binary entrypoint.
var Func = &cobra.Command{
	Use:           "func",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func fatal(err error) {
	if derr, ok := err.(*client.DiagnosticsError); ok {
		if err := derr.PrintDiagnostics(os.Stderr); err != nil {
			log.Fatal(err)
		}
		os.Exit(1)
	}
	log.Fatal(err)
}

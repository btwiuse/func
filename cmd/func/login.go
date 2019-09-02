package main

import (
	"context"
	"fmt"
	"os"

	"github.com/func/func/auth"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var loginCommand = &cobra.Command{
	Use:   "login",
	Short: "Login to func service",
	Run: func(cmd *cobra.Command, args []string) {
		apiEndpoint, err := cmd.Flags().GetString("endpoint")
		if err != nil {
			panic(err)
		}

		ex, err := auth.LoadCredentials()
		if err == nil && ex != nil && isatty.IsTerminal(os.Stdout.Fd()) {
			fmt.Fprintf(os.Stderr, "Already logged in as %s\n\n", ex.Name())
			fmt.Fprintf(os.Stderr, "If you log in again, the existing credentials will be overwritten.\n")
			fmt.Fprintf(os.Stderr, "Proceed [y/n]? ")
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" {
				fmt.Fprintln(os.Stderr, "Cancelled. No changes made")
				os.Exit(2)
			}
		}

		fmt.Fprintln(os.Stderr, "Logging you in using the browser")
		ctx := context.Background()
		creds, err := auth.Authorize(ctx, apiEndpoint)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := creds.Save(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprint(os.Stderr, "Logged in as ")
		fmt.Fprint(os.Stdout, creds.Name())
		fmt.Fprint(os.Stderr, "\n")
	},
}

func init() {
	cmd.AddCommand(loginCommand)
}

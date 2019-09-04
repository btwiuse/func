package main

import (
	"context"
	"fmt"
	"os"

	"github.com/func/func/auth"
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

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
		ctx := signalContext(context.Background())

		endpoint, err := cmd.Flags().GetString("endpoint")
		if err != nil {
			panic(err)
		}

		auth := &auth.Auth{
			Endpoint: endpoint,
		}

		if err := auth.Login(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	cmd.AddCommand(loginCommand)
}

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
	Short: "Log in to func service",
	Run: func(cmd *cobra.Command, args []string) {
		aud, err := cmd.Flags().GetString("audience")
		if err != nil {
			panic(err)
		}
		if aud == "" {
			fmt.Fprintf(os.Stderr, "Audience not set.\n\n%s", cmd.UsageString())
			os.Exit(2)
		}

		issuer, err := cmd.Flags().GetString("issuer")
		if err != nil {
			panic(err)
		}
		clientID, err := cmd.Flags().GetString("client-id")
		if err != nil {
			panic(err)
		}

		oidc := &auth.Client{
			Issuer:   issuer,
			ClientID: clientID,
		}

		tok, err := oidc.Authorize(context.Background(), aud, "profile", "offline_access")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		_ = tok
	},
}

func init() {
	loginCommand.Flags().String("issuer", DefaultIssuer, "OpenID Connect domain for auth token issuer")
	loginCommand.Flags().String("client-id", DefaultClientID, "Authorization client id")
	loginCommand.Flags().String("audience", DefaultEndpoint, "Func service endpoint")

	cmd.AddCommand(loginCommand)
}

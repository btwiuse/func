package main

import (
	"context"
	"fmt"
	"os"

	"github.com/func/func/auth/oidc"
	"github.com/kr/pretty"
	"github.com/spf13/cobra"
)

var (
	authEndpoint = "https://dev-func.eu.auth0.com"
	clientID     = "WiKX7zTA5lNbIPsx8HonmZS6IuldcyI6"
)

var loginCommand = &cobra.Command{
	Use:   "login",
	Short: "Login to func service",
	Run: func(cmd *cobra.Command, args []string) {
		apiEndpoint, err := cmd.Flags().GetString("endpoint")
		if err != nil {
			panic(err)
		}

		auth := oidc.NewClient(&oidc.PKCE{
			Endpoint: authEndpoint,
			ClientID: clientID,
			Audience: apiEndpoint,
			Scope: []string{
				"openid", "profile", "offline_access",
			},
		})

		ctx := context.Background()
		creds, err := auth.Authorize(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		pretty.Println(creds)
	},
}

func init() {
	cmd.AddCommand(loginCommand)
}

package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/func/func/auth"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	// DefaultOIDCEndpoint is the default value for the OIDC endpoint flag.
	// If should be set at build time with -ldflags "-X main.DefaultOIDCEndpoint=..."
	DefaultOIDCEndpoint = ""

	// DefaultClientID is the default value for the client id flag.
	// If should be set at build time with -ldflags "-X main.DefaultClientId=..."
	DefaultClientID = ""

	// DefaultCallback is the default value for the callback flag.
	DefaultCallback = "http://127.0.0.1:30428/callback"
)

var loginCommand = &cobra.Command{
	Use:   "login",
	Short: "Log in to func service",
	Long: `Log in a user using OAuth2

Endpoints can be defined by either providing an OpenID Connect url, or two
separate auth and token OAuth2 urls.

Retrieve authorize and token urls from an OpenID Connect url:
    --oidc <url>

or

Manually set authorize and token urls for OAuth2:
    --auth-url <url> --token-url <url>

If all 3 have been specified, oidc is used.
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := signalContext(context.Background())

		aud, err := cmd.Flags().GetString("audience")
		if err != nil {
			panic(err)
		}
		if aud == "" {
			fmt.Fprintf(os.Stderr, "Audience not set.\n\n%s", cmd.UsageString())
			os.Exit(2)
		}

		clientID, err := cmd.Flags().GetString("client-id")
		if err != nil {
			panic(err)
		}
		if clientID == "" {
			fmt.Fprintf(os.Stderr, "Client id not set.\n\n%s", cmd.UsageString())
			os.Exit(2)
		}

		callbackStr, err := cmd.Flags().GetString("callback")
		if err != nil {
			panic(err)
		}
		var callback *url.URL
		if callbackStr != "" {
			u, err := url.Parse(callbackStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not parse callback: %v", err)
				os.Exit(2)
			}
			callback = u
		}

		authURL, err := cmd.Flags().GetString("auth-url")
		if err != nil {
			panic(err)
		}
		tokenURL, err := cmd.Flags().GetString("token-url")
		if err != nil {
			panic(err)
		}
		endpoint := oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		}

		oidcURL, err := cmd.Flags().GetString("oidc")
		if err != nil {
			panic(err)
		}
		if oidcURL != "" {
			provider, err := oidc.NewProvider(ctx, cleanEndpoint(oidcURL))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			endpoint = provider.Endpoint()
		}

		oidc := &auth.Client{
			Endpoint: endpoint,
			ClientID: clientID,
			Callback: callback,
		}
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		scope := []string{"openid", "profile", "offline_access"}
		tok, err := oidc.Authorize(ctx, scope, oauth2.SetAuthURLParam("audience", cleanEndpoint(aud)))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println(tok)
	},
}

func cleanEndpoint(endpoint string) string {
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	return endpoint
}

func init() {
	loginCommand.Flags().String("oidc", DefaultOIDCEndpoint, "OpenID Connect endpoint")
	loginCommand.Flags().String("auth-url", "", "OAuth2 authorize url")
	loginCommand.Flags().String("token-url", "", "OAuth2 token exchange url")
	loginCommand.Flags().String("client-id", DefaultClientID, "Authorization client id")
	loginCommand.Flags().String("callback", DefaultCallback, "OAuth2 callback url")
	loginCommand.Flags().String("audience", DefaultEndpoint, "Func service endpoint")

	cmd.AddCommand(loginCommand)
}

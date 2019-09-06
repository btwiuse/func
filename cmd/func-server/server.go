package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/func/func/api"
	"github.com/func/func/api/httpapi"
	"github.com/func/func/auth"
	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
	"github.com/func/func/resource/reconciler"
	"github.com/func/func/resource/validation"
	"github.com/func/func/source/s3"
	"github.com/func/func/storage/dynamodb"
	"github.com/mattn/go-isatty"
	"github.com/segmentio/ksuid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/square/go-jose.v2/jwt"
)

var (
	// DefaultJWKSEndpoint is the default value for the jwks endpoint flag.
	DefaultJWKSEndpoint = ""

	// DefaultAudience is the default value for the audience flag.
	DefaultAudience = ""

	// DefaultIssuer is the default value for the issuer flag.
	DefaultIssuer = ""
)

var defaultAddress = "0.0.0.0:5088"

var startCommand = &cobra.Command{
	Use:   "start",
	Short: "Start func API Server",
	Run: func(cmd *cobra.Command, args []string) {
		validator := validation.New()
		validation.AddBuiltin(validator)

		reg := &resource.Registry{}
		aws.Register(reg)
		aws.AddValidators(validator)

		cfg, err := external.LoadDefaultAWSConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		bucket, err := cmd.Flags().GetString("s3-bucket")
		if err != nil {
			panic(err)
		}
		if bucket == "" {
			bucket = os.Getenv("FUNC_S3_BUCKET")
		}
		if bucket == "" {
			fmt.Fprintf(os.Stderr, "S3 bucket not set\n%s", cmd.UsageString())
			os.Exit(2)
		}
		expiry, err := cmd.Flags().GetDuration("upload-expiry")
		if err != nil {
			panic(err)
		}
		s3src := s3.New(cfg, bucket, expiry)

		table, err := cmd.Flags().GetString("dynamodb-table")
		if err != nil {
			panic(err)
		}
		if table == "" {
			table = os.Getenv("FUNC_DYNAMODB_TABLE")
		}
		if table == "" {
			fmt.Fprintf(os.Stderr, "DynamoDB table not set\n%s", cmd.UsageString())
			os.Exit(2)
		}
		dynamo := dynamodb.New(cfg, table, reg)

		var logger *zap.Logger
		if isatty.IsTerminal(os.Stdout.Fd()) {
			l, err := zap.NewDevelopment()
			if err != nil {
				panic(err)
			}
			logger = l
		} else {
			l, err := zap.NewProduction()
			if err != nil {
				panic(err)
			}
			logger = l
			defer func() {
				_ = logger.Sync()
			}()
		}

		api := &api.Server{
			Logger:    logger.Named("server"),
			Registry:  reg,
			Source:    s3src,
			Storage:   dynamo,
			Validator: validator,

			// Setting reconciler enables sync reconciliation
			Reconciler: &reconciler.Reconciler{
				Logger:    logger.Named("reconciler"),
				Resources: dynamo,
				Source:    s3src,
				Registry:  reg,
				IDGen: reconciler.IDGeneratorFunc(func() string {
					return ksuid.New().String()
				}),
			},
		}

		aud, err := cmd.Flags().GetString("audience")
		if err != nil {
			panic(err)
		}
		iss, err := cmd.Flags().GetString("issuer")
		if err != nil {
			panic(err)
		}

		server := &httpapi.Server{
			API:    api,
			Logger: logger.Named("http_api"),
			KeyProvider: &auth.KeyProviderJWKS{
				Endpoint: "https://dev-func.eu.auth0.com/.well-known/jwks.json",
			},
			ExpectedClaims: jwt.Expected{
				Audience: []string{aud},
				Issuer:   iss,
			},
		}

		addr, err := cmd.Flags().GetString("address")
		if err != nil {
			panic(err)
		}

		logger.Info("Starting server", zap.String("address", addr))

		if err := http.ListenAndServe(addr, server); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	startCommand.Flags().String("address", defaultAddress, "Server address to listen on. Env var: FUNC_ADDR")
	startCommand.Flags().String("s3-bucket", "", "S3 bucket for source code uploads. Env var: FUNC_S3_BUCKET")
	startCommand.Flags().Duration("upload-expiry", 5*time.Minute, "Time for upload url expiry")
	startCommand.Flags().String("dynamodb-table", "", "DynamoDB table for storage. Env var: FUNC_DYNAMODB_TABLE")
	startCommand.Flags().String("jwks-endpoint", DefaultJWKSEndpoint, "JWKS endpoint for signing keys")
	startCommand.Flags().String("audience", DefaultAudience, "Audience to verify in JWT token")
	startCommand.Flags().String("issuer", DefaultIssuer, "Issuer to verify in JWT token")

	cmd.AddCommand(startCommand)
}

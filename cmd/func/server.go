package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/func/func/api"
	"github.com/func/func/api/rpc"
	"github.com/func/func/graph/reconciler"
	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
	"github.com/func/func/source/s3"
	"github.com/func/func/storage"
	"github.com/func/func/storage/kvbackend"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

var serverCommand = &cobra.Command{
	Use:   "server",
	Short: "Start func API server",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		addr, err := cmd.Flags().GetString("address")
		if err != nil {
			log.Fatalf("Get server address: %v", err)
		}

		bucket, err := cmd.Flags().GetString("bucket")
		if err != nil {
			log.Fatalf("Get server address: %v", err)
		}

		if bucket == "" {
			// TODO(akupila): Move this somewhere else
			bucket = os.Getenv("FUNC_S3_BUCKET")
		}
		if bucket == "" {
			fmt.Fprintf(os.Stderr, "Required flag \"bucket\" or FUNC_S3_BUCKET env var not set\n\n%s\n", cmd.UsageString())
			os.Exit(2)
		}

		var logger *zap.Logger
		if terminal.IsTerminal(int(os.Stdout.Fd())) {
			logger, err = zap.NewDevelopment()
		} else {
			logger, err = zap.NewProduction()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer func() {
			// Ignore potential sync error
			// https://github.com/uber-go/zap/issues/370
			_ = logger.Sync()
		}()

		cfg, err := external.LoadDefaultAWSConfig()
		if err != nil {
			log.Fatalf("Could not load AWS SDK config: %v", err)
		}
		cfg.Region = "eu-central-1"

		src := &s3.Storage{
			Bucket:          bucket,
			UploadURLExpiry: 5 * time.Minute,
			Client:          awss3.New(cfg),
		}

		reg := &resource.Registry{}
		aws.Register(reg)

		bolt, err := kvbackend.NewBolt()
		if err != nil {
			log.Fatalf("Open BoltDB: %v", err)
		}
		defer func() {
			if err := bolt.Close(); err != nil {
				log.Fatalf("Close BoltDB: %v", err)
			}
		}()

		reco := &reconciler.Reconciler{
			State:  &storage.KV{Backend: bolt, ResourceCodec: reg},
			Source: src,
			Logger: logger,
		}

		funcAPI := &api.Func{
			Logger:     logger,
			Source:     src,
			Resources:  reg,
			Reconciler: reco,
		}

		server := &http.Server{
			Addr:    addr,
			Handler: rpc.NewHandler(logger, funcAPI),
		}

		go func() {
			if err := server.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					return
				}
				logger.Fatal("ListenAndServe", zap.Error(err))
			}
		}()
		logger.Info("Accepting connections", zap.String("address", addr))

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)

		s := <-sig

		logger.Debug("Shutting down", zap.String("sig", s.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatal("Error shutting down", zap.Error(err))
		}
	},
}

func init() {
	serverCommand.Flags().String("address", "0.0.0.0:5088", "Address to listen to")
	serverCommand.Flags().String("bucket", "", "S3 bucket for source code storage (FUNC_S3_BUCKET)")

	Func.AddCommand(serverCommand)
}

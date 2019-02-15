package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/func/func/api"
	"github.com/func/func/graph/reconciler"
	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
	"github.com/func/func/server"
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

		logger, done := loggerOrExit()
		defer done()

		src, err := startLocalStorage()
		if err != nil {
			log.Fatalf("Could not start local storage: %v", err)
		}

		reg := &resource.Registry{}
		reg.Register(&aws.LambdaFunction{})
		reg.Register(&aws.IAMRole{})

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
			Storage: &storage.KV{
				Backend:       bolt,
				ResourceCodec: reg,
			},
		}

		srv := &server.Server{
			Logger:     logger,
			Source:     src,
			Resources:  reg,
			Reconciler: reco,
		}
		handler := api.NewFuncServer(srv, nil)

		server := &http.Server{
			Addr:    addr,
			Handler: handler,
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

	Func.AddCommand(serverCommand)
}

// loggerOrExit creates a logger and returns it, or prints the error and
// exits with code 1 if there was an error.
//
// If a TTY is attached, the logger is started with debug mode, otherwise in
// production mode. In production mode logs are printed as json.
func loggerOrExit() (*zap.Logger, func()) {
	var log *zap.Logger
	var err error

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	done := func() {
		// Ignore potential sync error
		// https://github.com/uber-go/zap/issues/370
		_ = log.Sync()
	}

	return log, done
}

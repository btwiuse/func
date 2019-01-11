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
	"github.com/func/func/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

var serverCommand = &cobra.Command{
	Use:   "server",
	Short: "Start func API server",
}

func init() {
	addr := serverCommand.Flags().String("address", "0.0.0.0:5088", "Address to listen to")

	serverCommand.Run = func(cmd *cobra.Command, args []string) {
		runServer(*addr)
	}

	Func.AddCommand(serverCommand)
}

func runServer(addr string) {
	logger, done := loggerOrExit()
	defer done()

	srv := &server.Server{
		Logger: logger,
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

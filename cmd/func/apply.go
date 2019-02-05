package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"github.com/func/func/api"
	"github.com/func/func/client"
	"github.com/func/func/resource"
	"github.com/func/func/server"
	"github.com/func/func/source"
	"github.com/func/func/source/disk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var applyCommand = &cobra.Command{
	Use:   "apply [dir]",
	Short: "Apply resources changes",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			args = []string{"."}
		}

		addr, err := cmd.Flags().GetString("server")
		if err != nil {
			log.Fatalf("Get server address: %v", err)
		}

		var apicli api.Func
		if addr == "local" {
			// Start local server
			src, err := startLocalStorage()
			if err != nil {
				log.Fatalf("Could not start local storage: %v", err)
			}

			logCfg := zap.NewDevelopmentConfig()
			logCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
			logger, err := logCfg.Build()
			if err != nil {
				log.Fatalf("Build logger: %v", err)
			}

			apicli = &server.Server{
				Logger:    logger,
				Source:    src,
				Resources: &resource.Registry{}, // For now, this is empty
			}
		} else {
			// Start protobuf client
			apicli = api.NewFuncProtobufClient(addr, http.DefaultClient)
		}

		cli := &client.Client{
			API: apicli,
		}

		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Fatalf("Get namespace: %v", err)
		}

		rootDir, err := cli.FindRoot(args[0])
		if err != nil {
			fatal(err)
		}

		ctx := signalContext(context.Background())

		if err := cli.Apply(ctx, rootDir, ns); err != nil {
			fatal(err)
		}
	},
}

func init() {
	applyCommand.Flags().String("namespace", "default", "Namespace to use")
	applyCommand.Flags().String("server", "local", "Server endpoint. If 'local', an embedded server is used.")

	Func.AddCommand(applyCommand)
}

func startLocalStorage() (source.Storage, error) {
	u, err := user.Current()
	if err != nil {
		return nil, errors.Wrap(err, "get user home dir")
	}
	dir := filepath.Join(u.HomeDir, ".func", "source")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, errors.Wrap(err, "create upload directory")
	}

	src := &disk.Storage{Dir: dir}
	go func() {
		if err := src.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	return src, nil
}

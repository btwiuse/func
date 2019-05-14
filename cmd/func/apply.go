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
	"github.com/func/func/api/rpc"
	"github.com/func/func/client"
	"github.com/func/func/graph/reconciler"
	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
	"github.com/func/func/source"
	"github.com/func/func/source/disk"
	"github.com/func/func/storage"
	"github.com/func/func/storage/kvbackend"
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

		var funcAPI api.API
		if addr == "local" {
			// Start local server
			src, err := startLocalStorage()
			if err != nil {
				log.Fatalf("Could not start local storage: %v", err)
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
				State:  &storage.KV{Backend: bolt, Registry: reg},
				Source: src,
			}

			funcAPI = &api.Func{
				Logger:     zap.NewNop(),
				Source:     src,
				Resources:  reg,
				Reconciler: reco,
			}
		} else {
			funcAPI = rpc.NewClient(addr, nil)
		}

		cli := &client.Client{API: funcAPI}

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

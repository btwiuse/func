package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/func/func/api"
	"github.com/func/func/config"
	"github.com/func/func/graph"
	"github.com/func/func/server"
	"github.com/func/func/source"
	"github.com/func/func/source/disk"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var applyCommand = &cobra.Command{
	Use:   "apply [dir]",
	Short: "Apply resources changes",
}

func init() {
	ns := applyCommand.Flags().String("namespace", "default", "Namespace to use")
	addr := applyCommand.Flags().String("server", "local", "Server endpoint. If 'local', an embedded server is used.")

	applyCommand.Run = func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Fprintln(os.Stderr, cmd.UsageString())
			os.Exit(2)
		}
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		if err := runApply(target, *ns, *addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	Func.AddCommand(applyCommand)
}

func runApply(target, ns, addr string) error {
	l := &config.Loader{
		Compressor: source.TarGZ{},
	}

	rootDir, diags := l.Root(target)
	if diags.HasErrors() {
		l.PrintDiagnostics(os.Stderr, diags)
		os.Exit(1)
	}

	body, diags := l.Load(rootDir)
	if diags.HasErrors() {
		l.PrintDiagnostics(os.Stderr, diags)
		os.Exit(1)
	}

	var cli api.Func
	if addr == "local" {
		// Start local server
		src, err := startLocalStorage()
		if err != nil {
			return errors.Wrap(err, "startl local storage")
		}

		cli = &server.Server{
			Logger: zap.NewNop(),
			Source: src,
			// For now, this is empty
			Resources: map[string]graph.Resource{},
		}
	} else {
		// Start protobuf client
		cli = api.NewFuncProtobufClient(addr, http.DefaultClient)
	}

	j, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req := &api.ApplyRequest{Namespace: ns, Config: j}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	resp, err := cli.Apply(ctx, req)
	if err != nil {
		checkDiags(l, err)
	}

	if sr := resp.GetSourceRequest(); sr != nil {
		if err = upload(ctx, sr, l); err != nil {
			return errors.Wrap(err, "upload source")
		}
		resp, err = cli.Apply(ctx, req)
		if err != nil {
			checkDiags(l, err)
		}
	}

	fmt.Println(resp)
	return nil
}

func upload(ctx context.Context, sr *api.SourceRequired, l *config.Loader) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, ur := range sr.GetUploads() {
		ur := ur
		g.Go(func() (err error) {
			src := l.Source(ur.GetDigest())
			if err := source.Upload(
				ctx,
				http.DefaultClient,
				ur.GetUrl(),
				ur.GetHeaders(),
				src,
			); err != nil {
				return errors.Wrap(err, ur.GetDigest())
			}
			return nil
		})
	}
	return g.Wait()
}

func checkDiags(l *config.Loader, err error) {
	defer os.Exit(1)
	if twerr, ok := err.(twirp.Error); ok {
		if diagJSON := twerr.Meta("diagnostics"); diagJSON != "" {
			var diags hcl.Diagnostics
			derr := json.Unmarshal([]byte(diagJSON), &diags)
			if derr == nil {
				l.PrintDiagnostics(os.Stderr, diags)
				return
			}
			fmt.Fprintln(os.Stderr, derr)
		}
	}
	fmt.Fprintln(os.Stderr, err)
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

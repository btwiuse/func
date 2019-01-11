package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/func/func/api"
	"github.com/func/func/config"
	"github.com/func/func/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var applyCommand = &cobra.Command{
	Use:   "apply [dir]",
	Short: "Apply resources changes",
}

func init() {
	ns := applyCommand.Flags().String("namespace", "default", "Namespace to use")
	addr := applyCommand.Flags().String("server", "local", "Server endpoint address. If 'local', an embedded server is used.")

	applyCommand.Run = func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Fprintln(os.Stderr, cmd.UsageString())
			os.Exit(2)
		}
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		runApply(target, *ns, *addr)
	}

	Func.AddCommand(applyCommand)
}

func runApply(target, ns, addr string) {
	l := &config.Loader{}

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
		cli = &server.Server{
			Logger: zap.NewNop(),
		}
	} else {
		// Start protobuf client
		cli = api.NewFuncProtobufClient(addr, http.DefaultClient)
	}

	j, err := json.Marshal(body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	req := &api.ApplyRequest{
		Namespace: ns,
		Config:    j,
	}

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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(resp)
}

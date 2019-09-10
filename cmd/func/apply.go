package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/func/func/api"
	"github.com/func/func/api/httpapi"
	"github.com/func/func/auth"
	"github.com/func/func/config"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var applyCommand = &cobra.Command{
	Use:   "apply [dir]",
	Short: "Apply resources changes",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			args = []string{"."}
		}

		start := time.Now()

		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			panic(err)
		}
		logcfg := zap.Config{
			Level:    zap.NewAtomicLevelAt(zap.ErrorLevel),
			Encoding: "console",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:     "T",
				LevelKey:    "L",
				NameKey:     "N",
				MessageKey:  "M",
				LineEnding:  zapcore.DefaultLineEnding,
				EncodeLevel: zapcore.CapitalColorLevelEncoder,
				EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString(t.Format("15:04:05.999"))
				},
				EncodeDuration: zapcore.StringDurationEncoder,
			},
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
		if verbose {
			logcfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		}
		logger, err := logcfg.Build()
		if err != nil {
			panic(err)
		}

		loader := &config.Loader{
			Compressor: source.TarGZ{},
		}

		project, err := config.FindProject(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if project == nil {
			projectNewCommand.Run(cmd, args)
			// Load created project
			proj, err := config.FindProject(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			project = proj
		}

		logger.Debug("Load config files")
		cfg, diags := loader.Load(project.RootDir)
		if len(diags) > 0 {
			loader.WriteDiagnostics(os.Stderr, diags)
			if diags.HasErrors() {
				os.Exit(2)
			}
		}

		endpoint, err := cmd.Flags().GetString("endpoint")
		if err != nil {
			panic(err)
		}
		clientID, err := cmd.Flags().GetString("client-id")
		if err != nil {
			panic(err)
		}

		httpclient, err := auth.HTTPClient(clientID)
		if err != nil {
			if err == auth.ErrTokenNotFound {
				logger.Fatal("Not logged in. Log in first with func login")
			}
			logger.Fatal(err.Error())
		}

		cli := &api.Client{
			API: &httpapi.Client{
				Endpoint:   endpoint,
				HTTPClient: httpclient,
			},
			Source: loader,
			Logger: logger,
		}

		req := &api.ApplyRequest{
			Project: project.Name,
			Config:  cfg,
		}

		ctx := signalContext(context.Background())
		if err := cli.Apply(ctx, req); err != nil {
			if diags, ok := err.(hcl.Diagnostics); ok {
				loader.WriteDiagnostics(os.Stderr, diags)
				os.Exit(2)
				return
			}
			logger.Fatal(err.Error())
		}

		logger.Info(fmt.Sprintf("Done in %s", time.Since(start).Truncate(time.Millisecond)))
	},
}

func init() {
	applyCommand.Flags().Bool("verbose", false, "Verbose output")
	applyCommand.Flags().String("endpoint", DefaultEndpoint, "Func service endpoint")
	applyCommand.Flags().String("client-id", DefaultClientID, "Authorization client id")

	cmd.AddCommand(applyCommand)
}

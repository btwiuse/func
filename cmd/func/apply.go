package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/func/func/api"
	"github.com/func/func/api/httpapi"
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
		logger := zap.NewNop()
		if verbose {
			cfg := zap.Config{
				Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
				Development: true,
				Encoding:    "console",
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
			l, err := cfg.Build()
			if err != nil {
				panic(err)
			}
			logger = l
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

		addr, err := cmd.Flags().GetString("server")
		if err != nil {
			panic(err)
		}

		cli := &api.Client{
			API:    &httpapi.Client{Endpoint: addr},
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
	applyCommand.Flags().String("server", "https://api.func.io", "Server endpoint")

	Func.AddCommand(applyCommand)
}

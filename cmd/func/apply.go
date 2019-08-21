package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/func/func/api"
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

		rootDir, err := loader.Root(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if rootDir == "" {
			fmt.Fprintln(os.Stderr, "Project not found")
			os.Exit(2)
		}

		logger.Debug("Load config files")
		cfg, diags := loader.Load(rootDir)
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

		cli := api.NewClient(addr, logger, loader)
		ctx := signalContext(context.Background())

		project := filepath.Base(rootDir)

		if err := cli.Apply(ctx, project, cfg); err != nil {
			if diags, ok := err.(hcl.Diagnostics); ok {
				loader.WriteDiagnostics(os.Stderr, diags)
				os.Exit(2)
				return
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		logger.Info(fmt.Sprintf("Done in %s", time.Since(start).Truncate(time.Millisecond)))
	},
}

func init() {
	applyCommand.Flags().Bool("verbose", false, "Verbose output")
	applyCommand.Flags().String("server", "http://"+defaultAddress, "Server endpoint") // TODO: https

	Func.AddCommand(applyCommand)
}

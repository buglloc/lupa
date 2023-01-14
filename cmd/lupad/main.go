package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/buglloc/lupa/internal/config"
)

var (
	configs []string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:           "lupad",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cobra.OnInitialize(
		func() {
			var err error
			cfg, err = config.LoadConfig(configs...)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "unable to load config: %v\n", err)
				os.Exit(1)
			}
		},
		func() {
			if cfg.Debug {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}
		},
	)

	flags := rootCmd.PersistentFlags()
	flags.StringSliceVar(&configs, "config", nil, "config file")

	rootCmd.AddCommand(
		startCmd,
	)
}

func main() {
	_, _ = maxprocs.Set(maxprocs.Logger(log.Info().Msgf))

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

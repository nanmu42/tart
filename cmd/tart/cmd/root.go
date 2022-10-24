package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"tart/config"

	"github.com/pelletier/go-toml/v2"

	"github.com/spf13/cobra"
)

var (
	cfgPath string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "tart.toml", "Path to the config file")
	_ = rootCmd.MarkPersistentFlagFilename("config")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "tart",
	Short:        "An educational purpose, unofficial Gitlab Runner.",
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func loadConfig() (cfg config.Config, err error) {
	file, err := os.Open(cfgPath)
	if err != nil {
		err = fmt.Errorf("opening config file: %w", err)
		return
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		err = fmt.Errorf("decoding config from TOML: %w", err)
		return
	}

	return
}

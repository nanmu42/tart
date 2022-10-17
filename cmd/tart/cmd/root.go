package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgPath string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "f", "./tart.toml", "Path to the config file")
	_ = rootCmd.MarkPersistentFlagFilename("f")
	_ = rootCmd.MarkPersistentFlagFilename("config")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tart",
	Short: "An educational purpose, unofficial Gitlab Runner.",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

package cmd

import (
	"fmt"
	"tart/version"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		fmt.Println(version.FullName)

		return
	},
}

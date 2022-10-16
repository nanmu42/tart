package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Listen and run CI jobs",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run called")
	},
}

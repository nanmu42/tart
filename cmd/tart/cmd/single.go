package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(singleCmd)
}

var singleCmd = &cobra.Command{
	Use:   "single",
	Short: "Listen, wait and run a single CI job, then exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("single called")
	},
}

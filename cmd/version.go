package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	ver    = "dev"
	commit = "none"
	date   = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("jigo %s (%s) built %s\n", ver, commit, date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

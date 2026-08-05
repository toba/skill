package cmd

import (
	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/nope"
)

var nopeCmd = &cobra.Command{
	Use:   "nope",
	Short: "Claude Code PreToolUse guard",
	Long:  "Guard against dangerous tool invocations. Reads JSON from stdin and exits 0 (allow) or 2 (block).",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nope.RunGuard(ver)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(nopeCmd)
}

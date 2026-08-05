package cmd

import (
	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/nope"
)

var nopeInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold nope rules in .jig.yaml and hook in .claude/settings.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		code := nope.RunInit()
		if code != 0 {
			return nope.ExitError{Code: code}
		}
		return nil
	},
}

func init() {
	nopeCmd.AddCommand(nopeInitCmd)
}

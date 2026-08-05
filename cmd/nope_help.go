package cmd

import (
	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/nope"
)

var nopeHelpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show nope guard reference",
	RunE: func(cmd *cobra.Command, args []string) error {
		nope.RunHelp()
		return nil
	},
}

func init() {
	nopeCmd.AddCommand(nopeHelpCmd)
}

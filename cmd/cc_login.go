package cmd

import (
	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cc"
	"github.com/toba/jig-go/v4/internal/nope"
)

var ccLoginCmd = &cobra.Command{
	Use:   "login <alias>",
	Short: "Run `<cli> /login` for the given alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cc.Load()
		if err != nil {
			return err
		}
		name, _, err := c.Resolve(args[0])
		if err != nil {
			return err
		}
		if err := cc.SeedClaudeJSON(c, name); err != nil {
			return err
		}
		code, err := cc.Launch(c, name, []string{"/login"})
		if err != nil {
			return err
		}
		if code != 0 {
			return nope.ExitError{Code: code}
		}
		return nil
	},
}

func init() {
	ccCmd.AddCommand(ccLoginCmd)
}

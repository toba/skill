package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cc"
)

var ccRemoveKeepDir bool

var ccRemoveCmd = &cobra.Command{
	Use:   "remove <alias>",
	Short: "Remove an alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cc.Load()
		if err != nil {
			return err
		}
		name, a, err := c.Resolve(args[0])
		if err != nil {
			return err
		}
		if a.IsSource {
			return fmt.Errorf("refusing to remove source alias %q", name)
		}
		delete(c.Aliases, name)
		if err := c.Save(); err != nil {
			return err
		}
		if !ccRemoveKeepDir {
			if err := cc.RemoveAliasDir(&cc.Config{
				Aliases: map[string]cc.Alias{name: a},
			}, name); err != nil {
				return err
			}
		}
		if jsonOut {
			return cc.EmitJSON(cc.JSONResponse{Success: true, Message: fmt.Sprintf("removed %q", name)})
		}
		fmt.Printf("Removed alias %q\n", name)
		return nil
	},
}

func init() {
	ccRemoveCmd.Flags().BoolVar(&ccRemoveKeepDir, "keep-dir", false, "do not delete the alias directory")
	ccCmd.AddCommand(ccRemoveCmd)
}

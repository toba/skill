package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cc"
)

var ccCloneCmd = &cobra.Command{
	Use:   "clone <src> <alias>",
	Short: "Clone an alias including private files",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cc.Load()
		if err != nil {
			return err
		}
		srcName, srcAlias, err := c.Resolve(args[0])
		if err != nil {
			return err
		}
		newName := args[1]
		if _, exists := c.Aliases[newName]; exists {
			return fmt.Errorf("alias %q already exists", newName)
		}
		path, err := cc.AliasDir(newName)
		if err != nil {
			return err
		}
		c.Aliases[newName] = cc.Alias{
			CLI:  srcAlias.CLI,
			Path: path,
		}
		if err := c.Save(); err != nil {
			return err
		}
		if err := cc.CopyPrivateFiles(srcAlias.Path, path, c.PrivateList()); err != nil {
			return err
		}
		rep, err := cc.Sync(c, newName)
		if err != nil {
			return err
		}
		if jsonOut {
			return cc.EmitJSON(cc.JSONResponse{
				Success: true,
				Message: fmt.Sprintf("cloned %q from %q", newName, srcName),
				Data:    rep,
			})
		}
		fmt.Printf("Cloned %q from %q at %s\n", newName, srcName, path)
		return nil
	},
}

func init() {
	ccCmd.AddCommand(ccCloneCmd)
}

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cc"
)

var ccAddCLI string

var ccAddCmd = &cobra.Command{
	Use:   "add <alias>",
	Short: "Create a new alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c, err := cc.Load()
		if err != nil {
			return err
		}
		if _, exists := c.Aliases[name]; exists {
			return fmt.Errorf("alias %q already exists", name)
		}
		path, err := cc.AliasDir(name)
		if err != nil {
			return err
		}
		c.Aliases[name] = cc.Alias{
			CLI:  ccAddCLI,
			Path: path,
		}
		if err := c.Save(); err != nil {
			return err
		}
		if err := cc.SeedClaudeJSON(c, name); err != nil {
			return err
		}
		rep, err := cc.Sync(c, name)
		if err != nil {
			return err
		}
		if jsonOut {
			return cc.EmitJSON(cc.JSONResponse{
				Success: true,
				Message: fmt.Sprintf("alias %q created", name),
				Data:    rep,
			})
		}
		fmt.Printf("Created alias %q at %s\n", name, path)
		fmt.Printf("  symlinks: %d created, %d skipped\n", len(rep.Created), len(rep.Skipped))
		if len(rep.Conflicts) > 0 {
			return errors.New("conflicts: real files in shared positions; resolve with `jigo cc rebuild`")
		}
		return nil
	},
}

func init() {
	ccAddCmd.Flags().StringVar(&ccAddCLI, "cli", "claude", "CLI binary to launch")
	ccCmd.AddCommand(ccAddCmd)
}

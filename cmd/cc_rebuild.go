package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cc"
)

var ccRebuildCmd = &cobra.Command{
	Use:   "rebuild [alias]",
	Short: "Recreate symlinks for one or all aliases",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cc.Load()
		if err != nil {
			return err
		}
		var targets []string
		if len(args) == 1 {
			name, _, err := c.Resolve(args[0])
			if err != nil {
				return err
			}
			targets = []string{name}
		} else {
			for _, n := range c.Names() {
				if !c.Aliases[n].IsSource {
					targets = append(targets, n)
				}
			}
		}
		reports := map[string]*cc.SyncReport{}
		for _, n := range targets {
			rep, err := cc.Sync(c, n)
			if err != nil {
				return err
			}
			reports[n] = rep
		}
		if jsonOut {
			return cc.EmitJSON(cc.JSONResponse{Success: true, Data: reports})
		}
		for n, rep := range reports {
			fmt.Printf("%s: %d created, %d skipped, %d repaired, %d conflicts\n",
				n, len(rep.Created), len(rep.Skipped), len(rep.Repaired), len(rep.Conflicts))
			for _, c := range rep.Conflicts {
				fmt.Printf("    conflict: %s\n", c)
			}
		}
		return nil
	},
}

func init() {
	ccCmd.AddCommand(ccRebuildCmd)
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cc"
)

var (
	ccInitSource string
	ccInitForce  bool
)

var ccInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Detect ~/.claude* dirs and create the cc config",
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := cc.Init(cc.InitOpts{
			Source: ccInitSource,
			Force:  ccInitForce,
		})
		if err != nil {
			if jsonOut {
				_ = cc.EmitJSON(cc.JSONResponse{Success: false, Error: err.Error()})
			}
			return err
		}
		if jsonOut {
			return cc.EmitJSON(cc.JSONResponse{
				Success: true,
				Message: "cc initialized",
				Data:    res,
			})
		}
		fmt.Printf("Wrote %s\n", res.ConfigPath)
		fmt.Printf("  source:  %s (alias %q)\n", res.SharedSource, res.SourceAlias)
		fmt.Printf("  aliases: %v\n", res.Aliases)
		for n, rep := range res.Synced {
			fmt.Printf("  synced %s: %d created, %d skipped, %d repaired, %d conflicts\n",
				n, len(rep.Created), len(rep.Skipped), len(rep.Repaired), len(rep.Conflicts))
		}
		return nil
	},
}

func init() {
	ccInitCmd.Flags().StringVar(&ccInitSource, "source", "", "explicit source ~/.claude path (default: auto-detect)")
	ccInitCmd.Flags().BoolVar(&ccInitForce, "force", false, "overwrite existing config")
	ccCmd.AddCommand(ccInitCmd)
}

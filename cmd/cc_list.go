package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cc"
)

var ccListCmd = &cobra.Command{
	Use:   "list",
	Short: "List aliases",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cc.Load()
		if err != nil {
			return err
		}
		if jsonOut {
			type entry struct {
				Name     string `json:"name"`
				CLI      string `json:"cli"`
				Path     string `json:"path"`
				IsSource bool   `json:"is_source"`
			}
			out := make([]entry, 0, len(c.Aliases))
			for _, n := range c.Names() {
				a := c.Aliases[n]
				out = append(out, entry{Name: n, CLI: a.CLI, Path: a.Path, IsSource: a.IsSource})
			}
			return cc.EmitJSON(cc.JSONResponse{Success: true, Data: out})
		}
		for _, n := range c.Names() {
			a := c.Aliases[n]
			marker := " "
			if a.IsSource {
				marker = "*"
			}
			fmt.Printf("%s %-12s  %s  %s\n", marker, n, a.CLI, a.Path)
		}
		return nil
	},
}

func init() {
	ccCmd.AddCommand(ccListCmd)
}

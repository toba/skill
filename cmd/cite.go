package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/config"
)

var citeCmd = &cobra.Command{
	Use:   "cite",
	Short: "Monitor cited repositories for changes",
	Long:  "Commands for checking cited repos, classifying changes by relevance, and tracking what you've reviewed.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// init doesn't need config loaded.
		if cmd.Name() == "init" || cmd.Name() == "add" || cmd.Name() == "doctor" || cmd.Name() == "update" {
			return nil
		}
		path := configPath()
		doc, c, err := config.Load(path)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		cfg = c
		cfgDoc = doc
		return nil
	},
}

func init() {
	rootCmd.AddCommand(citeCmd)
}

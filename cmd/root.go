package cmd

import (
	"cmp"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/toba/jig/internal/config"
	"github.com/toba/jig/internal/constants"
	"github.com/toba/jig/internal/nope"
)

var (
	cfgPath string
	jsonOut bool
	cfg     *config.Config
	cfgDoc  *config.Document
)

var rootCmd = &cobra.Command{
	Use:   "jigo",
	Short: "Multi-tool CLI for citation monitoring and Claude Code security guard",
	Long:  "Multi-tool CLI combining citation monitoring, companion management, and Claude Code security guard.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	// Bare `jigo` (no subcommand) prints help. Args are constrained to none so
	// unknown subcommands still error instead of silently printing help.
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to config file (default .jig.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output as JSON")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if exitErr, ok := errors.AsType[nope.ExitError](err); ok {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}

func configPath() string {
	return cmp.Or(cfgPath, constants.ConfigFileName)
}

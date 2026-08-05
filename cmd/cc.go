package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/cc"
	"github.com/toba/jig-go/v4/internal/nope"
)

var ccCmd = &cobra.Command{
	Use:   "cc [alias] [-- claude flags...]",
	Short: "Manage and launch multiple Claude Code profiles",
	Long: `Manage multiple Claude Code "aliases" (profiles), each with shared
agents/skills/commands symlinked from a single source and isolated
credentials per alias.

  jigo cc init                     auto-detect ~/.claude* dirs
  jigo cc add <alias>              create a new alias
  jigo cc list                     list aliases
  jigo cc <alias> [claude flags]   launch claude with that alias
  jigo cc                          interactive picker`,
	DisableFlagParsing: true,
	SilenceUsage:       true,
	SilenceErrors:      true,
	RunE:               ccDispatch,
}

// ccDispatch handles the parent's RunE. It is called only when no subcommand
// matched. With DisableFlagParsing the args contain the full original tail
// after `jigo cc`.
func ccDispatch(cmd *cobra.Command, args []string) error {
	// Help & no args.
	if len(args) == 0 {
		return ccLaunchInteractive()
	}

	first := args[0]

	// Built-in cobra help flags fall through to subcommand handler when no
	// subcommand exists. Handle them here.
	if first == "-h" || first == "--help" {
		return cmd.Help()
	}

	// If first arg matches a known subcommand, defer to it directly. Cobra's
	// own dispatch is bypassed because DisableFlagParsing is true on the
	// parent.
	for _, sc := range cmd.Commands() {
		if sc.Name() == first {
			sc.SetArgs(args[1:])
			return sc.Execute()
		}
	}

	// Otherwise treat as alias name.
	return ccLaunch(first, args[1:])
}

func ccLaunchInteractive() error {
	c, err := cc.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("no cc config found; run `jigo cc init` first")
		}
		return err
	}
	cwd, _ := os.Getwd()
	preselect := cc.LastAlias(cwd)
	name, err := cc.PickAlias(c, preselect)
	if err != nil {
		return err
	}
	if name == "" {
		return nil
	}
	return ccLaunch(name, nil)
}

func ccLaunch(query string, extraArgs []string) error {
	c, err := cc.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("no cc config found; run `jigo cc init` first")
		}
		return err
	}
	code, err := cc.Launch(c, query, extraArgs)
	if err != nil {
		return err
	}
	if code != 0 {
		return nope.ExitError{Code: code}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(ccCmd)
}

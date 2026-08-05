package cmd

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

var skipStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Faint(true) // yellow/dim

var jigInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new jigo project",
	Long: `Creates .jig.yaml and runs all init subcommands:
  nope  — writes nope rules and .claude/settings.json hook
  cite  — adds starter citations section
  brew  — creates companion tap repo (skipped if not configured)
  zed   — creates companion extension repo (skipped if not configured)

Safe to run multiple times; each init is idempotent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		type step struct {
			name   string
			cmd    *cobra.Command
			remote bool // remote inits always skip on error
		}
		steps := []step{
			{"nope", nopeInitCmd, false},
			{"cite", initCmd, false},
			{"brew", brewInitCmd, true},
			{"zed", zedInitCmd, true},
		}

		var failed int
		for _, s := range steps {
			fmt.Printf("%s ... ", s.name)
			err := s.cmd.RunE(s.cmd, nil)
			if err == nil {
				fmt.Println(passStyle.Render("ok"))
				continue
			}

			msg := err.Error()

			// Treat "already exists/contains" as success (idempotent).
			if strings.Contains(msg, "already") {
				fmt.Println(passStyle.Render("ok"))
				continue
			}

			// Remote inits (brew, zed) skip gracefully on any error.
			if s.remote {
				fmt.Println(skipStyle.Render("skip") + " " + skipStyle.Render(msg))
				continue
			}

			fmt.Println(failStyle.Render("FAIL"))
			fmt.Printf("  %s\n", msg)
			failed++
		}

		if failed > 0 {
			fmt.Printf("\n%s\n", failStyle.Render(fmt.Sprintf("%d of %d inits failed", failed, len(steps))))
			return fmt.Errorf("%d init(s) failed", failed)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(jigInitCmd)
}

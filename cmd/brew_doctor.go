package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/brew"
	"github.com/toba/jig-go/v4/internal/nope"
)

var brewDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify brew tap setup is healthy",
	RunE: func(cmd *cobra.Command, args []string) error {
		tap, err := resolveTap("", configPath())
		if err != nil {
			fmt.Fprintf(os.Stderr, "OK:   brew not in packages (nothing to check)\n")
			return nil
		}

		// Detect source repo.
		out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").Output()
		if err != nil {
			return fmt.Errorf("detecting source repo: %w", err)
		}
		repo := strings.TrimSpace(string(out))

		// Derive tool name from source repo (last path component).
		repoParts := strings.SplitN(repo, "/", 2)
		if len(repoParts) != 2 {
			return fmt.Errorf("unexpected repo format: %s", repo)
		}
		tool := repoParts[1]

		code := brew.RunDoctor(brew.DoctorOpts{
			Tap:  tap,
			Repo: repo,
			Tool: tool,
		})
		if code != 0 {
			return nope.ExitError{Code: code}
		}
		return nil
	},
}

func init() {
	brewCmd.AddCommand(brewDoctorCmd)
}

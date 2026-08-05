package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/nope"
	"github.com/toba/jig-go/v4/internal/zed"
)

var zedDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify Zed extension setup is healthy",
	RunE: func(cmd *cobra.Command, args []string) error {
		ext := resolveExt("", configPath())
		if ext == "" {
			fmt.Fprintf(os.Stderr, "OK:   zed_extension not configured (nothing to check)\n")
			return nil
		}

		// Detect source repo.
		out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").Output()
		if err != nil {
			return fmt.Errorf("detecting source repo: %w", err)
		}
		repo := strings.TrimSpace(string(out))

		code := zed.RunDoctor(zed.DoctorOpts{
			Ext:  ext,
			Repo: repo,
		})
		if code != 0 {
			return nope.ExitError{Code: code}
		}
		return nil
	},
}

func init() {
	zedCmd.AddCommand(zedDoctorCmd)
}

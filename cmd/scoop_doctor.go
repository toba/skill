package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/nope"
	"github.com/toba/jig-go/v4/internal/scoop"
)

var scoopDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify scoop bucket setup is healthy",
	RunE: func(cmd *cobra.Command, args []string) error {
		bucket, err := resolveBucket("", configPath())
		if err != nil {
			fmt.Fprintf(os.Stderr, "OK:   scoop not in packages (nothing to check)\n")
			return nil
		}

		// Detect source repo.
		out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").Output()
		if err != nil {
			return fmt.Errorf("detecting source repo: %w", err)
		}
		repo := strings.TrimSpace(string(out))

		// Derive tool name from source repo.
		repoParts := strings.SplitN(repo, "/", 2)
		if len(repoParts) != 2 {
			return fmt.Errorf("unexpected repo format: %s", repo)
		}
		tool := repoParts[1]

		code := scoop.RunDoctor(scoop.DoctorOpts{
			Bucket: bucket,
			Repo:   repo,
			Tool:   tool,
		})
		if code != 0 {
			return nope.ExitError{Code: code}
		}
		return nil
	},
}

func init() {
	scoopCmd.AddCommand(scoopDoctorCmd)
}

package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/config"
)

var brewCmd = &cobra.Command{
	Use:   "brew",
	Short: "Homebrew tap management",
	Long:  "Commands for managing Homebrew tap formulas.",
}

func init() {
	rootCmd.AddCommand(brewCmd)
}

// resolveTap determines the tap repo using (in order):
//  1. explicit --tap flag
//  2. packages list contains "brew" → derive owner/homebrew-tap
//  3. convention: owner/homebrew-tap derived from the current GitHub repo
func resolveTap(flag, cfgPath string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if tap := tapFromPackages(cfgPath); tap != "" {
		return tap, nil
	}
	if tap := tapFromConvention(); tap != "" {
		return tap, nil
	}
	return "", fmt.Errorf("--tap is required (or add brew to packages in %s)", cfgPath)
}

// tapFromPackages checks if "brew" is in the packages list and derives
// the tap repo from the current GitHub repo's org.
func tapFromPackages(cfgPath string) string {
	doc, err := config.LoadDocument(cfgPath)
	if err != nil {
		return ""
	}
	if !config.HasPackage(doc, "brew") {
		return ""
	}
	return tapFromConvention()
}

// tapFromConvention derives "owner/homebrew-tap" from the current GitHub repo.
func tapFromConvention() string {
	out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").Output()
	if err != nil {
		return ""
	}
	repo := strings.TrimSpace(string(out))
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[0] + "/homebrew-tap"
}

// repoFromGitURL extracts "owner/repo" from a git URL or short-form repo name.
// Handles https://github.com/owner/repo.git, git@github.com:owner/repo.git,
// and bare owner/repo.
func repoFromGitURL(u string) string {
	// Strip trailing .git
	u = strings.TrimSuffix(u, ".git")

	// HTTPS: https://github.com/owner/repo
	if strings.Contains(u, "://") {
		parts := strings.Split(u, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
		return ""
	}

	// SSH: git@github.com:owner/repo
	if _, after, ok := strings.Cut(u, ":"); ok {
		return after
	}

	// Bare owner/repo
	if strings.Count(u, "/") == 1 && u[0] != '/' {
		return u
	}

	return ""
}

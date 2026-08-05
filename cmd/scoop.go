package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/config"
)

var scoopCmd = &cobra.Command{
	Use:   "scoop",
	Short: "Scoop bucket management",
	Long:  "Commands for managing Scoop bucket manifests.",
}

func init() {
	rootCmd.AddCommand(scoopCmd)
}

// resolveBucket determines the bucket repo using (in order):
//  1. explicit --bucket flag
//  2. packages list contains "scoop" → derive owner/scoop-bucket
//  3. convention: owner/scoop-bucket derived from the current GitHub repo
func resolveBucket(flag, cfgPath string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if bucket := bucketFromPackages(cfgPath); bucket != "" {
		return bucket, nil
	}
	if bucket := bucketFromConvention(); bucket != "" {
		return bucket, nil
	}
	return "", fmt.Errorf("--bucket is required (or add scoop to packages in %s)", cfgPath)
}

// bucketFromPackages checks if "scoop" is in the packages list and derives
// the bucket repo from the current GitHub repo's org.
func bucketFromPackages(cfgPath string) string {
	doc, err := config.LoadDocument(cfgPath)
	if err != nil {
		return ""
	}
	if !config.HasPackage(doc, "scoop") {
		return ""
	}
	return bucketFromConvention()
}

// bucketFromConvention derives "owner/scoop-bucket" from the current GitHub repo.
func bucketFromConvention() string {
	out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").Output()
	if err != nil {
		return ""
	}
	repo := strings.TrimSpace(string(out))
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[0] + "/scoop-bucket"
}

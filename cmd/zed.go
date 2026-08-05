package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/config"
)

var zedCmd = &cobra.Command{
	Use:   "zed",
	Short: "Zed extension management",
	Long:  "Commands for managing Zed editor extension companion repos.",
}

func init() {
	rootCmd.AddCommand(zedCmd)
}

// extFromConfig reads the zed_extension value from .jig.yaml
// and extracts the "owner/repo" identifier. Returns "" on any failure.
func extFromConfig(cfgPath string) string {
	doc, err := config.LoadDocument(cfgPath)
	if err != nil {
		return ""
	}
	val := config.LoadZedExtension(doc)
	if val == "" {
		return ""
	}
	return repoFromGitURL(val)
}

// resolveExt determines the extension repo using (in order):
//  1. explicit --ext flag
//  2. zed_extension from .jig.yaml
func resolveExt(flag, cfgPath string) string {
	if flag != "" {
		return flag
	}
	if ext := extFromConfig(cfgPath); ext != "" {
		return ext
	}
	return ""
}

// extNameFromRepo extracts just the repo name from "owner/repo".
func extNameFromRepo(repo string) string {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return repo
	}
	return parts[1]
}

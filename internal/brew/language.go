package brew

import (
	"fmt"
	"os"

	"github.com/toba/jig-go/v4/internal/constants"
)

// Language describes the project language and its release conventions.
type Language struct {
	Name string // "go", "swift", "rust"
}

// DetectLanguage inspects the working directory for marker files and returns
// the detected project language. Falls back to Go if nothing matches.
func DetectLanguage() Language {
	// .goreleaser.yaml/.yml or go.mod → Go
	for _, f := range []string{constants.GoreleaserYAML, constants.GoreleaserYML, "go.mod"} {
		if _, err := os.Stat(f); err == nil {
			return Language{Name: "go"}
		}
	}
	if _, err := os.Stat("Package.swift"); err == nil {
		return Language{Name: "swift"}
	}
	if _, err := os.Stat("Cargo.toml"); err == nil {
		return Language{Name: "rust"}
	}
	return Language{Name: "go"}
}

// AssetName returns the expected darwin arm64 asset filename for a release.
func (l Language) AssetName(tool, tag string) string {
	switch l.Name {
	case "swift":
		return fmt.Sprintf("%s-%s-arm64.tar.gz", tool, tag)
	case "rust":
		return fmt.Sprintf("%s-%s-aarch64-apple-darwin.tar.gz", tool, tag)
	default: // go
		return tool + "_darwin_arm64.tar.gz"
	}
}

// AssetNameTemplate returns the expected asset filename pattern with
// ${{ github.ref_name }} in place of the tag, for matching in workflows.
func (l Language) AssetNameTemplate(tool string) string {
	switch l.Name {
	case "swift":
		return tool + "-${{ github.ref_name }}-arm64.tar.gz"
	case "rust":
		return tool + "-${{ github.ref_name }}-aarch64-apple-darwin.tar.gz"
	default: // go — goreleaser handles naming, no tag in asset name
		return tool + "_darwin_arm64.tar.gz"
	}
}

// ChecksumMode returns "checksums.txt" for Go (goreleaser convention) or
// "sidecar" for languages that produce per-asset .sha256 files.
func (l Language) ChecksumMode() string {
	if l.Name == "go" {
		return "checksums.txt"
	}
	return "sidecar"
}

// WorkflowBuildMarkers returns strings to look for in the release workflow
// to confirm it builds the project.
func (l Language) WorkflowBuildMarkers() []string {
	switch l.Name {
	case "swift":
		return []string{"swift build"}
	case "rust":
		return []string{"cargo build", "cross"}
	default: // go
		return []string{"goreleaser/goreleaser-action", "go build", "GOOS="}
	}
}

// WorkflowBuildLabel returns a human-readable label for diagnostics.
func (l Language) WorkflowBuildLabel() string {
	switch l.Name {
	case "swift":
		return "swift build step"
	case "rust":
		return "cargo/cross build step"
	default:
		return "Go build step (goreleaser-action, go build, or GOOS=)"
	}
}

// HasBuildToolCheck returns true if the language has a local build tool config
// file to validate (currently only Go with .goreleaser.yaml).
func (l Language) HasBuildToolCheck() bool {
	return l.Name == "go"
}

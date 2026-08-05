package scoop

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateManifest(t *testing.T) {
	m := GenerateManifest(ManifestParams{
		Tool:        "jig",
		Desc:        "Multi-tool CLI",
		Homepage:    "https://github.com/toba/jig-go",
		License:     "Apache-2.0",
		Tag:         "v1.0.0",
		Repo:        "toba/jig",
		SHA256AMD64: "abc123",
		SHA256ARM64: "def456",
	})

	// Valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(m), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	checks := []string{
		`"version": "1.0.0"`,
		`"description": "Multi-tool CLI"`,
		`"homepage": "https://github.com/toba/jig-go"`,
		`"license": "Apache-2.0"`,
		`"jig.exe"`,
		"jig_windows_amd64.zip",
		"jig_windows_arm64.zip",
		`"hash": "abc123"`,
		`"hash": "def456"`,
	}
	for _, want := range checks {
		if !strings.Contains(m, want) {
			t.Errorf("manifest missing %q", want)
		}
	}
}

func TestGenerateManifestNoARM64(t *testing.T) {
	m := GenerateManifest(ManifestParams{
		Tool:        "jig",
		Desc:        "CLI",
		Homepage:    "https://example.com",
		License:     "MIT",
		Tag:         "v2.0.0",
		Repo:        "org/tool",
		SHA256AMD64: "abc",
	})

	if strings.Contains(m, "arm64") {
		t.Error("manifest should not include arm64 when SHA256ARM64 is empty")
	}
}

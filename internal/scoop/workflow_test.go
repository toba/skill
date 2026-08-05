package scoop

import (
	"strings"
	"testing"

	"github.com/toba/jig-go/v4/internal/companion"
)

func TestGenerateWorkflowJob(t *testing.T) {
	job := GenerateWorkflowJob(WorkflowParams{
		Tool:    "jig",
		Org:     "toba",
		Bucket:  "toba/scoop-bucket",
		Desc:    "Multi-tool CLI",
		License: "Apache-2.0",
	})

	checks := []string{
		"update-scoop:",
		"needs: release",
		"HOMEBREW_TAP_TOKEN",
		"jig_windows_amd64.zip",
		"jig_windows_arm64.zip",
		"toba/scoop-bucket.git",
		"jig.json",
		`"jig.exe"`,
		"bump jig to ${VERSION}",
	}
	for _, want := range checks {
		if !strings.Contains(job, want) {
			t.Errorf("job missing %q", want)
		}
	}
	if strings.Contains(job, "scoop-jig") {
		t.Error("job should not reference scoop-jig (shared bucket)")
	}
	// Manifests should be at repo root, not bucket/ subdir.
	if strings.Contains(job, "bucket/jig.json") {
		t.Error("manifest should be at repo root, not bucket/ subdir")
	}
}

func TestGenerateWorkflowJobCustomNeeds(t *testing.T) {
	job := GenerateWorkflowJob(WorkflowParams{
		Tool:   "tool",
		Org:    "org",
		Bucket: "org/scoop-bucket",
		Needs:  "checksums",
	})
	if !strings.Contains(job, "needs: checksums") {
		t.Error("expected needs: checksums")
	}
}

func TestInjectWorkflowJob(t *testing.T) {
	existing := `name: Release

on:
  push:
    tags:
      - "v*"

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`

	p := WorkflowParams{
		Tool:    "jig",
		Org:     "toba",
		Bucket:  "toba/scoop-bucket",
		Desc:    "Multi-tool CLI",
		License: "Apache-2.0",
	}

	result, err := InjectWorkflowJob(existing, p)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "update-scoop:") {
		t.Error("result missing update-scoop job")
	}
	if !strings.Contains(result, "needs: release") {
		t.Error("result should depend on release job")
	}
}

func TestInjectWorkflowJobAlreadyExists(t *testing.T) {
	existing := `jobs:
  release:
    runs-on: ubuntu-latest
  update-scoop:
    needs: release
`
	_, err := InjectWorkflowJob(existing, WorkflowParams{Tool: "jig", Org: "toba", Bucket: "toba/scoop-bucket"})
	if err == nil {
		t.Error("expected error for existing update-scoop job")
	}
}

func TestDetectLastJob(t *testing.T) {
	content := `jobs:
  build:
    runs-on: ubuntu-latest
  checksums:
    needs: build
  release:
    needs: checksums
`
	got := companion.DetectLastJob(content)
	if got != "release" {
		t.Errorf("DetectLastJob = %q, want release", got)
	}
}

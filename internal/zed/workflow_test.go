package zed

import (
	"strings"
	"testing"

	"github.com/toba/jig-go/v4/internal/companion"
)

func TestGenerateSyncExtensionJob(t *testing.T) {
	job := GenerateSyncExtensionJob(WorkflowParams{
		Org: "toba",
		Ext: "gozer",
	})

	checks := []string{
		"sync-extension:",
		"needs: release",
		"toba/gozer/dispatches",
		"event_type=bump-version",
		"EXTENSION_PAT",
		"github.ref_name",
	}
	for _, want := range checks {
		if !strings.Contains(job, want) {
			t.Errorf("job missing %q", want)
		}
	}
}

func TestGenerateSyncExtensionJobCustomNeeds(t *testing.T) {
	job := GenerateSyncExtensionJob(WorkflowParams{
		Org:   "toba",
		Ext:   "gozer",
		Needs: "checksums",
	})
	if !strings.Contains(job, "needs: checksums") {
		t.Error("expected needs: checksums")
	}
}

func TestInjectSyncExtensionJob(t *testing.T) {
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
		Org: "toba",
		Ext: "gozer",
	}

	result, err := InjectSyncExtensionJob(existing, p)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "sync-extension:") {
		t.Error("result missing sync-extension job")
	}
	if !strings.Contains(result, "needs: release") {
		t.Error("result should depend on release job")
	}
}

func TestInjectSyncExtensionJobAlreadyExists(t *testing.T) {
	existing := `jobs:
  release:
    runs-on: ubuntu-latest
  sync-extension:
    needs: release
`
	_, err := InjectSyncExtensionJob(existing, WorkflowParams{Org: "toba", Ext: "gozer"})
	if err == nil {
		t.Error("expected error for existing sync-extension job")
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
		t.Errorf("detectLastJob = %q, want release", got)
	}
}

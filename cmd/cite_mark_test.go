package cmd

import (
	"testing"

	"github.com/toba/jig-go/v4/internal/config"
	"github.com/toba/jig-go/v4/internal/github"
)

// fakeMarkClient implements just enough of github.Client to drive markSource
// without shelling out to gh.
type fakeMarkClient struct {
	github.Client
	release *github.Release
	headSHA string
}

func (f *fakeMarkClient) GetLatestRelease(string) (*github.Release, error) {
	return f.release, nil
}

func (f *fakeMarkClient) GetHeadSHA(string, string) (string, error) {
	return f.headSHA, nil
}

// TestMarkSource_Commits verifies that a commit-tracked source is marked to the
// branch HEAD SHA with an empty tag.
func TestMarkSource_Commits(t *testing.T) {
	client := &fakeMarkClient{headSHA: "abc123"}
	src := config.Source{Repo: "realm/SwiftLint", Branch: "main"}

	tag, sha, err := markSource(client, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "" {
		t.Errorf("tag = %q; want empty for commit-tracked source", tag)
	}
	if sha != "abc123" {
		t.Errorf("sha = %q; want abc123", sha)
	}
}

// TestMarkSource_Releases verifies that a release-tracked source is marked to
// the latest release tag and that tag's SHA.
func TestMarkSource_Releases(t *testing.T) {
	client := &fakeMarkClient{
		release: &github.Release{TagName: "v1.2.0"},
		headSHA: "def456",
	}
	src := config.Source{Repo: "groue/GRDB.swift", Track: "releases"}

	tag, sha, err := markSource(client, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.0" {
		t.Errorf("tag = %q; want v1.2.0", tag)
	}
	if sha != "def456" {
		t.Errorf("sha = %q; want def456", sha)
	}
}

// TestMarkCmdRegistered ensures the mark subcommand is wired onto citeCmd.
func TestMarkCmdRegistered(t *testing.T) {
	found := false
	for _, c := range citeCmd.Commands() {
		if c.Name() == "mark" {
			found = true
			break
		}
	}
	if !found {
		t.Error("citeCmd missing 'mark' subcommand")
	}
}

package cmd

import (
	"strings"
	"testing"

	"github.com/toba/jig-go/v4/internal/classify"
	"github.com/toba/jig-go/v4/internal/config"
	"github.com/toba/jig-go/v4/internal/github"
)

// fakeReleaseClient implements just enough of github.Client to drive
// checkSourceReleases without shelling out to gh.
type fakeReleaseClient struct {
	github.Client
	release *github.Release
	headSHA string
}

func (f *fakeReleaseClient) GetLatestRelease(string) (*github.Release, error) {
	return f.release, nil
}

func (f *fakeReleaseClient) GetHeadSHA(string, string) (string, error) {
	return f.headSHA, nil
}

func (f *fakeReleaseClient) Compare(string, string, string) (*github.CompareResponse, error) {
	return &github.CompareResponse{}, nil
}

// TestCheckSourceReleases_NoNewRelease ensures that when LastCheckedTag
// already equals the latest release's tag_name, the result does not
// re-surface the release as a new change.
func TestCheckSourceReleases_NoNewRelease(t *testing.T) {
	client := &fakeReleaseClient{
		release: &github.Release{TagName: "v7.11.0", Name: "GRDB 7.11.0", Body: "notes"},
		headSHA: "deadbeef",
	}
	src := config.Source{
		Repo:           "groue/GRDB.swift",
		Track:          "releases",
		LastCheckedTag: "v7.11.0",
	}

	result, headSHA, tag, err := checkSourceReleases(client, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v7.11.0" || headSHA != "deadbeef" {
		t.Errorf("tag/headSHA = %q/%q; want v7.11.0/deadbeef", tag, headSHA)
	}
	if result.Release != nil {
		t.Errorf("expected result.Release to be nil when LastCheckedTag == latest.TagName, got %+v", result.Release)
	}
	if len(result.Commits) != 0 {
		t.Errorf("expected no commits, got %d", len(result.Commits))
	}
}

func TestCapDiff_Empty(t *testing.T) {
	out, trunc, skip := capDiff("")
	if out != "" || trunc || skip {
		t.Errorf("capDiff(\"\") = (%q, %v, %v); want (\"\", false, false)", out, trunc, skip)
	}
}

func TestCapDiff_Small(t *testing.T) {
	patch := "@@\n+a\n-b\n"
	out, trunc, skip := capDiff(patch)
	if out != patch || trunc || skip {
		t.Errorf("capDiff small patch unexpectedly modified: out=%q trunc=%v skip=%v", out, trunc, skip)
	}
}

func TestCapDiff_TruncatesByLines(t *testing.T) {
	// Build a patch with diffLineCap+50 lines.
	var b strings.Builder
	for range diffLineCap + 50 {
		b.WriteString("+line\n")
	}
	out, trunc, skip := capDiff(b.String())
	if skip {
		t.Fatal("expected truncated, not skipped")
	}
	if !trunc {
		t.Fatal("expected truncated=true")
	}
	if got := strings.Count(out, "\n"); got != diffLineCap {
		t.Errorf("got %d lines in truncated diff, want %d", got, diffLineCap)
	}
}

func TestCapDiff_SkipsOversize(t *testing.T) {
	patch := strings.Repeat("a", diffSizeCap+1)
	out, trunc, skip := capDiff(patch)
	if !skip {
		t.Fatal("expected skip=true for oversize diff")
	}
	if trunc {
		t.Error("expected trunc=false when skipped")
	}
	if out != "" {
		t.Errorf("expected empty out when skipped, got %d bytes", len(out))
	}
}

func TestBuildFileResult_NoDiffsByDefault(t *testing.T) {
	prev := reviewWithDiffs
	reviewWithDiffs = false
	defer func() { reviewWithDiffs = prev }()

	patches := patchesByFilename([]github.File{{Filename: "a.go", Patch: "+x\n"}})
	if patches != nil {
		t.Errorf("patchesByFilename should return nil when --with-diffs is off, got %v", patches)
	}
	fr := buildFileResult("a.go", classify.High, patches)
	if fr.Diff != "" || fr.DiffTruncated || fr.DiffSkipped {
		t.Errorf("expected empty diff fields by default, got %+v", fr)
	}
}

func TestBuildFileResult_WithDiffs(t *testing.T) {
	prev := reviewWithDiffs
	reviewWithDiffs = true
	defer func() { reviewWithDiffs = prev }()

	patches := patchesByFilename([]github.File{{Filename: "a.go", Patch: "+hello\n"}})
	fr := buildFileResult("a.go", classify.High, patches)
	if fr.Diff != "+hello\n" {
		t.Errorf("expected diff populated, got %q", fr.Diff)
	}
	if fr.DiffTruncated || fr.DiffSkipped {
		t.Errorf("expected no truncation/skip flags, got %+v", fr)
	}

	// Missing patch — no flags, empty diff.
	fr2 := buildFileResult("missing.go", classify.Low, patches)
	if fr2.Diff != "" || fr2.DiffTruncated || fr2.DiffSkipped {
		t.Errorf("expected empty fields for missing patch, got %+v", fr2)
	}
}

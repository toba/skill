package cite

import (
	"strings"
	"testing"

	"github.com/toba/jig-go/v4/internal/config"
)

func TestParseRepoArg(t *testing.T) {
	tests := []struct {
		input    string
		owner    string
		repo     string
		isGitHub bool
	}{
		{"owner/repo", "owner", "repo", true},
		{"toba/jig", "toba", "jig", true},
		{"https://github.com/toba/jig", "toba", "jig", true},
		{"https://github.com/toba/jig.git", "toba", "jig", true},
		{"https://github.com/toba/jig/", "toba", "jig", true},
		{"git@github.com:toba/jig.git", "toba", "jig", true},
		{"git@github.com:toba/jig", "toba", "jig", true},
		{"https://gitlab.com/foo/bar", "", "", false},
		{"https://gitlab.com/foo/bar.git", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseRepoArg(tt.input)
			if got.IsGitHub != tt.isGitHub {
				t.Errorf("IsGitHub = %v, want %v", got.IsGitHub, tt.isGitHub)
			}
			if got.Owner != tt.owner {
				t.Errorf("Owner = %q, want %q", got.Owner, tt.owner)
			}
			if got.Repo != tt.repo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.repo)
			}
		})
	}
}

func TestSuggestPaths(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		wantHigh []string
		wantMed  []string
	}{
		{
			name:     "Go project",
			files:    []string{"main.go", "cmd/root.go", "go.mod", "go.sum", "README.md"},
			wantHigh: []string{"**/*.go"},
			wantMed:  []string{"go.mod", "go.sum"},
		},
		{
			name:     "Swift project with Sources dir",
			files:    []string{"Sources/Lib/Foo.swift", "Sources/Lib/Bar.swift", "Package.swift", "README.md"},
			wantHigh: []string{"Sources/**/*.swift"},
			wantMed:  []string{"Package.swift"},
		},
		{
			name:     "Rust project with src dir",
			files:    []string{"src/main.rs", "src/lib.rs", "Cargo.toml", "Cargo.lock"},
			wantHigh: []string{"src/**/*.rs"},
			wantMed:  []string{"Cargo.toml", "Cargo.lock"},
		},
		{
			name:     "TypeScript project with src dir",
			files:    []string{"src/index.ts", "src/app.tsx", "package.json"},
			wantHigh: []string{"src/**/*.ts", "src/**/*.tsx"},
			wantMed:  []string{"package.json"},
		},
		{
			name:     "Python project",
			files:    []string{"app.py", "utils.py", "requirements.txt", "pyproject.toml"},
			wantHigh: []string{"**/*.py"},
			wantMed:  []string{"requirements.txt", "pyproject.toml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestPaths(tt.files)
			assertSlice(t, "high", got.High, tt.wantHigh)
			assertSlice(t, "medium", got.Medium, tt.wantMed)
			if len(got.Low) == 0 {
				t.Error("expected low paths to be populated")
			}
		})
	}
}

func assertSlice(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %v, want %v", label, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q", label, i, got[i], want[i])
		}
	}
}

func TestFormatSourceYAML(t *testing.T) {
	src := &config.Source{
		Repo:   "toba/jig",
		Branch: "main",
		Scope:  "todo subsystem — issue model, TUI",
		Notes:  "Multi-tool CLI",
		Paths: config.PathDefs{
			High:   []string{"**/*.go"},
			Medium: []string{"go.mod"},
			Low:    []string{".github/**"},
		},
	}
	got, err := FormatSourceYAML(src)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected non-empty output")
	}
	// Verify flow sequence format.
	if !strings.Contains(got, `["**/*.go"]`) {
		t.Errorf("expected flow sequence for high paths, got:\n%s", got)
	}
	for _, want := range []string{"repo: toba/jig", "branch: main", "scope:", "todo subsystem", "notes: Multi-tool CLI", `"**/*.go"`, "go.mod"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestFormatSourceYAMLWithTrack(t *testing.T) {
	src := &config.Source{
		Repo:  "owner/lib",
		Track: "releases",
		Paths: config.PathDefs{
			High: []string{"**/*.go"},
			Low:  []string{".github/**"},
		},
	}
	got, err := FormatSourceYAML(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "track: releases") {
		t.Errorf("expected track: releases in output:\n%s", got)
	}
	// Branch should not appear when empty.
	if strings.Contains(got, "branch:") {
		t.Errorf("expected no branch line when empty:\n%s", got)
	}
}

func TestExtractRepoSlug(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://gitlab.com/foo/bar", "foo/bar"},
		{"https://gitlab.com/foo/bar.git", "foo/bar"},
		{"git@gitlab.com:foo/bar.git", "foo/bar"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractRepoSlug(tt.url)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

package classify

import (
	"testing"

	"github.com/toba/jig-go/v4/internal/config"
)

func TestClassify(t *testing.T) {
	paths := config.PathDefs{
		High:   []string{"Sources/**/*.swift"},
		Medium: []string{"Package.swift", "Tests/**"},
		Low:    []string{".github/**", "README.md"},
	}

	files := []string{
		"Sources/Tools/Project/Target.swift",
		"Package.swift",
		"Tests/ToolTests/ProjectTests.swift",
		".github/workflows/ci.yml",
		"README.md",
		"Makefile",
	}

	results := Classify(files, paths)

	expected := map[string]Level{
		"Sources/Tools/Project/Target.swift": High,
		"Package.swift":                      Medium,
		"Tests/ToolTests/ProjectTests.swift": Medium,
		".github/workflows/ci.yml":           Low,
		"README.md":                          Low,
		"Makefile":                           Unclassified,
	}

	for _, r := range results {
		want, ok := expected[r.Path]
		if !ok {
			t.Errorf("unexpected file %q", r.Path)
			continue
		}
		if r.Level != want {
			t.Errorf("file %q: level = %s, want %s", r.Path, r.Level, want)
		}
	}
}

func TestClassifyHighestWins(t *testing.T) {
	paths := config.PathDefs{
		High:   []string{"**/*.swift"},
		Medium: []string{"Sources/**"},
	}

	results := Classify([]string{"Sources/Foo.swift"}, paths)
	if len(results) != 1 || results[0].Level != High {
		t.Errorf("expected HIGH for overlapping match, got %s", results[0].Level)
	}
}

func TestMaxLevel(t *testing.T) {
	results := []Result{
		{Path: "a", Level: Low},
		{Path: "b", Level: High},
		{Path: "c", Level: Medium},
	}
	if got := MaxLevel(results); got != High {
		t.Errorf("MaxLevel = %s, want HIGH", got)
	}
}

func TestGroupByLevel(t *testing.T) {
	results := []Result{
		{Path: "a", Level: High},
		{Path: "b", Level: High},
		{Path: "c", Level: Low},
	}
	grouped := GroupByLevel(results)
	if len(grouped[High]) != 2 {
		t.Errorf("HIGH group = %d, want 2", len(grouped[High]))
	}
	if len(grouped[Low]) != 1 {
		t.Errorf("LOW group = %d, want 1", len(grouped[Low]))
	}
}

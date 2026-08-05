package classify

import (
	"github.com/bmatcuk/doublestar/v4"
	"github.com/toba/jig-go/v4/internal/config"
)

// Level represents the relevance level of a file change.
type Level int

const (
	Unclassified Level = iota
	Low
	Medium
	High
)

func (l Level) String() string {
	switch l {
	case High:
		return "HIGH"
	case Medium:
		return "MEDIUM"
	case Low:
		return "LOW"
	default:
		return "UNCLASSIFIED"
	}
}

// Result holds the classification of a single file.
type Result struct {
	Path  string
	Level Level
}

// Classify matches file paths against the source's path patterns and returns
// results grouped by relevance. Highest matching level wins.
func Classify(files []string, paths config.PathDefs) []Result {
	results := make([]Result, 0, len(files))
	for _, f := range files {
		level := matchLevel(f, paths)
		results = append(results, Result{Path: f, Level: level})
	}
	return results
}

// MaxLevel returns the highest classification level from a set of results.
func MaxLevel(results []Result) Level {
	max := Unclassified
	for _, r := range results {
		if r.Level > max {
			max = r.Level
		}
	}
	return max
}

// GroupByLevel groups results by their classification level.
func GroupByLevel(results []Result) map[Level][]Result {
	grouped := make(map[Level][]Result)
	for _, r := range results {
		grouped[r.Level] = append(grouped[r.Level], r)
	}
	return grouped
}

func matchLevel(path string, defs config.PathDefs) Level {
	if matchAny(path, defs.High) {
		return High
	}
	if matchAny(path, defs.Medium) {
		return Medium
	}
	if matchAny(path, defs.Low) {
		return Low
	}
	return Unclassified
}

func matchAny(path string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := doublestar.Match(p, path); matched {
			return true
		}
	}
	return false
}

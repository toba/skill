package update

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type migration struct {
	sectionKey string
	candidates []string
}

var migrations = []migration{
	{sectionKey: "nope", candidates: []string{".claude/nope.yml", ".claude/nope.yaml"}},
}

// Run discovers legacy config files and merges them into jigPath.
func Run(jigPath string) error {
	type found struct {
		migration
		path    string
		content []byte
	}

	// Discover legacy files.
	var hits []found
	for _, m := range migrations {
		for _, c := range m.candidates {
			data, err := os.ReadFile(c)
			if err != nil {
				continue
			}
			hits = append(hits, found{migration: m, path: c, content: data})
			break // first hit wins
		}
	}

	// Rename legacy upstream: key to citations: in .jig.yaml.
	if renamed, err := migrateUpstreamKey(jigPath); err != nil {
		return fmt.Errorf("upstream key migration: %w", err)
	} else if renamed {
		fmt.Fprintf(os.Stderr, "update: renamed upstream → citations in %s\n", jigPath)
	}

	// Migrate legacy upstream skill (parsed from SKILL.md, not a verbatim copy).
	upMigrated, upPath, err := migrateCiteSkill(jigPath)
	if err != nil {
		return fmt.Errorf("cite skill migration: %w", err)
	}
	if upMigrated {
		fmt.Fprintf(os.Stderr, "update: migrated %s → %s (citations section)\n", upPath, jigPath)
	}

	// Migrate commit command (scripts/commit.sh → jigo commit).
	commitMigrated, err := migrateCommitCommand(jigPath)
	if err != nil {
		return fmt.Errorf("commit command migration: %w", err)
	}

	if len(hits) == 0 && !upMigrated && !commitMigrated {
		fmt.Fprintln(os.Stderr, "update: no legacy config files found")
		return nil
	}

	if len(hits) == 0 {
		return nil
	}

	// Read existing .jig.yaml (may have been updated by upstream migration).
	existing, err := os.ReadFile(jigPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", jigPath, err)
	}
	tobaContent := string(existing)

	var migrated []found
	lines := splitLines(tobaContent)

	for _, h := range hits {
		if sectionExists(lines, h.sectionKey) {
			fmt.Fprintf(os.Stderr, "update: skipped %s — section already exists in %s\n", h.sectionKey, jigPath)
			continue
		}
		migrated = append(migrated, h)
	}

	if len(migrated) == 0 {
		return nil
	}

	// Append migrated content.
	for _, h := range migrated {
		// Ensure trailing newline + blank separator.
		if tobaContent != "" && !strings.HasSuffix(tobaContent, "\n\n") {
			if !strings.HasSuffix(tobaContent, "\n") {
				tobaContent += "\n"
			}
			tobaContent += "\n"
		}
		tobaContent += string(wrapInSection(h.sectionKey, h.content))
		// Ensure content ends with newline.
		if !strings.HasSuffix(tobaContent, "\n") {
			tobaContent += "\n"
		}
	}

	// Write .jig.yaml.
	if err := os.WriteFile(jigPath, []byte(tobaContent), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", jigPath, err)
	}

	// Cleanup legacy files and report.
	for _, h := range migrated {
		if err := os.Remove(h.path); err != nil {
			fmt.Fprintf(os.Stderr, "update: warning: could not remove %s: %v\n", h.path, err)
		}
		fmt.Fprintf(os.Stderr, "update: migrated %s → %s (%s section)\n", h.path, jigPath, h.sectionKey)
	}

	return nil
}

// migrateUpstreamKey renames an existing "upstream:" key to "citations:" in .jig.yaml
// using the yaml.v3 Node API to preserve formatting. Returns true if renamed.
func migrateUpstreamKey(jigPath string) (bool, error) {
	data, err := os.ReadFile(jigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", jigPath, err)
	}

	lines := splitLines(string(data))
	if !sectionExists(lines, "upstream") {
		return false, nil
	}
	// Already has citations: — don't double-migrate.
	if sectionExists(lines, "citations") {
		return false, nil
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return false, fmt.Errorf("parsing %s: %w", jigPath, err)
	}

	mapping := &root
	if mapping.Kind == yaml.DocumentNode && len(mapping.Content) > 0 {
		mapping = mapping.Content[0]
	}
	if mapping.Kind != yaml.MappingNode {
		return false, nil
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value != "upstream" {
			continue
		}
		mapping.Content[i].Value = "citations"
		out, err := yaml.Marshal(&root)
		if err != nil {
			return false, fmt.Errorf("marshaling %s: %w", jigPath, err)
		}
		if err := os.WriteFile(jigPath, out, 0o644); err != nil {
			return false, fmt.Errorf("writing %s: %w", jigPath, err)
		}
		return true, nil
	}
	return false, nil
}

// wrapInSection wraps content under sectionKey if it doesn't already start with it.
func wrapInSection(key string, content []byte) []byte {
	prefix := key + ":"
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && (lines[0] == prefix || strings.HasPrefix(lines[0], prefix+" ")) {
		return content // already wrapped
	}
	var b strings.Builder
	b.WriteString(prefix + "\n")
	for _, line := range lines {
		if line == "" {
			b.WriteString("\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}
	return []byte(strings.TrimRight(b.String(), "\n") + "\n")
}

// splitLines splits s into lines, preserving empty lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// sectionExists checks if sectionKey appears as a top-level YAML key.
func sectionExists(lines []string, key string) bool {
	prefix := key + ":"
	for _, l := range lines {
		if l == prefix || strings.HasPrefix(l, prefix+" ") {
			return true
		}
	}
	return false
}

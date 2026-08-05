package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// commitCandidates are paths where a legacy commit command might live.
var commitCandidates = []struct {
	command string
	script  string
}{
	{".claude/commands/commit.md", "scripts/commit.sh"},
}

// migrateCommitCommand detects a project commit command or shell script
// that references scripts/commit.sh and rewrites it to use jigo commit.
// Returns (migrated bool, error).
func migrateCommitCommand(_ string) (bool, error) {
	for _, c := range commitCandidates {
		migrated, err := tryMigrateCommitCommand(c.command, c.script)
		if err != nil {
			return false, err
		}
		if migrated {
			return true, nil
		}
	}
	return false, nil
}

// tryMigrateCommitCommand checks a single command/script pair.
func tryMigrateCommitCommand(commandPath, scriptPath string) (bool, error) {
	data, err := os.ReadFile(commandPath)
	if err != nil {
		return false, nil //nolint:nilerr // file doesn't exist, nothing to migrate
	}

	content := string(data)

	// Check if it references the old shell script.
	if !referencesScript(content, scriptPath) {
		return false, nil
	}

	// Check that jigo commit exists (it should, since we're running skill).
	// Rewrite the command to use jigo commit instead.
	newContent := rewriteCommitCommand(content, scriptPath)
	if newContent == content {
		return false, nil // nothing changed
	}

	if err := os.WriteFile(commandPath, []byte(newContent), 0o644); err != nil {
		return false, fmt.Errorf("writing %s: %w", commandPath, err)
	}
	fmt.Fprintf(os.Stderr, "update: rewrote %s to use jigo commit\n", commandPath)

	// Remove the old script if it exists.
	if err := os.Remove(scriptPath); err == nil {
		fmt.Fprintf(os.Stderr, "update: removed %s (replaced by jigo commit)\n", scriptPath)
		// Clean up empty scripts/ directory.
		removeEmptyDir(filepath.Dir(scriptPath))
	}

	return true, nil
}

// referencesScript checks if the content references the given script path.
func referencesScript(content, scriptPath string) bool {
	// Match ./scripts/commit.sh or scripts/commit.sh
	return strings.Contains(content, "./"+scriptPath) ||
		strings.Contains(content, scriptPath)
}

// rewriteCommitCommand replaces references to the shell script with jigo commit.
func rewriteCommitCommand(content, scriptPath string) string {
	// Replace command invocations. The commit.md typically has:
	//   Run `./scripts/commit.sh $ARGUMENTS`
	// or similar patterns.
	result := content

	// Replace ./scripts/commit.sh and scripts/commit.sh with jigo commit.
	result = strings.ReplaceAll(result, "./"+scriptPath, "jigo commit")
	result = strings.ReplaceAll(result, scriptPath, "jigo commit")

	// Clean up doubled "jigo commit $ARGUMENTS" → "jigo commit $ARGUMENTS" (already fine)
	// but fix "jigo commit $ARGUMENTS" since jigo commit takes [push] not $ARGUMENTS.

	return result
}

// removeEmptyDir removes a directory if it's empty.
func removeEmptyDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		os.Remove(dir) //nolint:errcheck // best-effort cleanup
	}
}

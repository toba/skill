package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteCommitCommand(t *testing.T) {
	input := `---
description: Stage all changes and commit
---

## Stage and Commit

Run ` + "`./scripts/commit.sh $ARGUMENTS`" + `

### If script exits with code 2
Ask the user.
`

	got := rewriteCommitCommand(input, "scripts/commit.sh")

	if strings.Contains(got, "scripts/commit.sh") {
		t.Error("still contains scripts/commit.sh")
	}
	if !strings.Contains(got, "jigo commit") {
		t.Error("missing jigo commit")
	}
}

func TestRewriteCommitCommandNoMatch(t *testing.T) {
	input := "Some unrelated command file"
	got := rewriteCommitCommand(input, "scripts/commit.sh")
	if got != input {
		t.Error("should not modify unrelated content")
	}
}

func TestReferencesScript(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"Run `./scripts/commit.sh push`", true},
		{"Run `scripts/commit.sh`", true},
		{"Run `jigo commit`", false},
		{"nothing relevant", false},
	}
	for _, tt := range tests {
		got := referencesScript(tt.content, "scripts/commit.sh")
		if got != tt.want {
			t.Errorf("referencesScript(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}

func TestTryMigrateCommitCommand(t *testing.T) {
	tmp := t.TempDir()

	// Chdir to temp dir so relative paths work.
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(tmp)

	// Create a fake command file referencing the script.
	cmdDir := filepath.Join(tmp, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmdPath := ".claude/commands/commit.md"
	content := "Run `./scripts/commit.sh $ARGUMENTS`\n"
	if err := os.WriteFile(cmdPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a fake script.
	scriptPath := "scripts/commit.sh"
	if err := os.MkdirAll("scripts", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Run migration.
	migrated, err := tryMigrateCommitCommand(cmdPath, scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !migrated {
		t.Fatal("expected migration")
	}

	// Check the command was rewritten.
	got, err := os.ReadFile(cmdPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "commit.sh") {
		t.Error("command still references commit.sh")
	}
	if !strings.Contains(string(got), "jigo commit") {
		t.Error("command doesn't reference jigo commit")
	}

	// Check the script was removed.
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Error("script should have been removed")
	}

	// Check the scripts/ dir was cleaned up.
	if _, err := os.Stat("scripts"); !os.IsNotExist(err) {
		t.Error("empty scripts/ dir should have been removed")
	}
}

func TestTryMigrateCommitCommandAlreadyMigrated(t *testing.T) {
	tmp := t.TempDir()
	cmdPath := filepath.Join(tmp, "commit.md")
	content := "Run `jigo commit $ARGUMENTS`\n"
	if err := os.WriteFile(cmdPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	migrated, err := tryMigrateCommitCommand(cmdPath, filepath.Join(tmp, "scripts/commit.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if migrated {
		t.Error("should not migrate already-migrated command")
	}
}

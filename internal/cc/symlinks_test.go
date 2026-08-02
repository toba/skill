package cc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func setupSyncTest(t *testing.T) (*Config, string) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, ".claude")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	// Shared entries.
	for _, n := range []string{"agents", "skills", "CLAUDE.md"} {
		p := filepath.Join(source, n)
		if filepath.Ext(n) == "" {
			_ = os.MkdirAll(p, 0o755)
		} else {
			_ = os.WriteFile(p, []byte("x"), 0o644)
		}
	}
	// Private entries (exist in source, must be ignored).
	_ = os.WriteFile(filepath.Join(source, ".credentials.json"), []byte("secret"), 0o644)

	work := filepath.Join(root, ".jig", "cc", "work")
	c := &Config{
		Version:      1,
		SharedSource: source,
		Private:      DefaultPrivate,
		Aliases: map[string]Alias{
			"main": {CLI: "claude", Path: source, IsSource: true},
			"work": {CLI: "claude", Path: work},
		},
	}
	return c, work
}

func TestSyncCreatesSymlinks(t *testing.T) {
	c, work := setupSyncTest(t)
	rep, err := Sync(c, "work")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"CLAUDE.md", "agents", "skills"}
	got := slices.Clone(rep.Created)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("created: got %v want %v", got, want)
	}
	for _, n := range want {
		link := filepath.Join(work, n)
		info, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("missing link %s: %v", link, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", link)
		}
	}
	// Private should NOT be linked.
	if _, err := os.Lstat(filepath.Join(work, ".credentials.json")); !os.IsNotExist(err) {
		t.Errorf(".credentials.json should not be linked into alias")
	}
}

func TestSyncIdempotent(t *testing.T) {
	c, _ := setupSyncTest(t)
	if _, err := Sync(c, "work"); err != nil {
		t.Fatal(err)
	}
	rep, err := Sync(c, "work")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Created) != 0 {
		t.Errorf("second sync should create nothing, got %v", rep.Created)
	}
	if len(rep.Skipped) == 0 {
		t.Error("second sync should skip existing links")
	}
}

func TestSyncRepairsWrongTarget(t *testing.T) {
	c, work := setupSyncTest(t)
	if _, err := Sync(c, "work"); err != nil {
		t.Fatal(err)
	}
	// Repoint a link to a bad target.
	link := filepath.Join(work, "agents")
	_ = os.Remove(link)
	_ = os.Symlink("/nonexistent", link)

	rep, err := Sync(c, "work")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(rep.Repaired, "agents") {
		t.Errorf("expected repaired to include 'agents', got %v", rep.Repaired)
	}
}

func TestSyncReportsConflict(t *testing.T) {
	c, work := setupSyncTest(t)
	_ = os.MkdirAll(work, 0o755)
	// Real file where a symlink should be.
	if err := os.WriteFile(filepath.Join(work, "agents"), []byte("oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := Sync(c, "work")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(rep.Conflicts, "agents") {
		t.Errorf("expected conflict on 'agents', got %v", rep.Conflicts)
	}
	// And the real file should still be intact.
	data, _ := os.ReadFile(filepath.Join(work, "agents"))
	if string(data) != "oops" {
		t.Error("conflicted file should not be overwritten")
	}
}

func TestSeedClaudeJSONStripsIdentity(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, ".claude")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	// A source .claude.json carrying account/install identity plus benign
	// launch config that should survive seeding.
	srcJSON := `{
		"machineID": "MACHINE-AAA",
		"userID": "USER-AAA",
		"oauthAccount": {"accountUuid": "ACC-AAA", "emailAddress": "a@example.com"},
		"hasCompletedOnboarding": true,
		"mcpServers": {"foo": {"command": "bar"}}
	}`
	if err := os.WriteFile(filepath.Join(source, ".claude.json"), []byte(srcJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	work := filepath.Join(root, ".jig", "cc", "work")
	c := &Config{
		Version:      1,
		SharedSource: source,
		Private:      DefaultPrivate,
		Aliases: map[string]Alias{
			"main": {CLI: "claude", Path: source, IsSource: true},
			"work": {CLI: "claude", Path: work},
		},
	}
	if err := SeedClaudeJSON(c, "work"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(work, ".claude.json"))
	if err != nil {
		t.Fatalf("seeded .claude.json missing: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("seeded .claude.json is not valid JSON: %v", err)
	}
	for _, k := range []string{"machineID", "userID", "oauthAccount"} {
		if _, ok := m[k]; ok {
			t.Errorf("identity field %q must be stripped from seeded .claude.json", k)
		}
	}
	for _, k := range []string{"hasCompletedOnboarding", "mcpServers"} {
		if _, ok := m[k]; !ok {
			t.Errorf("benign field %q should be preserved when seeding", k)
		}
	}
}

func TestSeedClaudeJSONDoesNotClobberExisting(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, ".claude")
	work := filepath.Join(root, ".jig", "cc", "work")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	// The alias already has its own account's .claude.json — seeding must
	// leave it untouched, never overwrite it with the source's.
	existing := `{"machineID":"WORK-OWN","userID":"WORK-USER"}`
	dst := filepath.Join(work, ".claude.json")
	if err := os.WriteFile(dst, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, ".claude.json"), []byte(`{"machineID":"SRC"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Config{
		Version:      1,
		SharedSource: source,
		Private:      DefaultPrivate,
		Aliases: map[string]Alias{
			"main": {CLI: "claude", Path: source, IsSource: true},
			"work": {CLI: "claude", Path: work},
		},
	}
	if err := SeedClaudeJSON(c, "work"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("existing .claude.json was clobbered: got %q want %q", got, existing)
	}
}

func TestSeedClaudeJSONSourceAbsent(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, ".claude") // no .claude.json inside
	work := filepath.Join(root, ".jig", "cc", "work")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	c := &Config{
		Version:      1,
		SharedSource: source,
		Private:      DefaultPrivate,
		Aliases: map[string]Alias{
			"main": {CLI: "claude", Path: source, IsSource: true},
			"work": {CLI: "claude", Path: work},
		},
	}
	if err := SeedClaudeJSON(c, "work"); err != nil {
		t.Fatalf("absent source should not error: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(work, ".claude.json")); !os.IsNotExist(err) {
		t.Error("no .claude.json should be written when source is absent")
	}
}

func TestScrubIdentity(t *testing.T) {
	// Empty payload yields an empty object, not an error.
	out, err := scrubIdentity(nil)
	if err != nil {
		t.Fatalf("empty payload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("empty scrub result is not JSON: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("empty payload should scrub to {}, got %v", m)
	}
	// Malformed JSON surfaces an error.
	if _, err := scrubIdentity([]byte("{not json")); err == nil {
		t.Error("malformed JSON should return an error")
	}
}

func TestCheckIdentityDetectsSharedIDs(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, ".claude")
	work := filepath.Join(root, ".jig", "cc", "work")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	// Two aliases (distinct accounts) sharing the same machineID + userID:
	// the exact cross-account fingerprint doctor must flag.
	shared := `{"machineID": "SAME-MACHINE", "userID": "SAME-USER"}`
	if err := os.WriteFile(filepath.Join(source, ".claude.json"), []byte(shared), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, ".claude.json"), []byte(shared), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Config{
		Version:      1,
		SharedSource: source,
		Private:      DefaultPrivate,
		Aliases: map[string]Alias{
			"main": {CLI: "claude", Path: source, IsSource: true},
			"work": {CLI: "claude", Path: work},
		},
	}
	cols, err := CheckIdentity(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) == 0 {
		t.Fatal("expected identity collisions for shared machineID/userID")
	}
	for _, col := range cols {
		if len(col.Aliases) < 2 {
			t.Errorf("collision %+v should list at least two aliases", col)
		}
		if !slices.Contains(col.Aliases, "main") || !slices.Contains(col.Aliases, "work") {
			t.Errorf("collision %+v should include both main and work", col)
		}
	}
}

func TestCheckIdentityCleanWhenDistinct(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, ".claude")
	work := filepath.Join(root, ".jig", "cc", "work")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, ".claude.json"), []byte(`{"machineID":"M1","userID":"U1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, ".claude.json"), []byte(`{"machineID":"M2","userID":"U2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Config{
		Version:      1,
		SharedSource: source,
		Private:      DefaultPrivate,
		Aliases: map[string]Alias{
			"main": {CLI: "claude", Path: source, IsSource: true},
			"work": {CLI: "claude", Path: work},
		},
	}
	cols, err := CheckIdentity(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 0 {
		t.Errorf("distinct identities should produce no collisions, got %+v", cols)
	}
}

func TestCheckHealth(t *testing.T) {
	c, work := setupSyncTest(t)
	if _, err := Sync(c, "work"); err != nil {
		t.Fatal(err)
	}
	h, err := CheckHealth(c, "work")
	if err != nil {
		t.Fatal(err)
	}
	if h.HasIssues() {
		t.Errorf("clean state should report no issues: %+v", h)
	}

	// Break a symlink.
	link := filepath.Join(work, "agents")
	_ = os.Remove(link)
	_ = os.Symlink("/nonexistent", link)
	h, _ = CheckHealth(c, "work")
	if !slices.Contains(h.Broken, "agents") {
		t.Errorf("expected broken to include 'agents', got %+v", h)
	}
}

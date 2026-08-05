package nope

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T, dir string)
		wantExit      int
		checkYAML     func(t *testing.T, dir string)
		checkSettings func(t *testing.T, dir string)
	}{
		{
			name:     "creates both files in empty dir",
			setup:    func(t *testing.T, dir string) {},
			wantExit: 0,
			checkYAML: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".jig.yaml")) //nolint:gosec // test path
				if err != nil {
					t.Fatalf(".jig.yaml not created: %v", err)
				}
				if !strings.Contains(string(data), "git-push") {
					t.Error(".jig.yaml missing git-push rule")
				}
				if !strings.Contains(string(data), "no-write-env") {
					t.Error(".jig.yaml missing no-write-env rule")
				}
				if !strings.Contains(string(data), "nope:") {
					t.Error(".jig.yaml missing nope: section")
				}
			},
			checkSettings: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")) //nolint:gosec // test path
				if err != nil {
					t.Fatalf("settings.json not created: %v", err)
				}
				var s map[string]any
				if err := json.Unmarshal(data, &s); err != nil {
					t.Fatalf("settings.json invalid JSON: %v", err)
				}
				if !hasNopeHook(s) {
					t.Error("settings.json missing nope hook")
				}
				// Verify matcher is ".*"
				hooks := s["hooks"].(map[string]any)
				ptu := hooks["PreToolUse"].([]any)
				entry := ptu[0].(map[string]any)
				if entry["matcher"] != ".*" {
					t.Errorf("matcher = %q, want %q", entry["matcher"], ".*")
				}
				// Verify command is "jigo nope"
				innerHooks := entry["hooks"].([]any)
				hm := innerHooks[0].(map[string]any)
				if hm["command"] != "jigo nope" {
					t.Errorf("command = %q, want %q", hm["command"], "jigo nope")
				}
			},
		},
		{
			name: "skips existing nope section in .jig.yaml",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, ".jig.yaml"), []byte("nope:\n  rules: []\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantExit: 0,
			checkYAML: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".jig.yaml")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "nope:\n  rules: []\n" {
					t.Error(".jig.yaml was overwritten")
				}
			},
			checkSettings: func(t *testing.T, dir string) {
				// settings.json should still be created
				if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); err != nil {
					t.Error("settings.json should have been created")
				}
			},
		},
		{
			name: "appends nope section to existing .jig.yaml",
			setup: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, ".jig.yaml"), []byte("citations:\n  sources: []\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantExit: 0,
			checkYAML: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".jig.yaml")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				content := string(data)
				if !strings.Contains(content, "citations:") {
					t.Error("existing citations section was lost")
				}
				if !strings.Contains(content, "nope:") {
					t.Error("nope section not appended")
				}
			},
		},
		{
			name: "skips existing settings.json with nope hook",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
					t.Fatal(err)
				}
				s := map[string]any{
					"hooks": map[string]any{
						"PreToolUse": []any{hookEntry},
					},
				}
				data, err := json.MarshalIndent(s, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), data, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantExit: 0,
			checkSettings: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				var s map[string]any
				if err := json.Unmarshal(data, &s); err != nil {
					t.Fatal(err)
				}
				hooks := s["hooks"].(map[string]any)
				ptu := hooks["PreToolUse"].([]any)
				if len(ptu) != 1 {
					t.Errorf("expected 1 PreToolUse entry, got %d", len(ptu))
				}
			},
		},
		{
			name: "merges into existing settings.json without nope hook",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
					t.Fatal(err)
				}
				s := map[string]any{
					"permissions": map[string]any{
						"allow": []string{"Read"},
					},
				}
				data, err := json.MarshalIndent(s, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), data, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantExit: 0,
			checkSettings: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				var s map[string]any
				if err := json.Unmarshal(data, &s); err != nil {
					t.Fatal(err)
				}
				// Hook should be added
				if !hasNopeHook(s) {
					t.Error("nope hook not added")
				}
				// Existing keys should be preserved
				if _, ok := s["permissions"]; !ok {
					t.Error("existing permissions key was lost")
				}
			},
		},
		{
			name: "migrates existing Bash matcher to .*",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
					t.Fatal(err)
				}
				// Write settings with old "Bash" matcher
				oldHook := map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "jigo nope",
						},
					},
				}
				s := map[string]any{
					"hooks": map[string]any{
						"PreToolUse": []any{oldHook},
					},
				}
				data, err := json.MarshalIndent(s, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), data, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantExit: 0,
			checkSettings: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				var s map[string]any
				if err := json.Unmarshal(data, &s); err != nil {
					t.Fatal(err)
				}
				hooks := s["hooks"].(map[string]any)
				ptu := hooks["PreToolUse"].([]any)
				entry := ptu[0].(map[string]any)
				if entry["matcher"] != ".*" {
					t.Errorf("matcher = %q, want %q", entry["matcher"], ".*")
				}
			},
		},
		{
			name: "migrates nogo command to jigo nope",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
					t.Fatal(err)
				}
				oldHook := map[string]any{
					"matcher": ".*",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "nogo",
						},
					},
				}
				s := map[string]any{
					"hooks": map[string]any{
						"PreToolUse": []any{oldHook},
					},
				}
				data, err := json.MarshalIndent(s, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), data, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantExit: 0,
			checkSettings: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				var s map[string]any
				if err := json.Unmarshal(data, &s); err != nil {
					t.Fatal(err)
				}
				hooks := s["hooks"].(map[string]any)
				ptu := hooks["PreToolUse"].([]any)
				entry := ptu[0].(map[string]any)
				innerHooks := entry["hooks"].([]any)
				hm := innerHooks[0].(map[string]any)
				if hm["command"] != "jigo nope" {
					t.Errorf("command = %q, want %q", hm["command"], "jigo nope")
				}
			},
		},
		{
			// The binary was renamed jig → jigo; existing installs carry the old
			// hook command and must be rewritten in place.
			name: "migrates jig nope command to jigo nope",
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
					t.Fatal(err)
				}
				oldHook := map[string]any{
					"matcher": ".*",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "jig nope",
						},
					},
				}
				s := map[string]any{
					"hooks": map[string]any{
						"PreToolUse": []any{oldHook},
					},
				}
				data, err := json.MarshalIndent(s, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), data, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantExit: 0,
			checkSettings: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				var s map[string]any
				if err := json.Unmarshal(data, &s); err != nil {
					t.Fatal(err)
				}
				hooks := s["hooks"].(map[string]any)
				ptu := hooks["PreToolUse"].([]any)
				if len(ptu) != 1 {
					t.Fatalf("PreToolUse entries = %d, want 1 (should rewrite, not append)", len(ptu))
				}
				entry := ptu[0].(map[string]any)
				innerHooks := entry["hooks"].([]any)
				hm := innerHooks[0].(map[string]any)
				if hm["command"] != "jigo nope" {
					t.Errorf("command = %q, want %q", hm["command"], "jigo nope")
				}
			},
		},
		{
			name: "idempotent — second run changes nothing",
			setup: func(t *testing.T, dir string) {
				// First run
				origDir, err := os.Getwd()
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Chdir(dir); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.Chdir(origDir); err != nil {
						t.Logf("warning: could not restore dir: %v", err)
					}
				}()
				RunInit()
			},
			wantExit: 0,
			checkYAML: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".jig.yaml")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(data), "git-push") {
					t.Error(".jig.yaml content unexpected after second run")
				}
			},
			checkSettings: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")) //nolint:gosec // test path
				if err != nil {
					t.Fatal(err)
				}
				var s map[string]any
				if err := json.Unmarshal(data, &s); err != nil {
					t.Fatal(err)
				}
				hooks := s["hooks"].(map[string]any)
				ptu := hooks["PreToolUse"].([]any)
				if len(ptu) != 1 {
					t.Errorf("expected 1 PreToolUse entry after idempotent run, got %d", len(ptu))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)

			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Chdir(origDir); err != nil {
					t.Logf("warning: could not restore dir: %v", err)
				}
			}()

			got := RunInit()
			if got != tc.wantExit {
				t.Errorf("RunInit() = %d, want %d", got, tc.wantExit)
			}
			if tc.checkYAML != nil {
				tc.checkYAML(t, dir)
			}
			if tc.checkSettings != nil {
				tc.checkSettings(t, dir)
			}
		})
	}
}

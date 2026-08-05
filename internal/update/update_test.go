package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string)
		check      func(t *testing.T, dir string)
		wantErr    bool
		wantStderr string // substring expected in stderr output
	}{
		{
			name: "migrates .claude/nope.yml into .jig.yaml and deletes legacy file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mkdir(t, filepath.Join(dir, ".claude"))
				writeFile(t, filepath.Join(dir, ".claude/nope.yml"), "nope:\n  rules:\n    - name: test\n")
			},
			check: func(t *testing.T, dir string) {
				t.Helper()
				data := readFile(t, filepath.Join(dir, ".jig.yaml"))
				if !strings.Contains(data, "nope:") {
					t.Error(".jig.yaml missing nope section")
				}
				if !strings.Contains(data, "- name: test") {
					t.Error(".jig.yaml missing rule content")
				}
				assertRemoved(t, filepath.Join(dir, ".claude/nope.yml"))
			},
		},
		{
			name: "skips when section already exists in .jig.yaml",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeFile(t, filepath.Join(dir, ".jig.yaml"), "nope:\n  rules:\n    - name: existing\n")
				mkdir(t, filepath.Join(dir, ".claude"))
				writeFile(t, filepath.Join(dir, ".claude/nope.yml"), "nope:\n  rules:\n    - name: legacy\n")
			},
			check: func(t *testing.T, dir string) {
				t.Helper()
				data := readFile(t, filepath.Join(dir, ".jig.yaml"))
				if strings.Contains(data, "legacy") {
					t.Error("legacy content should not have been merged")
				}
				// Legacy file should be preserved when skipped.
				if _, err := os.Stat(filepath.Join(dir, ".claude/nope.yml")); err != nil {
					t.Error(".claude/nope.yml should still exist when skipped")
				}
			},
		},
		{
			name: "prefers .yml over .yaml when both exist",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mkdir(t, filepath.Join(dir, ".claude"))
				writeFile(t, filepath.Join(dir, ".claude/nope.yml"), "nope:\n  rules:\n    - name: from-yml\n")
				writeFile(t, filepath.Join(dir, ".claude/nope.yaml"), "nope:\n  rules:\n    - name: from-yaml\n")
			},
			check: func(t *testing.T, dir string) {
				t.Helper()
				data := readFile(t, filepath.Join(dir, ".jig.yaml"))
				if !strings.Contains(data, "from-yml") {
					t.Error("should have used .yml content")
				}
				if strings.Contains(data, "from-yaml") {
					t.Error("should not have used .yaml content")
				}
				assertRemoved(t, filepath.Join(dir, ".claude/nope.yml"))
				// .yaml variant should still exist (not touched).
				if _, err := os.Stat(filepath.Join(dir, ".claude/nope.yaml")); err != nil {
					t.Error(".claude/nope.yaml should still exist")
				}
			},
		},
		{
			name: "wraps bare rules under nope section when legacy file lacks nope: wrapper",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mkdir(t, filepath.Join(dir, ".claude"))
				writeFile(t, filepath.Join(dir, ".claude/nope.yaml"), "rules:\n  - name: no-force-push\n    glob: \"**\"\n")
			},
			check: func(t *testing.T, dir string) {
				t.Helper()
				data := readFile(t, filepath.Join(dir, ".jig.yaml"))
				if !strings.Contains(data, "nope:\n  rules:") {
					t.Errorf("expected rules nested under nope:, got:\n%s", data)
				}
				if !strings.Contains(data, "    - name: no-force-push") {
					t.Error("rule content not properly indented under nope:")
				}
				assertRemoved(t, filepath.Join(dir, ".claude/nope.yaml"))
			},
		},
		{
			name: "no legacy files found — no error, .jig.yaml unchanged",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeFile(t, filepath.Join(dir, ".jig.yaml"), "citations:\n  sources: []\n")
			},
			check: func(t *testing.T, dir string) {
				t.Helper()
				data := readFile(t, filepath.Join(dir, ".jig.yaml"))
				if data != "citations:\n  sources: []\n" {
					t.Error(".jig.yaml should be unchanged")
				}
			},
		},
		{
			name: "creates .jig.yaml if it doesn't exist",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mkdir(t, filepath.Join(dir, ".claude"))
				writeFile(t, filepath.Join(dir, ".claude/nope.yml"), "nope:\n  rules: []\n")
			},
			check: func(t *testing.T, dir string) {
				t.Helper()
				data := readFile(t, filepath.Join(dir, ".jig.yaml"))
				if !strings.Contains(data, "nope:") {
					t.Error(".jig.yaml should have been created with nope section")
				}
			},
		},
		{
			name: "preserves existing sections during append",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeFile(t, filepath.Join(dir, ".jig.yaml"), "citations:\n  sources:\n    - repo: foo/bar\n")
				mkdir(t, filepath.Join(dir, ".claude"))
				writeFile(t, filepath.Join(dir, ".claude/nope.yml"), "nope:\n  rules: []\n")
			},
			check: func(t *testing.T, dir string) {
				t.Helper()
				data := readFile(t, filepath.Join(dir, ".jig.yaml"))
				if !strings.Contains(data, "citations:") {
					t.Error("existing citations section was lost")
				}
				if !strings.Contains(data, "foo/bar") {
					t.Error("existing citations content was lost")
				}
				if !strings.Contains(data, "nope:") {
					t.Error("nope section not appended")
				}
			},
		},
		{
			name: "idempotent: second run finds nothing to do",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mkdir(t, filepath.Join(dir, ".claude"))
				writeFile(t, filepath.Join(dir, ".claude/nope.yml"), "nope:\n  rules: []\n")
				// First run.
				if err := Run(filepath.Join(dir, ".jig.yaml")); err != nil {
					t.Fatalf("first run: %v", err)
				}
			},
			check: func(t *testing.T, dir string) {
				t.Helper()
				data := readFile(t, filepath.Join(dir, ".jig.yaml"))
				if !strings.Contains(data, "nope:") {
					t.Error("nope section missing after second run")
				}
				// Count occurrences — should be exactly one.
				if strings.Count(data, "nope:") != 1 {
					t.Error("nope section duplicated")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

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

			tc.setup(t, dir)

			err = Run(filepath.Join(dir, ".jig.yaml"))
			if (err != nil) != tc.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.check != nil {
				tc.check(t, dir)
			}
		})
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test path
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatal(err)
	}
}

func assertRemoved(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("%s should have been removed", path)
	}
}

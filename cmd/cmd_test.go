package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toba/jig-go/v4/internal/config"
)

// writeTempConfig creates a .jig.yaml with the given content in a temp dir and returns the path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".jig.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- repoFromGitURL tests ---

func TestRepoFromGitURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"https with .git", "https://github.com/owner/repo.git", "owner/repo"},
		{"https without .git", "https://github.com/owner/repo", "owner/repo"},
		{"ssh with .git", "git@github.com:owner/repo.git", "owner/repo"},
		{"ssh without .git", "git@github.com:owner/repo", "owner/repo"},
		{"bare owner/repo", "owner/repo", "owner/repo"},
		{"https deep path", "https://github.com/org/project", "org/project"},
		{"empty string", "", ""},
		{"absolute path", "/usr/local/bin", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repoFromGitURL(tt.url)
			if got != tt.want {
				t.Errorf("repoFromGitURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

// --- extNameFromRepo tests ---

func TestExtNameFromRepo(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"owner/repo", "toba/gozer", "gozer"},
		{"no slash", "gozer", "gozer"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extNameFromRepo(tt.input)
			if got != tt.want {
				t.Errorf("extNameFromRepo(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- resolveExt tests ---

func TestResolveExt(t *testing.T) {
	t.Run("flag takes precedence", func(t *testing.T) {
		got := resolveExt("toba/gozer", "/nonexistent/path")
		if got != "toba/gozer" {
			t.Errorf("resolveExt() = %q, want %q", got, "toba/gozer")
		}
	})

	t.Run("empty flag and no config returns empty", func(t *testing.T) {
		got := resolveExt("", "/nonexistent/path")
		if got != "" {
			t.Errorf("resolveExt() = %q, want empty", got)
		}
	})
}

// --- configPath tests ---

func TestConfigPath(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		old := cfgPath
		defer func() { cfgPath = old }()
		cfgPath = ""

		got := configPath()
		if got != ".jig.yaml" {
			t.Errorf("configPath() = %q, want %q", got, ".jig.yaml")
		}
	})

	t.Run("custom", func(t *testing.T) {
		old := cfgPath
		defer func() { cfgPath = old }()
		cfgPath = "/custom/path.yaml"

		got := configPath()
		if got != "/custom/path.yaml" {
			t.Errorf("configPath() = %q, want %q", got, "/custom/path.yaml")
		}
	})
}

// --- collectFlags tests ---

func TestCollectFlags(t *testing.T) {
	// Test on the applyCmd which has known flags.
	flags := collectFlags(applyCmd)

	var hasMessage, hasVersion, hasPush bool
	for _, f := range flags {
		if strings.Contains(f, "--message") {
			hasMessage = true
		}
		if strings.Contains(f, "--version") {
			hasVersion = true
		}
		if strings.Contains(f, "--push") {
			hasPush = true
		}
	}

	if !hasMessage {
		t.Error("collectFlags(applyCmd) missing --message")
	}
	if !hasVersion {
		t.Error("collectFlags(applyCmd) missing --version")
	}
	if !hasPush {
		t.Error("collectFlags(applyCmd) missing --push")
	}
}

// --- command structure tests ---

func TestCommandStructure(t *testing.T) {
	// Verify key commands are registered on rootCmd.
	cmds := rootCmd.Commands()
	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Name()] = true
	}

	expected := []string{"cite", "nope", "brew", "zed", "commit", "scoop", "cc", "doctor", "version", "help-all", "update"}
	for _, name := range expected {
		if !cmdNames[name] {
			t.Errorf("rootCmd missing subcommand %q", name)
		}
	}
}

func TestCommitSubcommands(t *testing.T) {
	cmds := commitCmd.Commands()
	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Name()] = true
	}

	if !cmdNames["gather"] {
		t.Error("commitCmd missing 'gather' subcommand")
	}
	if !cmdNames["apply"] {
		t.Error("commitCmd missing 'apply' subcommand")
	}
}

func TestApplyCmdRequiredFlags(t *testing.T) {
	// The --message flag should be marked required.
	f := applyCmd.Flags().Lookup("message")
	if f == nil {
		t.Fatal("applyCmd missing --message flag")
	}

	// Check shorthand.
	if f.Shorthand != "m" {
		t.Errorf("--message shorthand = %q, want %q", f.Shorthand, "m")
	}
}

func TestGatherCmdNoArgs(t *testing.T) {
	if gatherCmd.Args == nil {
		t.Fatal("gatherCmd.Args should not be nil")
	}
	// cobra.NoArgs returns error when args provided.
	err := gatherCmd.Args(gatherCmd, []string{"extra"})
	if err == nil {
		t.Error("gatherCmd should reject arguments")
	}
}

func TestApplyCmdNoArgs(t *testing.T) {
	if applyCmd.Args == nil {
		t.Fatal("applyCmd.Args should not be nil")
	}
	err := applyCmd.Args(applyCmd, []string{"extra"})
	if err == nil {
		t.Error("applyCmd should reject arguments")
	}
}

func TestNopeSubcommands(t *testing.T) {
	cmds := nopeCmd.Commands()
	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Name()] = true
	}

	for _, name := range []string{"init", "doctor", "help"} {
		if !cmdNames[name] {
			t.Errorf("nopeCmd missing %q subcommand", name)
		}
	}
}

func TestCiteSubcommands(t *testing.T) {
	cmds := citeCmd.Commands()
	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Name()] = true
	}

	for _, name := range []string{"init", "review", "add", "update"} {
		if !cmdNames[name] {
			t.Errorf("citeCmd missing %q subcommand", name)
		}
	}
}

func TestReviewCmdAlias(t *testing.T) {
	if len(reviewCmd.Aliases) == 0 {
		t.Fatal("reviewCmd should have aliases")
	}
	found := false
	for _, a := range reviewCmd.Aliases {
		if a == "check" {
			found = true
		}
	}
	if !found {
		t.Error("reviewCmd missing 'check' alias")
	}
}

func TestCiteUpdateCmdFlags(t *testing.T) {
	flags := []string{
		"branch", "track", "scope", "notes", "repo",
		"paths-high", "paths-medium", "paths-low",
		"clear-track", "clear-scope", "clear-notes",
		"clear-paths-high", "clear-paths-medium", "clear-paths-low",
	}
	for _, name := range flags {
		f := citeUpdateCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("citeUpdateCmd missing --%s flag", name)
		}
	}
}

func TestRunCiteUpdate(t *testing.T) {
	t.Run("updates branch and scope", func(t *testing.T) {
		path := writeTempConfig(t, `citations:
  - repo: owner/name
    branch: main
    scope: "old scope"
    paths:
      high: ["**/*.go"]
`)
		old := cfgPath
		defer func() { cfgPath = old }()
		cfgPath = path

		citeUpdateCmd.Flags().Set("branch", "develop")
		citeUpdateCmd.Flags().Set("scope", "new scope")
		defer func() {
			citeUpdateCmd.Flags().Set("branch", "")
			citeUpdateCmd.Flags().Set("scope", "")
		}()

		if err := runCiteUpdate(citeUpdateCmd, []string{"owner/name"}); err != nil {
			t.Fatalf("runCiteUpdate() error: %v", err)
		}

		_, c, err := config.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		src := config.FindSource(c, "owner/name")
		if src == nil {
			t.Fatal("source not found after update")
		}
		if src.Branch != "develop" {
			t.Errorf("branch = %q, want develop", src.Branch)
		}
		if src.Scope != "new scope" {
			t.Errorf("scope = %q, want 'new scope'", src.Scope)
		}
	})

	t.Run("source not found", func(t *testing.T) {
		path := writeTempConfig(t, `citations:
  - repo: owner/name
    branch: main
    paths:
      high: ["**/*.go"]
`)
		old := cfgPath
		defer func() { cfgPath = old }()
		cfgPath = path

		err := runCiteUpdate(citeUpdateCmd, []string{"nonexistent"})
		if err == nil {
			t.Fatal("expected error for nonexistent source")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want 'not found'", err.Error())
		}
	})
}

func TestBrewSubcommands(t *testing.T) {
	cmds := brewCmd.Commands()
	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Name()] = true
	}

	for _, name := range []string{"init", "doctor"} {
		if !cmdNames[name] {
			t.Errorf("brewCmd missing %q subcommand", name)
		}
	}
}

func TestZedSubcommands(t *testing.T) {
	cmds := zedCmd.Commands()
	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Name()] = true
	}

	for _, name := range []string{"init", "doctor"} {
		if !cmdNames[name] {
			t.Errorf("zedCmd missing %q subcommand", name)
		}
	}
}

// --- flag parsing tests ---

func TestBrewInitFlags(t *testing.T) {
	flags := []string{"tap", "tag", "repo", "desc", "license", "dry-run"}
	for _, name := range flags {
		f := brewInitCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("brewInitCmd missing --%s flag", name)
		}
	}
}

func TestZedInitFlags(t *testing.T) {
	flags := []string{"ext", "tag", "repo", "desc", "lsp-name", "languages", "dry-run"}
	for _, name := range flags {
		f := zedInitCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("zedInitCmd missing --%s flag", name)
		}
	}
}

func TestApplyCmdFlags(t *testing.T) {
	flags := []string{"message", "version", "push"}
	for _, name := range flags {
		f := applyCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("applyCmd missing --%s flag", name)
		}
	}
}

func TestRootCmdPersistentFlags(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("config")
	if f == nil {
		t.Error("rootCmd missing --config persistent flag")
	}

	f = rootCmd.PersistentFlags().Lookup("json")
	if f == nil {
		t.Error("rootCmd missing --json persistent flag")
	}
}

// --- resolveTap tests ---

func TestResolveTap(t *testing.T) {
	t.Run("flag takes precedence", func(t *testing.T) {
		got, err := resolveTap("owner/homebrew-tool", "/nonexistent/path")
		if err != nil {
			t.Fatalf("resolveTap() error: %v", err)
		}
		if got != "owner/homebrew-tool" {
			t.Errorf("resolveTap() = %q, want %q", got, "owner/homebrew-tool")
		}
	})

	t.Run("no flag falls through to convention or error", func(t *testing.T) {
		got, err := resolveTap("", "/nonexistent/path/.jig.yaml")
		if err != nil {
			// Expected when gh CLI is not available or not in a GitHub repo.
			if !strings.Contains(err.Error(), "--tap is required") {
				t.Errorf("resolveTap() error = %q, expected '--tap is required'", err.Error())
			}
		} else {
			// If gh succeeds, we should get a convention-based tap.
			if got == "" {
				t.Error("resolveTap() returned empty string without error")
			}
			if !strings.Contains(got, "homebrew-") {
				t.Errorf("resolveTap() convention = %q, expected it to contain 'homebrew-'", got)
			}
		}
	})
}

// --- printCommandTree is hard to unit test but we can verify
// collectFlags works with various commands ---

func TestCollectFlagsOnRootCmd(t *testing.T) {
	// Root cmd has persistent flags but no local flags (except help).
	flags := collectFlags(rootCmd)
	// Should not include help.
	for _, f := range flags {
		if strings.Contains(f, "--help") {
			t.Error("collectFlags should not include --help")
		}
	}
}

func TestCollectFlagsOnBrewInit(t *testing.T) {
	flags := collectFlags(brewInitCmd)
	if len(flags) == 0 {
		t.Error("collectFlags(brewInitCmd) returned empty, expected flags")
	}

	// Check that dry-run (bool) does not have <type> annotation.
	for _, f := range flags {
		if strings.Contains(f, "--dry-run") {
			if strings.Contains(f, "<") {
				t.Errorf("bool flag --dry-run should not have type annotation: %s", f)
			}
		}
	}
}

// --- starterConfig test ---

func TestStarterConfig(t *testing.T) {
	if !strings.Contains(starterConfig, "citations:") {
		t.Error("starterConfig missing 'citations:' key")
	}
	if !strings.Contains(starterConfig, "owner/repo") {
		t.Error("starterConfig missing 'owner/repo' placeholder")
	}
	if !strings.Contains(starterConfig, "high:") {
		t.Error("starterConfig missing 'high:' classification")
	}
	if !strings.Contains(starterConfig, "medium:") {
		t.Error("starterConfig missing 'medium:' classification")
	}
	if !strings.Contains(starterConfig, "low:") {
		t.Error("starterConfig missing 'low:' classification")
	}
}

// --- runInit (cite init) tests ---

func TestRunInit(t *testing.T) {
	t.Run("creates new config file", func(t *testing.T) {
		dir := t.TempDir()
		old := cfgPath
		defer func() { cfgPath = old }()
		cfgPath = filepath.Join(dir, ".jig.yaml")

		err := runInit(nil, nil)
		if err != nil {
			t.Fatalf("runInit() error: %v", err)
		}

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "citations:") {
			t.Error("created file missing 'citations:' section")
		}
	})

	t.Run("appends to existing file without citations", func(t *testing.T) {
		dir := t.TempDir()
		old := cfgPath
		defer func() { cfgPath = old }()
		cfgPath = filepath.Join(dir, ".jig.yaml")

		if err := os.WriteFile(cfgPath, []byte("nope:\n  network: block\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := runInit(nil, nil)
		if err != nil {
			t.Fatalf("runInit() error: %v", err)
		}

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if !strings.Contains(content, "nope:") {
			t.Error("existing content was lost")
		}
		if !strings.Contains(content, "citations:") {
			t.Error("citations section not added")
		}
	})

	t.Run("errors if citations section already exists", func(t *testing.T) {
		dir := t.TempDir()
		old := cfgPath
		defer func() { cfgPath = old }()
		cfgPath = filepath.Join(dir, ".jig.yaml")

		if err := os.WriteFile(cfgPath, []byte("citations:\n  - repo: owner/repo\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := runInit(nil, nil)
		if err == nil {
			t.Fatal("runInit() expected error when citations section already exists")
		}
		if !strings.Contains(err.Error(), "already contains") {
			t.Errorf("runInit() error = %q, expected 'already contains'", err.Error())
		}
	})
}

// --- version command tests ---

func TestVersionVars(t *testing.T) {
	// Verify the default values are set.
	if ver == "" {
		t.Error("ver should not be empty")
	}
	if commit == "" {
		t.Error("commit should not be empty")
	}
	if date == "" {
		t.Error("date should not be empty")
	}
}

// --- nopeCmd configuration tests ---

func TestNopeCmdSilenceFlags(t *testing.T) {
	if !nopeCmd.SilenceUsage {
		t.Error("nopeCmd should have SilenceUsage set")
	}
	if !nopeCmd.SilenceErrors {
		t.Error("nopeCmd should have SilenceErrors set")
	}
}

func TestCommitCmdSilenceFlags(t *testing.T) {
	if !commitCmd.SilenceUsage {
		t.Error("commitCmd should have SilenceUsage set")
	}
	if !commitCmd.SilenceErrors {
		t.Error("commitCmd should have SilenceErrors set")
	}
}

// --- printCommandTree tests ---

func TestPrintCommandTree(t *testing.T) {
	// Capture stdout by using a buffer.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printCommandTree(rootCmd, "")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should contain top-level commands.
	if !strings.Contains(output, "jig") {
		t.Error("printCommandTree() output missing 'jig'")
	}
	if !strings.Contains(output, "commit") {
		t.Error("printCommandTree() output missing 'commit'")
	}
	if !strings.Contains(output, "cite") {
		t.Error("printCommandTree() output missing 'cite'")
	}
	if !strings.Contains(output, "nope") {
		t.Error("printCommandTree() output missing 'nope'")
	}

	// Should include subcommands.
	if !strings.Contains(output, "gather") {
		t.Error("printCommandTree() output missing 'gather' subcommand")
	}
	if !strings.Contains(output, "apply") {
		t.Error("printCommandTree() output missing 'apply' subcommand")
	}

	// Should include flag info.
	if !strings.Contains(output, "flags:") {
		t.Error("printCommandTree() output missing 'flags:' section")
	}

	// Should NOT include help-all itself or hidden commands.
	if strings.Contains(output, "help-all") {
		t.Error("printCommandTree() should not include 'help-all'")
	}
}

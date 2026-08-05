package zed

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/toba/jig-go/v4/internal/companion"
)

// InitOpts holds the inputs for zed init.
type InitOpts struct {
	Ext       string // extension repo, e.g. "toba/gozer"
	Tag       string // e.g. "v1.2.3" (empty = latest)
	Repo      string // source repo, e.g. "toba/go-template-lsp" (empty = detect)
	Desc      string // extension description (empty = detect)
	LSPName   string // LSP binary name (empty = source repo name)
	Languages string // comma-separated language list (required)
	DryRun    bool
}

// InitResult describes what was done (or would be done).
type InitResult struct {
	Ext           string `json:"ext"`
	Repo          string `json:"repo"`
	Tag           string `json:"tag"`
	LSPName       string `json:"lsp_name"`
	ExtensionToml string `json:"extension_toml"`
	CargoToml     string `json:"cargo_toml"`
	LibRs         string `json:"lib_rs"`
	BumpScript    string `json:"bump_script"`
	BumpWorkflow  string `json:"bump_workflow"`
	License       string `json:"license"`
	Readme        string `json:"readme"`
	WorkflowJob   string `json:"workflow_job"`
	ExtCreated    bool   `json:"ext_created"`
	ExtPushed     bool   `json:"ext_pushed"`
	WorkflowMod   bool   `json:"workflow_modified"`
}

// RunInit performs the full Zed extension setup workflow.
func RunInit(opts InitOpts) (*InitResult, error) {
	// Step 1: Detect source repo info.
	info, err := companion.DetectRepoInfo(opts.Repo, "nameWithOwner,description")
	if err != nil {
		return nil, fmt.Errorf("detecting repo info: %w", err)
	}

	repo := info.NameWithOwner
	desc := cmp.Or(opts.Desc, info.Description)

	// Derive org and tool name from source repo.
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format %q, expected owner/name", repo)
	}
	org := parts[0]
	tool := parts[1]

	// Derive LSP binary name.
	lspName := cmp.Or(opts.LSPName, tool)

	// Derive extension repo parts.
	extParts := strings.SplitN(opts.Ext, "/", 2)
	if len(extParts) != 2 {
		return nil, fmt.Errorf("invalid extension repo %q, expected owner/name", opts.Ext)
	}
	extRepo := extParts[1]
	extID := extRepo // e.g. "gozer"

	// Extension display name is PascalCase of the ID.
	extName := pascalCase(extID)

	// Step 2: Detect latest release if tag not given.
	tag := opts.Tag
	if tag == "" {
		tag, err = companion.DetectLatestTag(repo)
		if err != nil {
			return nil, fmt.Errorf("detecting latest release: %w", err)
		}
	}

	// Strip v prefix for version fields.
	version := strings.TrimPrefix(tag, "v")

	// Parse languages.
	languages := parseLanguages(opts.Languages)
	if len(languages) == 0 {
		return nil, errors.New("--languages is required")
	}

	// Build params for generation.
	p := ExtensionParams{
		ExtID:     extID,
		ExtName:   extName,
		Version:   version,
		Desc:      desc,
		Org:       org,
		ExtRepo:   extRepo,
		LSPRepo:   repo,
		LSPName:   lspName,
		Languages: languages,
	}

	// Step 3–9: Generate all files.
	extensionToml := GenerateExtensionToml(p)
	cargoToml := GenerateCargoToml(p)
	libRs := GenerateLibRs(p)
	bumpScript := GenerateBumpVersionScript()
	bumpWorkflow := GenerateBumpVersionWorkflow()
	license := GenerateLicense()
	readme := GenerateReadme(p)

	// Step 12: Generate workflow job.
	wp := WorkflowParams{
		Org: org,
		Ext: extRepo,
	}
	workflowJob := GenerateSyncExtensionJob(wp)

	result := &InitResult{
		Ext:           opts.Ext,
		Repo:          repo,
		Tag:           tag,
		LSPName:       lspName,
		ExtensionToml: extensionToml,
		CargoToml:     cargoToml,
		LibRs:         libRs,
		BumpScript:    bumpScript,
		BumpWorkflow:  bumpWorkflow,
		License:       license,
		Readme:        readme,
		WorkflowJob:   workflowJob,
	}

	if opts.DryRun {
		return result, nil
	}

	// Step 10: Create extension repo on GitHub.
	if err := createExtRepo(opts.Ext, desc); err != nil {
		return nil, fmt.Errorf("creating extension repo: %w", err)
	}
	result.ExtCreated = true

	// Step 11: Push initial content.
	files := map[string]struct {
		content    string
		executable bool
	}{
		"extension.toml":                     {extensionToml, false},
		"Cargo.toml":                         {cargoToml, false},
		"src/lib.rs":                         {libRs, false},
		"scripts/bump-version.sh":            {bumpScript, true},
		".github/workflows/bump-version.yml": {bumpWorkflow, false},
		"LICENSE":                            {license, false},
		"README.md":                          {readme, false},
	}
	if err := pushInitialContent(opts.Ext, files); err != nil {
		return nil, fmt.Errorf("pushing initial content: %w", err)
	}
	result.ExtPushed = true

	// Step 12: Inject sync-extension job into release.yml.
	workflowPath := companion.WorkflowPath
	if err := injectWorkflow(workflowPath, wp); err != nil {
		// Non-fatal — print warning but don't fail.
		fmt.Fprintf(os.Stderr, "Warning: could not inject workflow job: %v\n", err)
	} else {
		result.WorkflowMod = true
	}

	return result, nil
}

func createExtRepo(ext, desc string) error {
	if companion.RepoExists(ext) {
		return nil
	}
	cmd := exec.Command("gh", "repo", "create", ext, "--public", //nolint:gosec // gh CLI wrapper
		"--description", desc)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func pushInitialContent(ext string, files map[string]struct {
	content    string
	executable bool
}) error {
	tmp, err := os.MkdirTemp("", "zed-init-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmp) //nolint:errcheck // best-effort cleanup

	// Clone the empty repo.
	cmd := exec.Command("gh", "repo", "clone", ext, tmp) //nolint:gosec // gh CLI wrapper
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cloning: %s", strings.TrimSpace(string(out)))
	}

	// Write all files.
	for path, f := range files {
		fullPath := filepath.Join(tmp, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		perm := os.FileMode(0o644)
		if f.executable {
			perm = 0o755
		}
		if err := os.WriteFile(fullPath, []byte(f.content), perm); err != nil {
			return err
		}
	}

	// Commit and push.
	cmds := [][]string{
		{"git", "-C", tmp, "add", "."},
		{"git", "-C", tmp, "commit", "-m", "initial extension scaffold"},
		{"git", "-C", tmp, "push"},
	}
	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...) //nolint:gosec // gh CLI wrapper
		if out, err := c.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func injectWorkflow(path string, p WorkflowParams) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	modified, err := InjectSyncExtensionJob(string(data), p)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(modified), 0o644)
}

// parseLanguages splits a comma-separated string into trimmed language names.
func parseLanguages(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

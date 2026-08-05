package brew

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/toba/jig-go/v4/internal/companion"
)

// InitOpts holds the inputs for brew init.
type InitOpts struct {
	Tap     string // e.g. "toba/homebrew-tap"
	Tag     string // e.g. "v1.2.3" (empty = latest)
	Repo    string // e.g. "toba/todo" (empty = detect)
	Desc    string // formula description (empty = detect)
	License string // license identifier (empty = detect)
	DryRun  bool
}

// InitResult describes what was done (or would be done).
type InitResult struct {
	Tap         string `json:"tap"`
	Repo        string `json:"repo"`
	Tool        string `json:"tool"`
	Tag         string `json:"tag"`
	Asset       string `json:"asset"`
	SHA256      string `json:"sha256"`
	Desc        string `json:"desc"`
	License     string `json:"license"`
	Formula     string `json:"formula"`
	WorkflowJob string `json:"workflow_job"`
	TapPushed   bool   `json:"tap_pushed"`
	WorkflowMod bool   `json:"workflow_modified"`
}

// RunInit performs the full brew tap setup workflow.
// The tap repo is expected to already exist (shared tap model).
func RunInit(opts InitOpts) (*InitResult, error) {
	// Step 1: Detect source repo info.
	info, err := companion.DetectRepoInfo(opts.Repo, "nameWithOwner,description,licenseInfo")
	if err != nil {
		return nil, fmt.Errorf("detecting repo info: %w", err)
	}

	repo := info.NameWithOwner
	desc := cmp.Or(opts.Desc, info.Description)
	license := cmp.Or(opts.License, info.LicenseInfo.SpdxID, "NOASSERTION")

	// Derive tool name and org from repo.
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format %q, expected owner/name", repo)
	}
	org := parts[0]
	tool := parts[1]

	// Step 2: Detect latest release if tag not given.
	tag := opts.Tag
	if tag == "" {
		tag, err = companion.DetectLatestTag(repo)
		if err != nil {
			return nil, fmt.Errorf("detecting latest release: %w", err)
		}
	}

	// Step 3: Derive asset name.
	asset := tool + "_darwin_arm64.tar.gz"

	// Step 4: Resolve SHA256.
	sha, err := ResolveSHA256(repo, tag, asset)
	if err != nil {
		return nil, fmt.Errorf("resolving SHA256 for %s: %w", asset, err)
	}

	// Step 5: Generate formula.
	formulaContent := GenerateFormula(FormulaParams{
		Tool:    tool,
		Desc:    desc,
		Repo:    repo,
		Tag:     tag,
		Asset:   asset,
		SHA256:  sha,
		License: license,
	})

	// Step 6: Generate workflow job.
	wpParams := WorkflowParams{
		Tool:    tool,
		Org:     org,
		Tap:     opts.Tap,
		Desc:    desc,
		License: license,
		Asset:   asset,
	}
	workflowJob := GenerateWorkflowJob(wpParams)

	result := &InitResult{
		Tap:         opts.Tap,
		Repo:        repo,
		Tool:        tool,
		Tag:         tag,
		Asset:       asset,
		SHA256:      sha,
		Desc:        desc,
		License:     license,
		Formula:     formulaContent,
		WorkflowJob: workflowJob,
	}

	if opts.DryRun {
		return result, nil
	}

	// Step 7: Push formula to shared tap repo.
	if err := pushFormula(opts.Tap, tool, formulaContent); err != nil {
		return nil, fmt.Errorf("pushing formula to tap: %w", err)
	}
	result.TapPushed = true

	// Step 8: Inject workflow job into release.yml.
	workflowPath := companion.WorkflowPath
	if err := injectWorkflow(workflowPath, wpParams); err != nil {
		// Non-fatal — print warning but don't fail.
		fmt.Fprintf(os.Stderr, "Warning: could not inject workflow job: %v\n", err)
	} else {
		result.WorkflowMod = true
	}

	return result, nil
}

// pushFormula clones the shared tap repo, adds or updates a formula, and pushes.
func pushFormula(tap, tool, formula string) error {
	tmp, err := os.MkdirTemp("", "brew-formula-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmp) //nolint:errcheck // best-effort cleanup

	cmd := exec.Command("gh", "repo", "clone", tap, tmp) //nolint:gosec // gh CLI wrapper
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cloning: %s", strings.TrimSpace(string(out)))
	}

	formulaDir := filepath.Join(tmp, "Formula")
	if err := os.MkdirAll(formulaDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(formulaDir, tool+".rb"), []byte(formula), 0o644); err != nil {
		return err
	}

	cmds := [][]string{
		{"git", "-C", tmp, "add", "Formula/" + tool + ".rb"},
		{"git", "-C", tmp, "commit", "-m", "add " + tool + " formula"},
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

	modified, err := InjectWorkflowJob(string(data), p)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(modified), 0o644)
}

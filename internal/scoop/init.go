package scoop

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/toba/jig-go/v4/internal/brew"
	"github.com/toba/jig-go/v4/internal/companion"
)

// InitOpts holds the inputs for scoop init.
type InitOpts struct {
	Bucket  string // e.g. "toba/scoop-bucket"
	Tag     string // e.g. "v1.2.3" (empty = latest)
	Repo    string // e.g. "toba/jig" (empty = detect)
	Desc    string // manifest description (empty = detect)
	License string // license identifier (empty = detect)
	DryRun  bool
}

// InitResult describes what was done (or would be done).
type InitResult struct {
	Bucket       string `json:"bucket"`
	Repo         string `json:"repo"`
	Tool         string `json:"tool"`
	Tag          string `json:"tag"`
	AssetAMD64   string `json:"asset_amd64"`
	AssetARM64   string `json:"asset_arm64"`
	SHA256AMD64  string `json:"sha256_amd64"`
	SHA256ARM64  string `json:"sha256_arm64"`
	Desc         string `json:"desc"`
	License      string `json:"license"`
	Manifest     string `json:"manifest"`
	WorkflowJob  string `json:"workflow_job"`
	BucketPushed bool   `json:"bucket_pushed"`
	WorkflowMod  bool   `json:"workflow_modified"`
}

// RunInit performs the full scoop bucket setup workflow.
// The bucket repo is expected to already exist (shared bucket model).
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

	// Step 3: Derive asset names.
	assetAMD64 := tool + "_windows_amd64.zip"
	assetARM64 := tool + "_windows_arm64.zip"

	// Step 4: Resolve SHA256 for amd64 (required) and arm64 (optional).
	shaAMD64, err := brew.ResolveSHA256(repo, tag, assetAMD64)
	if err != nil {
		return nil, fmt.Errorf("resolving SHA256 for %s: %w", assetAMD64, err)
	}
	shaARM64, _ := brew.ResolveSHA256(repo, tag, assetARM64)

	// Step 5: Generate manifest.
	manifestContent := GenerateManifest(ManifestParams{
		Tool:        tool,
		Desc:        desc,
		Homepage:    "https://github.com/" + repo,
		License:     license,
		Tag:         tag,
		Repo:        repo,
		SHA256AMD64: shaAMD64,
		SHA256ARM64: shaARM64,
	})

	// Step 6: Generate workflow job.
	wpParams := WorkflowParams{
		Tool:    tool,
		Org:     org,
		Bucket:  opts.Bucket,
		Desc:    desc,
		License: license,
	}
	workflowJob := GenerateWorkflowJob(wpParams)

	result := &InitResult{
		Bucket:      opts.Bucket,
		Repo:        repo,
		Tool:        tool,
		Tag:         tag,
		AssetAMD64:  assetAMD64,
		AssetARM64:  assetARM64,
		SHA256AMD64: shaAMD64,
		SHA256ARM64: shaARM64,
		Desc:        desc,
		License:     license,
		Manifest:    manifestContent,
		WorkflowJob: workflowJob,
	}

	if opts.DryRun {
		return result, nil
	}

	// Step 7: Push manifest to shared bucket repo.
	if err := pushManifest(opts.Bucket, tool, manifestContent); err != nil {
		return nil, fmt.Errorf("pushing manifest to bucket: %w", err)
	}
	result.BucketPushed = true

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

// pushManifest clones the shared bucket repo, adds or updates a manifest, and pushes.
// Manifests are placed at repo root (e.g. jig.json), matching the charmbracelet convention.
func pushManifest(bucket, tool, manifest string) error {
	tmp, err := os.MkdirTemp("", "scoop-manifest-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmp) //nolint:errcheck // best-effort cleanup

	cmd := exec.Command("gh", "repo", "clone", bucket, tmp) //nolint:gosec // gh CLI wrapper
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cloning: %s", strings.TrimSpace(string(out)))
	}

	if err := os.WriteFile(tmp+"/"+tool+".json", []byte(manifest), 0o644); err != nil {
		return err
	}

	cmds := [][]string{
		{"git", "-C", tmp, "add", tool + ".json"},
		{"git", "-C", tmp, "commit", "-m", "add " + tool + " manifest"},
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

package zed

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/toba/jig-go/v4/internal/companion"
	"golang.org/x/sync/errgroup"
)

// DoctorOpts holds the inputs for zed doctor.
type DoctorOpts struct {
	Ext  string // e.g. "toba/gozer"
	Repo string // e.g. "toba/go-template-lsp"
}

// RunDoctor validates the Zed extension companion setup is healthy.
// Returns 0 on success, 1 on any FAIL.
func RunDoctor(opts DoctorOpts) int {
	ok := true

	// Derive org and extension name.
	extParts := strings.SplitN(opts.Ext, "/", 2)
	if len(extParts) != 2 {
		fmt.Fprintf(os.Stderr, "FAIL: invalid extension repo format: %s\n", opts.Ext)
		return 1
	}
	org := extParts[0]
	extName := extParts[1]

	// 1. zed_extension configured
	if opts.Ext == "" {
		fmt.Fprintf(os.Stderr, "FAIL: zed_extension not configured in .jig.yaml\n")
		return 1
	}
	fmt.Fprintf(os.Stderr, "OK:   zed_extension configured: %s\n", opts.Ext)

	// Checks 2-6: independent gh API calls — run concurrently.
	type checkResult struct {
		msg    string
		passed bool
	}
	results := make([]checkResult, 5)
	var mu sync.Mutex
	setResult := func(i int, msg string, passed bool) {
		mu.Lock()
		results[i] = checkResult{msg: msg, passed: passed}
		mu.Unlock()
	}

	g := new(errgroup.Group)
	g.SetLimit(5)

	// 2. extension repo exists on GitHub
	g.Go(func() error {
		cmd := exec.Command("gh", "repo", "view", opts.Ext) //nolint:gosec // gh CLI wrapper
		if out, err := cmd.CombinedOutput(); err != nil {
			setResult(0, fmt.Sprintf("FAIL: extension repo %s not found on GitHub: %s", opts.Ext, strings.TrimSpace(string(out))), false)
		} else {
			setResult(0, "OK:   extension repo exists: "+opts.Ext, true)
		}
		return nil
	})

	// 3. extension.toml exists in extension repo
	g.Go(func() error {
		cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/contents/extension.toml", opts.Ext)) //nolint:gosec // gh CLI wrapper
		if out, err := cmd.CombinedOutput(); err != nil {
			setResult(1, fmt.Sprintf("FAIL: extension.toml not found in %s: %s", opts.Ext, strings.TrimSpace(string(out))), false)
		} else {
			setResult(1, "OK:   extension.toml exists in "+opts.Ext, true)
		}
		return nil
	})

	// 4. Cargo.toml exists in extension repo
	g.Go(func() error {
		cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/contents/Cargo.toml", opts.Ext)) //nolint:gosec // gh CLI wrapper
		if out, err := cmd.CombinedOutput(); err != nil {
			setResult(2, fmt.Sprintf("FAIL: Cargo.toml not found in %s: %s", opts.Ext, strings.TrimSpace(string(out))), false)
		} else {
			setResult(2, "OK:   Cargo.toml exists in "+opts.Ext, true)
		}
		return nil
	})

	// 5. bump-version.yml workflow exists in extension repo
	g.Go(func() error {
		cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/contents/.github/workflows/bump-version.yml", opts.Ext)) //nolint:gosec // gh CLI wrapper
		if out, err := cmd.CombinedOutput(); err != nil {
			setResult(3, fmt.Sprintf("FAIL: bump-version.yml workflow not found in %s: %s", opts.Ext, strings.TrimSpace(string(out))), false)
		} else {
			setResult(3, "OK:   bump-version.yml workflow exists in "+opts.Ext, true)
		}
		return nil
	})

	// 6. scripts/bump-version.sh exists in extension repo
	g.Go(func() error {
		cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/contents/scripts/bump-version.sh", opts.Ext)) //nolint:gosec // gh CLI wrapper
		if out, err := cmd.CombinedOutput(); err != nil {
			setResult(4, fmt.Sprintf("FAIL: scripts/bump-version.sh not found in %s: %s", opts.Ext, strings.TrimSpace(string(out))), false)
		} else {
			setResult(4, "OK:   scripts/bump-version.sh exists in "+opts.Ext, true)
		}
		return nil
	})

	_ = g.Wait()

	// Display results in original order.
	for _, r := range results {
		fmt.Fprintln(os.Stderr, r.msg)
		if !r.passed {
			ok = false
		}
	}

	// 7. source repo has releases
	tag := ""
	cmd := exec.Command("gh", "release", "list", "--repo", opts.Repo, "--limit", "1", "--json", "tagName", "--jq", ".[0].tagName") //nolint:gosec // gh CLI wrapper
	if out, err := cmd.Output(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: no releases found for %s\n", opts.Repo)
		ok = false
	} else {
		tag = strings.TrimSpace(string(out))
		if tag == "" {
			fmt.Fprintf(os.Stderr, "FAIL: no releases found for %s\n", opts.Repo)
			ok = false
		} else {
			fmt.Fprintf(os.Stderr, "OK:   latest release: %s\n", tag)
		}
	}

	// 8. extension repo has a matching tag
	if tag != "" {
		cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/git/ref/tags/%s", opts.Ext, tag)) //nolint:gosec // gh CLI wrapper
		if _, err := cmd.Output(); err != nil {
			fmt.Fprintf(os.Stderr, "WARN: extension repo %s missing tag %s (may not have synced yet)\n", opts.Ext, tag)
		} else {
			fmt.Fprintf(os.Stderr, "OK:   extension repo has tag: %s\n", tag)
		}
	}

	// 9. latest release has platform assets (goreleaser output)
	if tag != "" {
		repoParts := strings.SplitN(opts.Repo, "/", 2)
		tool := ""
		if len(repoParts) == 2 {
			tool = repoParts[1]
		}
		if tool != "" {
			checkReleaseAssets(opts.Repo, tag, tool, &ok)
		}
	}

	// 10. release workflow exists locally
	workflowPath := companion.WorkflowPath
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %s not found\n", workflowPath)
		ok = false
	} else {
		fmt.Fprintf(os.Stderr, "OK:   workflow exists: %s\n", workflowPath)

		workflowStr := string(content)

		// 11. workflow has sync-extension job
		if !strings.Contains(workflowStr, "sync-extension:") {
			fmt.Fprintf(os.Stderr, "FAIL: workflow missing sync-extension job\n")
			ok = false
		} else {
			fmt.Fprintf(os.Stderr, "OK:   workflow has sync-extension job\n")
		}

		// 12. workflow references correct extension repo
		if !strings.Contains(workflowStr, fmt.Sprintf("%s/%s", org, extName)) {
			fmt.Fprintf(os.Stderr, "FAIL: workflow does not reference %s/%s\n", org, extName)
			ok = false
		} else {
			fmt.Fprintf(os.Stderr, "OK:   workflow references %s/%s\n", org, extName)
		}

		// 13. workflow references EXTENSION_PAT secret
		if !strings.Contains(workflowStr, "EXTENSION_PAT") {
			fmt.Fprintf(os.Stderr, "FAIL: workflow missing EXTENSION_PAT secret reference\n")
			ok = false
		} else {
			fmt.Fprintf(os.Stderr, "OK:   workflow references EXTENSION_PAT secret\n")
		}
	}

	// 14. .goreleaser.yaml exists (extensions need release assets)
	ok = checkGoreleaserExists() && ok

	if !ok {
		return 1
	}
	return 0
}

// checkReleaseAssets verifies the latest release has platform-specific assets
// that the extension's lib.rs will download.
func checkReleaseAssets(repo, tag, tool string, ok *bool) {
	cmd := exec.Command("gh", "release", "view", tag, "--repo", repo, "--json", "assets") //nolint:gosec // gh CLI wrapper
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: could not fetch release %s assets\n", tag)
		*ok = false
		return
	}

	var release struct {
		Assets []struct {
			Name string `json:"name"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(out, &release); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: could not parse release assets: %v\n", err)
		*ok = false
		return
	}

	// Check for at least one platform asset matching the tool_os_arch pattern.
	expectedDarwin := tool + "_darwin_arm64.tar.gz"
	expectedLinux := tool + "_linux_amd64.tar.gz"
	hasDarwin, hasLinux := false, false
	for _, a := range release.Assets {
		if a.Name == expectedDarwin {
			hasDarwin = true
		}
		if a.Name == expectedLinux {
			hasLinux = true
		}
	}

	if !hasDarwin {
		fmt.Fprintf(os.Stderr, "FAIL: release %s missing asset %s\n", tag, expectedDarwin)
		*ok = false
	} else {
		fmt.Fprintf(os.Stderr, "OK:   release has asset: %s\n", expectedDarwin)
	}

	if !hasLinux {
		fmt.Fprintf(os.Stderr, "WARN: release %s missing asset %s (Zed extensions support Linux too)\n", tag, expectedLinux)
	} else {
		fmt.Fprintf(os.Stderr, "OK:   release has asset: %s\n", expectedLinux)
	}
}

// checkGoreleaserExists verifies .goreleaser.yaml or .goreleaser.yml exists.
func checkGoreleaserExists() bool {
	_, name, found := companion.CheckGoreleaserExists()
	if !found {
		fmt.Fprintf(os.Stderr, "FAIL: .goreleaser.yaml not found\n")
		return false
	}
	fmt.Fprintf(os.Stderr, "OK:   %s exists\n", name)
	return true
}

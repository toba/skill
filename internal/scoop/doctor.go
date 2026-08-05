package scoop

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/toba/jig-go/v4/internal/companion"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// DoctorOpts holds the inputs for scoop doctor.
type DoctorOpts struct {
	Bucket string // e.g. "toba/scoop-jig"
	Repo   string // e.g. "toba/jig"
	Tool   string // e.g. "jig"
}

// RunDoctor validates the scoop bucket chain is healthy.
// Returns 0 on success, 1 on any FAIL.
func RunDoctor(opts DoctorOpts) int {
	ok := true

	// 1. scoop package configured
	if opts.Bucket == "" {
		fmt.Fprintf(os.Stderr, "FAIL: scoop not configured (add to packages in .jig.yaml)\n")
		return 1
	}
	fmt.Fprintf(os.Stderr, "OK:   scoop configured: %s\n", opts.Bucket)

	// 2. .goreleaser.yaml has windows + amd64 builds and zip format (if present)
	if _, _, found := companion.CheckGoreleaserExists(); found {
		ok = checkGoreleaserScoop(opts.Tool) && ok
	} else {
		fmt.Fprintf(os.Stderr, "WARN: .goreleaser.yaml not found (manual builds assumed)\n")
	}

	// Checks 3, 4, 5: independent gh calls — run concurrently.
	type checkResult struct {
		msg    string
		passed bool
	}
	results := make([]checkResult, 3)
	var mu sync.Mutex
	setResult := func(i int, msg string, passed bool) {
		mu.Lock()
		results[i] = checkResult{msg: msg, passed: passed}
		mu.Unlock()
	}

	tag := ""
	g := new(errgroup.Group)
	g.SetLimit(3)

	// 3. bucket repo exists on GitHub
	g.Go(func() error {
		cmd := exec.Command("gh", "repo", "view", opts.Bucket) //nolint:gosec // gh CLI wrapper
		if out, err := cmd.CombinedOutput(); err != nil {
			setResult(0, fmt.Sprintf("FAIL: bucket repo %s not found on GitHub: %s", opts.Bucket, strings.TrimSpace(string(out))), false)
		} else {
			setResult(0, "OK:   bucket repo exists: "+opts.Bucket, true)
		}
		return nil
	})

	// 4. manifest exists in bucket (at repo root)
	g.Go(func() error {
		manifestPath := fmt.Sprintf("repos/%s/contents/%s.json", opts.Bucket, opts.Tool)
		cmd := exec.Command("gh", "api", manifestPath) //nolint:gosec // gh CLI wrapper
		if out, err := cmd.CombinedOutput(); err != nil {
			setResult(1, fmt.Sprintf("FAIL: manifest not found at %s.json in %s: %s", opts.Tool, opts.Bucket, strings.TrimSpace(string(out))), false)
		} else {
			setResult(1, fmt.Sprintf("OK:   manifest exists: %s.json", opts.Tool), true)
		}
		return nil
	})

	// 5. source repo has releases
	g.Go(func() error {
		cmd := exec.Command("gh", "release", "list", "--repo", opts.Repo, "--limit", "1", "--json", "tagName", "--jq", ".[0].tagName") //nolint:gosec // gh CLI wrapper
		if out, err := cmd.Output(); err != nil {
			setResult(2, "FAIL: no releases found for "+opts.Repo, false)
		} else {
			t := strings.TrimSpace(string(out))
			if t == "" {
				setResult(2, "FAIL: no releases found for "+opts.Repo, false)
			} else {
				mu.Lock()
				tag = t
				mu.Unlock()
				setResult(2, "OK:   latest release: "+t, true)
			}
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

	// 6. latest release has windows amd64 asset (depends on tag from check 5)
	expectedAsset := opts.Tool + "_windows_amd64.zip"
	if tag != "" {
		cmd := exec.Command("gh", "release", "view", tag, "--repo", opts.Repo, "--json", "assets") //nolint:gosec // gh CLI wrapper
		if out, err := cmd.Output(); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: could not fetch release %s assets\n", tag)
			ok = false
		} else {
			var release struct {
				Assets []struct {
					Name string `json:"name"`
				} `json:"assets"`
			}
			if err := json.Unmarshal(out, &release); err != nil {
				fmt.Fprintf(os.Stderr, "FAIL: could not parse release assets: %v\n", err)
				ok = false
			} else {
				foundAsset := false
				hasChecksums := false
				for _, a := range release.Assets {
					if a.Name == expectedAsset {
						foundAsset = true
					}
					if a.Name == "checksums.txt" {
						hasChecksums = true
					}
				}
				if !foundAsset {
					fmt.Fprintf(os.Stderr, "FAIL: release %s missing asset %s\n", tag, expectedAsset)
					ok = false
				} else {
					fmt.Fprintf(os.Stderr, "OK:   release has asset: %s\n", expectedAsset)
				}
				// 7. checksum verification
				if !hasChecksums {
					fmt.Fprintf(os.Stderr, "FAIL: release %s missing checksums.txt\n", tag)
					ok = false
				} else {
					fmt.Fprintf(os.Stderr, "OK:   release has checksums.txt\n")
				}
			}
		}
	}

	// 8. release workflow exists locally
	workflowPath := companion.WorkflowPath
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %s not found\n", workflowPath)
		ok = false
	} else {
		fmt.Fprintf(os.Stderr, "OK:   workflow exists: %s\n", workflowPath)

		workflowStr := string(content)

		// 9. workflow has update-scoop job
		if !strings.Contains(workflowStr, "update-scoop:") {
			fmt.Fprintf(os.Stderr, "FAIL: workflow missing update-scoop job\n")
			ok = false
		} else {
			fmt.Fprintf(os.Stderr, "OK:   workflow has update-scoop job\n")
		}

		// 10. workflow references correct bucket repo
		bucketRepo := filepath.Base(opts.Bucket) // e.g. "scoop-jig"
		if !strings.Contains(workflowStr, bucketRepo) {
			fmt.Fprintf(os.Stderr, "FAIL: workflow does not reference %s\n", bucketRepo)
			ok = false
		} else {
			fmt.Fprintf(os.Stderr, "OK:   workflow references %s\n", bucketRepo)
		}
	}

	if !ok {
		return 1
	}
	return 0
}

// goreleaserConfig is the subset of .goreleaser.yaml we validate for scoop.
type goreleaserConfig struct {
	Builds []struct {
		Goos   []string `yaml:"goos"`
		Goarch []string `yaml:"goarch"`
	} `yaml:"builds"`
	Archives []struct {
		Format          string   `yaml:"format"`
		Formats         []string `yaml:"formats"`
		FormatOverrides []struct {
			Goos    string   `yaml:"goos"`
			Format  string   `yaml:"format"`
			Formats []string `yaml:"formats"`
		} `yaml:"format_overrides"`
	} `yaml:"archives"`
}

// checkGoreleaserScoop validates .goreleaser.yaml is present and correctly
// configured for scoop distribution. Returns true if all checks pass.
func checkGoreleaserScoop(_ string) bool {
	ok := true

	data, _, found := companion.CheckGoreleaserExists()
	if !found {
		fmt.Fprintf(os.Stderr, "FAIL: .goreleaser.yaml not found\n")
		return false
	}

	var cfg goreleaserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: .goreleaser.yaml parse error: %v\n", err)
		return false
	}

	// Check builds include windows + amd64.
	hasWindows, hasAMD64 := false, false
	for _, b := range cfg.Builds {
		for _, goos := range b.Goos {
			if goos == "windows" {
				hasWindows = true
			}
		}
		for _, arch := range b.Goarch {
			if arch == "amd64" {
				hasAMD64 = true
			}
		}
	}
	if !hasWindows {
		fmt.Fprintf(os.Stderr, "FAIL: goreleaser builds missing goos: windows\n")
		ok = false
	} else if !hasAMD64 {
		fmt.Fprintf(os.Stderr, "FAIL: goreleaser builds missing goarch: amd64\n")
		ok = false
	} else {
		fmt.Fprintf(os.Stderr, "OK:   goreleaser builds windows/amd64\n")
	}

	// Check archive produces zip for windows (either default or format_overrides).
	if len(cfg.Archives) > 0 {
		a := cfg.Archives[0]
		hasZip := a.Format == "zip"
		for _, f := range a.Formats {
			if f == "zip" {
				hasZip = true
			}
		}
		for _, o := range a.FormatOverrides {
			if o.Goos == "windows" {
				if o.Format == "zip" {
					hasZip = true
				}
				for _, f := range o.Formats {
					if f == "zip" {
						hasZip = true
					}
				}
			}
		}
		if !hasZip {
			fmt.Fprintf(os.Stderr, "FAIL: goreleaser archive format does not include zip for windows\n")
			ok = false
		} else {
			fmt.Fprintf(os.Stderr, "OK:   goreleaser archive produces zip for windows\n")
		}
	}

	return ok
}

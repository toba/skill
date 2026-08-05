package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/toba/jig/internal/config"
	"github.com/toba/jig/internal/github"
)

var markCmd = &cobra.Command{
	Use:   "mark [source]",
	Short: "Record cited repositories as reviewed up to their current HEAD",
	Long: "Advance last_checked_sha/last_checked_date (and last_checked_tag for " +
		"release-tracked sources) to the current HEAD, so the next review only " +
		"shows newer changes. Optionally limit to a single source. Run this after " +
		"reviewing with `jigo cite review` (which is read-only and never advances).",
	Args: cobra.MaximumNArgs(1),
	RunE: runMark,
}

func init() {
	citeCmd.AddCommand(markCmd)
}

// markSource determines the current marker for a source without modifying it.
// For release-tracked sources it returns the latest release tag and that tag's
// HEAD SHA; otherwise it returns the branch HEAD SHA with an empty tag.
func markSource(client github.Client, src config.Source) (tag, sha string, err error) {
	if src.TracksReleases() {
		latest, err := client.GetLatestRelease(src.Repo)
		if err != nil {
			return "", "", fmt.Errorf("fetching latest release: %w", err)
		}
		headSHA, err := client.GetHeadSHA(src.Repo, latest.TagName)
		if err != nil {
			return "", "", fmt.Errorf("getting tag SHA: %w", err)
		}
		return latest.TagName, headSHA, nil
	}

	headSHA, err := client.GetHeadSHA(src.Repo, src.Branch)
	if err != nil {
		return "", "", fmt.Errorf("getting HEAD: %w", err)
	}
	return "", headSHA, nil
}

type markResult struct {
	source config.Source
	tag    string
	sha    string
}

func runMark(cmd *cobra.Command, args []string) error {
	sources := *cfg
	if len(args) > 0 {
		src := config.FindSource(cfg, args[0])
		if src == nil {
			return fmt.Errorf("source %q not found in config", args[0])
		}
		sources = []config.Source{*src}
	}

	client := github.NewClient()
	results := make([]*markResult, len(sources))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, src := range sources {
		wg.Go(func() {
			tag, sha, err := markSource(client, src)
			if err != nil {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "warning: %s: %v\n", src.Repo, err)
				mu.Unlock()
				return
			}
			results[i] = &markResult{source: src, tag: tag, sha: sha}
		})
	}
	wg.Wait()

	dirty := false
	for _, r := range results {
		if r == nil {
			continue // error path — markSource failed
		}
		origSrc := config.FindSource(cfg, r.source.Repo)
		if origSrc == nil {
			continue
		}
		if r.tag != "" {
			config.MarkSourceRelease(origSrc, r.tag, r.sha)
		} else {
			config.MarkSource(origSrc, r.sha)
		}
		dirty = true
	}
	if dirty {
		if err := config.Save(cfgDoc, cfg); err != nil {
			return fmt.Errorf("saving last_checked: %w", err)
		}
	}

	return renderMarkResults(os.Stdout, results)
}

// markJSON is the machine-readable shape for `jigo cite mark --json`.
type markJSON struct {
	Repo string `json:"repo"`
	Tag  string `json:"tag,omitempty"`
	SHA  string `json:"sha"`
}

func renderMarkResults(w io.Writer, results []*markResult) error {
	if jsonOut {
		out := make([]markJSON, 0, len(results))
		for _, r := range results {
			if r == nil {
				continue
			}
			out = append(out, markJSON{Repo: r.source.Repo, Tag: r.tag, SHA: r.sha})
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	marked := 0
	for _, r := range results {
		if r == nil {
			continue
		}
		marked++
		ref := shortSHA(r.sha)
		if r.tag != "" {
			ref = fmt.Sprintf("%s (%s)", r.tag, ref)
		}
		fmt.Fprintf(w, "Marked %s reviewed at %s\n", r.source.Repo, ref)
	}
	if marked == 0 {
		fmt.Fprintln(w, "Nothing marked.")
	}
	return nil
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

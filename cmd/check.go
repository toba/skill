package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/toba/jig-go/v4/internal/classify"
	"github.com/toba/jig-go/v4/internal/config"
	"github.com/toba/jig-go/v4/internal/display"
	"github.com/toba/jig-go/v4/internal/github"
	"golang.org/x/sync/errgroup"
)

// commitDateFormat is the date layout used when rendering commit dates.
const commitDateFormat = "2006-01-02"

var reviewWithDiffs bool

var reviewCmd = &cobra.Command{
	Use:     "review [source]",
	Aliases: []string{"check"},
	Short:   "Review cited repositories for changes grouped by relevance",
	Long:    "Review cited repositories for changes since last check. Optionally filter to a specific source.",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runCheck,
}

func init() {
	reviewCmd.Flags().BoolVar(&reviewWithDiffs, "with-diffs", false,
		"include unified diff per changed file in JSON output (capped per file)")
	citeCmd.AddCommand(reviewCmd)
}

// Per-file diff caps for --with-diffs. Files exceeding either bound are
// truncated; files larger than diffSizeCap (50 KB) are skipped entirely.
const (
	diffLineCap = 500
	diffSizeCap = 50 * 1024
)

// capDiff applies the per-file diff caps. Returns the (possibly trimmed)
// diff and flags indicating truncated/skipped state.
func capDiff(patch string) (out string, truncated, skipped bool) {
	if patch == "" {
		return "", false, false
	}
	if len(patch) > diffSizeCap {
		return "", false, true
	}
	lineCount := strings.Count(patch, "\n") + 1
	if lineCount > diffLineCap {
		// Keep the first diffLineCap lines.
		idx := 0
		for n := 0; n < diffLineCap && idx < len(patch); n++ {
			next := strings.IndexByte(patch[idx:], '\n')
			if next < 0 {
				idx = len(patch)
				break
			}
			idx += next + 1
		}
		return patch[:idx], true, false
	}
	return patch, false, false
}

type checkResult struct {
	display display.SourceResult
}

func runCheck(cmd *cobra.Command, args []string) error {
	sources := *cfg
	if len(args) > 0 {
		src := config.FindSource(cfg, args[0])
		if src == nil {
			return fmt.Errorf("source %q not found in config", args[0])
		}
		sources = []config.Source{*src}
	}

	client := github.NewClient()
	results := make([]checkResult, len(sources))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, src := range sources {
		wg.Go(func() {
			if src.TracksReleases() {
				result, _, _, err := checkSourceReleases(client, src)
				if err != nil {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", src.Repo, err)
					mu.Unlock()
					results[i] = checkResult{display: display.SourceResult{Source: src}}
					return
				}
				results[i] = checkResult{display: *result}
			} else {
				result, _, err := checkSource(client, src)
				if err != nil {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", src.Repo, err)
					mu.Unlock()
					results[i] = checkResult{display: display.SourceResult{Source: src}}
					return
				}
				results[i] = checkResult{display: *result}
			}
		})
	}
	wg.Wait()

	// review is read-only: it never advances last_checked. Use `jigo cite mark`
	// to record what you've reviewed. This keeps review idempotent so an agent
	// can re-run it (or recover from lost output) without losing the changes.

	displayResults := make([]display.SourceResult, len(results))
	for i, r := range results {
		displayResults[i] = r.display
	}

	if jsonOut {
		return display.RenderJSON(os.Stdout, displayResults)
	}
	display.RenderText(os.Stdout, displayResults)
	return nil
}

func checkSource(client github.Client, src config.Source) (*display.SourceResult, string, error) {
	result := &display.SourceResult{Source: src}

	var commits []github.Commit
	var aggregateFiles []github.File
	newestFirst := true // List Commits API returns newest-first; Compare API returns oldest-first.

	if src.LastCheckedSHA == "" {
		// First run: fetch last 30 commits.
		var err error
		commits, err = client.GetCommits(src.Repo, src.Branch, 30)
		if err != nil {
			return nil, "", fmt.Errorf("fetching commits: %w", err)
		}

		// Fetch file details for the most recent 5 commits in parallel.
		aggregateFiles = fetchCommitDetails(client, src.Repo, commits, min(5, len(commits)))
	} else {
		// Compare since last checked SHA.
		cmp, err := client.Compare(src.Repo, src.LastCheckedSHA, src.Branch)
		if err != nil {
			// Fallback: force-push scenario (404 from compare).
			if strings.Contains(err.Error(), "404") && src.LastCheckedDate != "" {
				commits, err = client.GetCommitsSince(src.Repo, src.Branch, src.LastCheckedDate, 100)
				if err != nil {
					return nil, "", fmt.Errorf("fetching commits since date: %w", err)
				}
				// Fetch file details for up to 5 commits in parallel.
				aggregateFiles = fetchCommitDetails(client, src.Repo, commits, min(5, len(commits)))
			} else {
				return nil, "", fmt.Errorf("comparing: %w", err)
			}
		} else {
			commits = cmp.Commits
			aggregateFiles = cmp.Files
			newestFirst = false // Compare API returns oldest-first.
		}
	}

	if len(commits) == 0 {
		// No new commits — still update the marker to record the check.
		headSHA, err := client.GetHeadSHA(src.Repo, src.Branch)
		if err != nil {
			return nil, "", fmt.Errorf("getting HEAD: %w", err)
		}
		return result, headSHA, nil
	}

	// The most recent commit SHA becomes the new last_checked reference.
	var headSHA string
	if newestFirst {
		headSHA = commits[0].SHA
	} else {
		headSHA = commits[len(commits)-1].SHA
	}

	// Classify aggregate files.
	filePaths := make([]string, 0, len(aggregateFiles))
	for _, f := range aggregateFiles {
		filePaths = append(filePaths, f.Filename)
	}
	fileResults := classify.Classify(filePaths, src.Paths)

	// Build file results for JSON output.
	patches := patchesByFilename(aggregateFiles)
	for _, fr := range fileResults {
		result.Files = append(result.Files, buildFileResult(fr.Path, fr.Level, patches))
	}

	result.Commits = buildCommitResults(commits, fileResults, src.Paths)

	return result, headSHA, nil
}

func checkSourceReleases(client github.Client, src config.Source) (*display.SourceResult, string, string, error) {
	result := &display.SourceResult{Source: src}

	latest, err := client.GetLatestRelease(src.Repo)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			fmt.Fprintf(os.Stderr, "warning: %s: no releases found\n", src.Repo)
			return result, "", "", nil
		}
		return nil, "", "", fmt.Errorf("fetching latest release: %w", err)
	}

	releaseInfo := &display.ReleaseInfo{
		TagName:     latest.TagName,
		Name:        latest.Name,
		Body:        latest.Body,
		PublishedAt: latest.PublishedAt,
		URL:         latest.HTMLURL,
	}

	if src.LastCheckedTag == latest.TagName {
		// No new release — refresh the timestamp but don't re-surface
		// the release: the display layer would otherwise render it as
		// either a fresh release or a first-run "Tracking releases from"
		// banner every check.
		headSHA, err := client.GetHeadSHA(src.Repo, latest.TagName)
		if err != nil {
			return nil, "", "", fmt.Errorf("getting tag SHA: %w", err)
		}
		return result, headSHA, latest.TagName, nil
	}

	if src.LastCheckedTag == "" {
		// First run — record current release without showing commits.
		headSHA, err := client.GetHeadSHA(src.Repo, latest.TagName)
		if err != nil {
			return nil, "", "", fmt.Errorf("getting tag SHA: %w", err)
		}
		result.Release = releaseInfo
		return result, headSHA, latest.TagName, nil
	}

	// New release since last check — compare tags.
	releaseInfo.PrevTag = src.LastCheckedTag

	cmp, err := client.Compare(src.Repo, src.LastCheckedTag, latest.TagName)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			// Old tag deleted/force-pushed — treat as first run.
			headSHA, err := client.GetHeadSHA(src.Repo, latest.TagName)
			if err != nil {
				return nil, "", "", fmt.Errorf("getting tag SHA: %w", err)
			}
			releaseInfo.PrevTag = ""
			result.Release = releaseInfo
			return result, headSHA, latest.TagName, nil
		}
		return nil, "", "", fmt.Errorf("comparing tags: %w", err)
	}

	// Classify files.
	filePaths := make([]string, 0, len(cmp.Files))
	for _, f := range cmp.Files {
		filePaths = append(filePaths, f.Filename)
	}
	fileResults := classify.Classify(filePaths, src.Paths)
	patches := patchesByFilename(cmp.Files)
	for _, fr := range fileResults {
		result.Files = append(result.Files, buildFileResult(fr.Path, fr.Level, patches))
	}

	result.Commits = buildCommitResults(cmp.Commits, fileResults, src.Paths)

	headSHA := ""
	if len(cmp.Commits) > 0 {
		headSHA = cmp.Commits[len(cmp.Commits)-1].SHA
	} else {
		headSHA, err = client.GetHeadSHA(src.Repo, latest.TagName)
		if err != nil {
			return nil, "", "", fmt.Errorf("getting tag SHA: %w", err)
		}
	}

	result.Release = releaseInfo
	return result, headSHA, latest.TagName, nil
}

// buildCommitResults converts github commits into display results, classifying
// each by its highest-level file. Commits without per-file data fall back to
// the aggregate-file classification (fileResults).
func buildCommitResults(commits []github.Commit, fileResults []classify.Result, paths config.PathDefs) []display.CommitResult {
	out := make([]display.CommitResult, 0, len(commits))
	for _, c := range commits {
		var level classify.Level
		if len(c.Files) > 0 {
			cFilePaths := make([]string, 0, len(c.Files))
			for _, f := range c.Files {
				cFilePaths = append(cFilePaths, f.Filename)
			}
			level = classify.MaxLevel(classify.Classify(cFilePaths, paths))
		} else {
			level = classify.MaxLevel(fileResults)
		}
		out = append(out, display.CommitResult{
			SHA:     c.SHA,
			Message: c.Message,
			Body:    c.Body,
			Author:  c.Author,
			Date:    c.Date.Format(commitDateFormat),
			Level:   level,
		})
	}
	return out
}

// patchesByFilename indexes file patches (most recent wins on duplicates).
func patchesByFilename(files []github.File) map[string]string {
	if !reviewWithDiffs {
		return nil
	}
	m := make(map[string]string, len(files))
	for _, f := range files {
		if f.Patch != "" {
			m[f.Filename] = f.Patch
		}
	}
	return m
}

// buildFileResult constructs a FileResult, attaching a capped diff when
// --with-diffs is set and a patch is available.
func buildFileResult(path string, level classify.Level, patches map[string]string) display.FileResult {
	fr := display.FileResult{Path: path, Level: level}
	if patches == nil {
		return fr
	}
	patch, ok := patches[path]
	if !ok {
		return fr
	}
	diff, truncated, skipped := capDiff(patch)
	fr.Diff = diff
	fr.DiffTruncated = truncated
	fr.DiffSkipped = skipped
	return fr
}

// fetchCommitDetails fetches file details for up to limit commits in parallel,
// setting each commit's Files field and returning the aggregate file list.
func fetchCommitDetails(client github.Client, repo string, commits []github.Commit, limit int) []github.File {
	type indexedFiles struct {
		idx   int
		files []github.File
	}

	var g errgroup.Group
	ch := make(chan indexedFiles, limit)

	for i := range limit {
		g.Go(func() error {
			detail, err := client.GetCommitDetail(repo, commits[i].SHA)
			if err != nil {
				return nil //nolint:nilerr // non-fatal: skip commits we can't fetch
			}
			ch <- indexedFiles{idx: i, files: detail.Files}
			return nil
		})
	}

	go func() {
		_ = g.Wait()
		close(ch)
	}()

	var aggregateFiles []github.File
	for result := range ch {
		commits[result.idx].Files = result.files
		aggregateFiles = append(aggregateFiles, result.files...)
	}

	return aggregateFiles
}

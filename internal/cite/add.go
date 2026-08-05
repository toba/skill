package cite

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/toba/jig-go/v4/internal/config"
	"github.com/toba/jig-go/v4/internal/constants"
	"github.com/toba/jig-go/v4/internal/github"
)

// RepoArg holds parsed repository information from a user-provided argument.
type RepoArg struct {
	Owner    string
	Repo     string
	Host     string // "github.com" or other host
	FullURL  string // original URL for non-GitHub repos
	IsGitHub bool
}

const defaultHost = "github.com"

var (
	// owner/repo shorthand
	shorthandRe = regexp.MustCompile(`^([a-zA-Z0-9._-]+)/([a-zA-Z0-9._-]+)$`)
	// https://github.com/owner/repo[.git]
	httpsGitHubRe = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/.]+?)(?:\.git)?/?$`)
	// git@github.com:owner/repo.git
	sshGitHubRe = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/.]+?)(?:\.git)?$`)
)

// ParseRepoArg extracts repo info from a URL or owner/repo shorthand.
func ParseRepoArg(arg string) RepoArg {
	if m := httpsGitHubRe.FindStringSubmatch(arg); m != nil {
		return RepoArg{Owner: m[1], Repo: m[2], Host: defaultHost, IsGitHub: true}
	}
	if m := sshGitHubRe.FindStringSubmatch(arg); m != nil {
		return RepoArg{Owner: m[1], Repo: m[2], Host: defaultHost, IsGitHub: true}
	}
	if m := shorthandRe.FindStringSubmatch(arg); m != nil {
		return RepoArg{Owner: m[1], Repo: m[2], Host: defaultHost, IsGitHub: true}
	}
	return RepoArg{FullURL: arg, IsGitHub: false}
}

// Inspect inspects a repository and returns a suggested Source config.
func Inspect(client github.Client, arg RepoArg) (*config.Source, error) {
	if arg.IsGitHub {
		return inspectGitHub(client, arg)
	}
	return inspectGit(arg)
}

func inspectGitHub(client github.Client, arg RepoArg) (*config.Source, error) {
	slug := arg.Owner + "/" + arg.Repo

	info, err := client.GetRepo(slug)
	if err != nil {
		return nil, fmt.Errorf("fetching repo %s: %w", slug, err)
	}

	branch := cmp.Or(info.DefaultBranch, constants.DefaultBranch)

	tree, err := client.GetTree(slug, branch)
	if err != nil {
		return nil, fmt.Errorf("fetching tree for %s@%s: %w", slug, branch, err)
	}

	var files []string
	for _, e := range tree.Tree {
		if e.Type == "blob" {
			files = append(files, e.Path)
		}
	}

	src := &config.Source{
		Repo:   slug,
		Branch: branch,
		Notes:  info.Description,
		Paths:  SuggestPaths(files),
	}
	return src, nil
}

func inspectGit(arg RepoArg) (*config.Source, error) {
	tmpDir, err := os.MkdirTemp("", "jig-cite-add-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // best-effort cleanup

	cmd := exec.Command("git", "clone", "--depth=1", "--single-branch", arg.FullURL, tmpDir) //nolint:gosec // git clone with user-provided URL
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cloning %s: %w", arg.FullURL, err)
	}

	// Detect default branch.
	branch := constants.DefaultBranch
	out, err := exec.Command("git", "-C", tmpDir, "symbolic-ref", "refs/remotes/origin/HEAD").Output() //nolint:gosec // git command
	if err == nil {
		ref := strings.TrimSpace(string(out))
		// refs/remotes/origin/main → main
		if parts := strings.Split(ref, "/"); len(parts) > 0 {
			branch = parts[len(parts)-1]
		}
	}

	// Walk the file tree.
	var files []string
	_ = filepath.Walk(tmpDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort walk: skip unreadable entries
		}
		rel, _ := filepath.Rel(tmpDir, p)
		if strings.HasPrefix(rel, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			files = append(files, rel)
		}
		return nil
	})

	// Try to extract owner/repo from URL path.
	repo := arg.FullURL
	if u := extractRepoSlug(arg.FullURL); u != "" {
		repo = u
	}

	src := &config.Source{
		Repo:   repo,
		Branch: branch,
		Paths:  SuggestPaths(files),
	}
	return src, nil
}

// extractRepoSlug attempts to extract an "owner/repo" slug from a git URL.
func extractRepoSlug(url string) string {
	// Strip common prefixes and suffixes.
	u := url
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimSuffix(u, "/")

	// Handle ssh-style URLs: git@host:owner/repo
	if idx := strings.Index(u, ":"); idx > 0 && !strings.Contains(u[:idx], "/") {
		u = u[idx+1:]
		if m := shorthandRe.FindStringSubmatch(u); m != nil {
			return m[1] + "/" + m[2]
		}
		return ""
	}

	// Handle https-style: host/owner/repo
	parts := strings.Split(u, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return ""
}

// SuggestPaths generates path classification globs based on file extensions in a tree.
func SuggestPaths(files []string) config.PathDefs {
	// Count extensions.
	extCount := map[string]int{}
	hasSrcDir := false
	hasSourcesDir := false

	for _, f := range files {
		ext := path.Ext(f)
		if ext != "" {
			extCount[ext]++
		}
		dir := strings.SplitN(f, "/", 2)[0]
		if dir == "src" {
			hasSrcDir = true
		}
		if dir == "Sources" {
			hasSourcesDir = true
		}
	}

	var pd config.PathDefs

	// Language-specific rules.
	switch {
	case extCount[".go"] > 0:
		pd.High = append(pd.High, "**/*.go")
		pd.Medium = addIfPresent(pd.Medium, files, "go.mod", "go.sum")
	case extCount[".swift"] > 0:
		if hasSourcesDir {
			pd.High = append(pd.High, "Sources/**/*.swift")
		} else {
			pd.High = append(pd.High, "**/*.swift")
		}
		pd.Medium = addIfPresent(pd.Medium, files, "Package.swift")
	case extCount[".rs"] > 0:
		if hasSrcDir {
			pd.High = append(pd.High, "src/**/*.rs")
		} else {
			pd.High = append(pd.High, "**/*.rs")
		}
		pd.Medium = addIfPresent(pd.Medium, files, "Cargo.toml", "Cargo.lock")
	case extCount[".ts"] > 0 || extCount[".tsx"] > 0:
		if hasSrcDir {
			pd.High = append(pd.High, "src/**/*.ts", "src/**/*.tsx")
		} else {
			pd.High = append(pd.High, "**/*.ts", "**/*.tsx")
		}
		pd.Medium = addIfPresent(pd.Medium, files, "package.json")
	case extCount[".js"] > 0 || extCount[".jsx"] > 0:
		if hasSrcDir {
			pd.High = append(pd.High, "src/**/*.js", "src/**/*.jsx")
		} else {
			pd.High = append(pd.High, "**/*.js", "**/*.jsx")
		}
		pd.Medium = addIfPresent(pd.Medium, files, "package.json")
	case extCount[".py"] > 0:
		pd.High = append(pd.High, "**/*.py")
		pd.Medium = addIfPresent(pd.Medium, files, "requirements.txt", "pyproject.toml", "setup.py")
	default:
		// Fallback: top 2 extensions by count.
		type extEntry struct {
			ext   string
			count int
		}
		var entries []extEntry
		for ext, count := range extCount {
			entries = append(entries, extEntry{ext, count})
		}
		slices.SortFunc(entries, func(a, b extEntry) int {
			return cmp.Compare(b.count, a.count) // descending
		})
		for i := range min(2, len(entries)) {
			pd.High = append(pd.High, "**/*"+entries[i].ext)
		}
	}

	// Always add low-priority patterns.
	pd.Low = []string{".github/**", "README.md", "LICENSE"}

	return pd
}

// addIfPresent adds file names to a slice only if they exist in the file list.
func addIfPresent(slice, files []string, names ...string) []string {
	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}
	for _, name := range names {
		if fileSet[name] {
			slice = append(slice, name)
		}
	}
	return slice
}

// FormatSourceYAML renders a Source as YAML for display.
func FormatSourceYAML(src *config.Source) (string, error) {
	var b strings.Builder
	b.WriteString("- repo: " + src.Repo + "\n")
	if src.Branch != "" {
		b.WriteString("  branch: " + src.Branch + "\n")
	}
	if src.Track != "" {
		b.WriteString("  track: " + src.Track + "\n")
	}
	if src.Scope != "" {
		b.WriteString("  scope: " + quote(src.Scope) + "\n")
	}
	if src.Notes != "" {
		b.WriteString("  notes: " + quote(src.Notes) + "\n")
	}
	b.WriteString("  paths:\n")
	if len(src.Paths.High) > 0 {
		b.WriteString("    high: " + flowSeq(src.Paths.High) + "\n")
	}
	if len(src.Paths.Medium) > 0 {
		b.WriteString("    medium: " + flowSeq(src.Paths.Medium) + "\n")
	}
	if len(src.Paths.Low) > 0 {
		b.WriteString("    low: " + flowSeq(src.Paths.Low) + "\n")
	}
	return b.String(), nil
}

func flowSeq(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = quote(s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func quote(s string) string {
	if strings.ContainsAny(s, `"'*{}[]!&|>#%@`+"`") || strings.Contains(s, ": ") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

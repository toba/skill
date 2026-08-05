package commit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GitignoreCandidates returns untracked files that match gitignore patterns.
func GitignoreCandidates() ([]string, error) {
	out, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output()
	if err != nil {
		return nil, fmt.Errorf("listing untracked files: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	var candidates []string
	for file := range strings.SplitSeq(raw, "\n") {
		if matchesGitignorePattern(file) {
			candidates = append(candidates, file)
		}
	}
	return candidates, nil
}

// matchesGitignorePattern reports whether a file path matches any built-in
// gitignore candidate pattern.
func matchesGitignorePattern(path string) bool {
	for _, re := range gitignorePatterns {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

// StageAll runs git add -A and returns the short status output.
func StageAll() (string, error) {
	if err := exec.Command("git", "add", "-A").Run(); err != nil {
		return "", fmt.Errorf("git add -A: %w", err)
	}
	out, err := exec.Command("git", "status", "--short").Output()
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// Diff returns the staged diff output.
func Diff() (string, error) {
	out, err := exec.Command("git", "diff", "--staged").Output()
	if err != nil {
		return "", fmt.Errorf("git diff --staged: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// LatestTag returns the latest v* tag by version sort, or "" if none exist.
func LatestTag() (string, error) {
	out, err := exec.Command("git", "tag", "-l", "v*", "--sort=version:refname").Output()
	if err != nil {
		return "", fmt.Errorf("git tag -l: %w", err)
	}
	lines := strings.TrimSpace(string(out))
	if lines == "" {
		return "", nil
	}
	parts := strings.Split(lines, "\n")
	return parts[len(parts)-1], nil
}

// RecentCommits returns recent commits for style reference.
// If a tag is provided and there are commits since it, returns those.
// Otherwise returns the last 20 commits so agents always have style context.
func RecentCommits(tag string) (string, error) {
	if tag != "" {
		cmd := exec.Command("git", "log", tag+"..HEAD", "--format=%h %s") //nolint:gosec // args from internal config
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git log: %w", err)
		}
		if result := strings.TrimRight(string(out), "\n"); result != "" {
			return result, nil
		}
	}
	// No tag or no commits since tag — show recent commits for style reference.
	out, err := exec.Command("git", "log", "--format=%h %s", "-20").Output()
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// HasStagedChanges reports whether there are any staged changes to commit.
func HasStagedChanges() (bool, error) {
	err := exec.Command("git", "diff", "--cached", "--quiet").Run()
	if err == nil {
		return false, nil // exit 0 = no differences
	}
	if _, ok := errors.AsType[*exec.ExitError](err); ok {
		return true, nil // exit 1 = differences exist
	}
	return false, fmt.Errorf("git diff --cached --quiet: %w", err)
}

// Commit creates a git commit with the given message.
// Stderr is captured and included in the error so hook failures are visible.
//
// If a pre-commit hook modifies staged files (e.g. a formatter), the
// modifications are automatically staged and amended into the commit so
// callers never see a dirty working tree caused by hooks.
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message) //nolint:gosec // args from internal config
	cmd.WaitDelay = 10 * time.Second
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return fmt.Errorf("git commit: %w\n%s", err, detail)
		}
		return fmt.Errorf("git commit: %w", err)
	}

	// Check if pre-commit hooks left unstaged modifications (e.g. formatter
	// rewrote files after they were committed). If so, fold them in.
	if dirty, _ := hasUnstagedChanges(); dirty {
		if err := exec.Command("git", "add", "-u").Run(); err != nil {
			return fmt.Errorf("restaging hook changes: %w", err)
		}
		amend := exec.Command("git", "commit", "--amend", "--no-edit")
		amend.WaitDelay = 10 * time.Second
		if out, err := amend.CombinedOutput(); err != nil {
			detail := strings.TrimSpace(string(out))
			if detail != "" {
				return fmt.Errorf("amending hook changes: %w\n%s", err, detail)
			}
			return fmt.Errorf("amending hook changes: %w", err)
		}
	}

	return nil
}

// hasUnstagedChanges reports whether tracked files have unstaged modifications.
func hasUnstagedChanges() (bool, error) {
	err := exec.Command("git", "diff", "--quiet").Run()
	if err == nil {
		return false, nil
	}
	if _, ok := errors.AsType[*exec.ExitError](err); ok {
		return true, nil
	}
	return false, fmt.Errorf("git diff --quiet: %w", err)
}

// Tag creates a git tag with the given name.
func Tag(version string) error {
	if err := exec.Command("git", "tag", version).Run(); err != nil { //nolint:gosec // args from internal config
		return fmt.Errorf("git tag %s: %w", version, err)
	}
	return nil
}

// pushRetryDelays defines the backoff delays between push retry attempts.
var pushRetryDelays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// pushLogf is the function used to log retry messages. It defaults to
// fmt.Fprintf(os.Stderr, ...) and can be overridden in tests.
var pushLogf = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// Push pushes the current branch, then pushes any unpushed version tags
// individually in semver order. This prevents out-of-order GitHub releases
// when multiple tags are pushed simultaneously.
//
// Transient network failures (git exit code 128) are retried with
// exponential backoff up to 3 times.
func Push() error {
	if err := pushWithRetry("git push", "git", "push"); err != nil {
		return err
	}

	tags, err := unpushedVersionTags()
	if err != nil {
		// Can't determine unpushed tags (e.g. no remote) — push all tags.
		if err := pushWithRetry("git push --tags", "git", "push", "--tags"); err != nil {
			return err
		}
		return nil
	}

	for _, tag := range tags {
		if err := pushWithRetry("git push origin "+tag, "git", "push", "origin", tag); err != nil { //nolint:gosec // args from internal config
			return err
		}
	}
	return nil
}

// pushWithRetry runs a git push command, retrying on transient failures
// (exit code 128, which git uses for network/TLS errors).
func pushWithRetry(desc string, args ...string) error {
	var lastErr error
	for attempt := range len(pushRetryDelays) + 1 {
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // args from internal caller
		lastErr = cmd.Run()
		if lastErr == nil {
			return nil
		}
		if !isTransientGitError(lastErr) {
			return fmt.Errorf("%s: %w", desc, lastErr)
		}
		if attempt < len(pushRetryDelays) {
			delay := pushRetryDelays[attempt]
			pushLogf("  %s failed (transient), retrying in %v…\n", desc, delay)
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("%s: %w (after %d retries)", desc, lastErr, len(pushRetryDelays))
}

// isTransientGitError reports whether an exec error represents a transient
// git failure (exit code 128, used for network/TLS/connection errors).
func isTransientGitError(err error) bool {
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		return ee.ExitCode() == 128
	}
	return false
}

// unpushedVersionTags returns local v* tags not present on the remote,
// sorted in version order (oldest first).
func unpushedVersionTags() ([]string, error) {
	// Get local v* tags sorted by version.
	localOut, err := exec.Command("git", "tag", "-l", "v*", "--sort=version:refname").Output()
	if err != nil {
		return nil, fmt.Errorf("listing local tags: %w", err)
	}

	// Get remote tags.
	remoteOut, err := exec.Command("git", "ls-remote", "--tags", "origin").Output()
	if err != nil {
		return nil, fmt.Errorf("listing remote tags: %w", err)
	}

	remote := make(map[string]bool)
	for line := range strings.SplitSeq(strings.TrimSpace(string(remoteOut)), "\n") {
		// Format: "<sha>\trefs/tags/<name>" (skip ^{} derefs)
		if _, ref, ok := strings.Cut(line, "\t"); ok {
			tag := strings.TrimPrefix(ref, "refs/tags/")
			if !strings.HasSuffix(tag, "^{}") {
				remote[tag] = true
			}
		}
	}

	var unpushed []string
	for tag := range strings.SplitSeq(strings.TrimSpace(string(localOut)), "\n") {
		if tag != "" && !remote[tag] {
			unpushed = append(unpushed, tag)
		}
	}
	return unpushed, nil
}

// Status returns the current git status output.
func Status() (string, error) {
	out, err := exec.Command("git", "status", "--short").Output()
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

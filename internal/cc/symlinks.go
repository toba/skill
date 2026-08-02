package cc

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// SyncReport describes the result of syncing one alias's symlinks.
type SyncReport struct {
	Alias     string   `json:"alias"`
	Created   []string `json:"created"`
	Skipped   []string `json:"skipped"`
	Repaired  []string `json:"repaired"`
	Conflicts []string `json:"conflicts"`
	Private   []string `json:"private"`
}

// Health describes the health of one alias's symlink layout.
type Health struct {
	Alias     string   `json:"alias"`
	Valid     []string `json:"valid"`
	Broken    []string `json:"broken"`
	Missing   []string `json:"missing"`
	Conflicts []string `json:"conflicts"`
	Orphaned  []string `json:"orphaned"`
}

// HasIssues reports whether the health record contains any failures.
func (h Health) HasIssues() bool {
	return len(h.Broken)+len(h.Missing)+len(h.Conflicts)+len(h.Orphaned) > 0
}

// sharedEntries returns the names of items in `source` that are NOT in private.
func sharedEntries(source string, private []string) ([]string, error) {
	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, fmt.Errorf("reading source %s: %w", source, err)
	}
	privSet := make(map[string]struct{}, len(private))
	for _, p := range private {
		privSet[p] = struct{}{}
	}
	var out []string
	for _, e := range entries {
		if _, isPrivate := privSet[e.Name()]; isPrivate {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// Sync ensures the alias dir contains correct symlinks for all shared
// entries. Real files in private positions are left alone. Real files in
// shared positions are reported as conflicts and not modified.
func Sync(c *Config, aliasName string) (*SyncReport, error) {
	a, ok := c.Aliases[aliasName]
	if !ok {
		return nil, fmt.Errorf("unknown alias %q", aliasName)
	}
	rep := &SyncReport{Alias: aliasName}
	if a.IsSource {
		// Source has no symlinks to manage.
		return rep, nil
	}

	if err := os.MkdirAll(a.Path, 0o755); err != nil {
		return nil, err
	}

	priv := c.PrivateList()
	shared, err := sharedEntries(c.SharedSource, priv)
	if err != nil {
		return nil, err
	}

	for _, name := range shared {
		target := filepath.Join(c.SharedSource, name)
		link := filepath.Join(a.Path, name)
		info, err := os.Lstat(link)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			if err := os.Symlink(target, link); err != nil {
				return nil, err
			}
			rep.Created = append(rep.Created, name)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			rep.Conflicts = append(rep.Conflicts, name)
			continue
		}
		actual, err := os.Readlink(link)
		if err != nil {
			return nil, err
		}
		if actual == target {
			rep.Skipped = append(rep.Skipped, name)
			continue
		}
		if err := os.Remove(link); err != nil {
			return nil, err
		}
		if err := os.Symlink(target, link); err != nil {
			return nil, err
		}
		rep.Repaired = append(rep.Repaired, name)
	}

	// Track private entries that exist as real files.
	for _, name := range priv {
		p := filepath.Join(a.Path, name)
		if _, err := os.Lstat(p); err == nil {
			rep.Private = append(rep.Private, name)
		}
	}

	return rep, nil
}

// CheckHealth classifies every entry under an alias dir.
func CheckHealth(c *Config, aliasName string) (*Health, error) {
	a, ok := c.Aliases[aliasName]
	if !ok {
		return nil, fmt.Errorf("unknown alias %q", aliasName)
	}
	h := &Health{Alias: aliasName}
	if a.IsSource {
		return h, nil
	}

	priv := c.PrivateList()
	shared, err := sharedEntries(c.SharedSource, priv)
	if err != nil {
		return nil, err
	}
	sharedSet := make(map[string]struct{}, len(shared))
	for _, n := range shared {
		sharedSet[n] = struct{}{}
	}
	privSet := make(map[string]struct{}, len(priv))
	for _, n := range priv {
		privSet[n] = struct{}{}
	}

	for _, name := range shared {
		link := filepath.Join(a.Path, name)
		target := filepath.Join(c.SharedSource, name)
		info, err := os.Lstat(link)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				h.Missing = append(h.Missing, name)
				continue
			}
			return nil, err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			h.Conflicts = append(h.Conflicts, name)
			continue
		}
		actual, _ := os.Readlink(link)
		if actual != target {
			h.Broken = append(h.Broken, name)
			continue
		}
		if _, err := os.Stat(target); err != nil {
			h.Broken = append(h.Broken, name)
			continue
		}
		h.Valid = append(h.Valid, name)
	}

	// Walk alias dir to find orphaned symlinks: links to source that no
	// longer exist there.
	entries, err := os.ReadDir(a.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return h, nil
		}
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if _, isShared := sharedSet[name]; isShared {
			continue
		}
		if _, isPriv := privSet[name]; isPriv {
			continue
		}
		// Anything else: only flag symlinks pointing into the source dir.
		link := filepath.Join(a.Path, name)
		info, err := os.Lstat(link)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		actual, _ := os.Readlink(link)
		if filepath.Dir(actual) == c.SharedSource {
			h.Orphaned = append(h.Orphaned, name)
		}
	}

	return h, nil
}

// CopyPrivateFiles copies real private files from src into dst, skipping
// symlinks and missing entries.
func CopyPrivateFiles(src, dst string, private []string) error {
	for _, name := range private {
		s := filepath.Join(src, name)
		d := filepath.Join(dst, name)
		info, err := os.Lstat(s)
		if err != nil {
			continue
		}
		// Skip symlinks; we only copy real data.
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		info, err := os.Lstat(s)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveAliasDir deletes the alias directory tree, refusing to remove the
// source alias's path.
func RemoveAliasDir(c *Config, name string) error {
	a, ok := c.Aliases[name]
	if !ok {
		return fmt.Errorf("unknown alias %q", name)
	}
	if a.IsSource {
		return fmt.Errorf("refusing to remove source alias %q", name)
	}
	return os.RemoveAll(a.Path)
}

// SeedClaudeJSON seeds an alias's .claude.json (if absent) from the shared
// source. Claude refuses to launch without this file, and it is in the private
// list so it is not symlinked. The source copy is scrubbed of IdentityFields
// (machineID, userID, oauthAccount) so a fresh alias for a different account
// does not inherit the source account's identity — each account regenerates
// its own on first login. This keeps accounts isolated; it does not hide
// anything from Anthropic.
func SeedClaudeJSON(c *Config, name string) error {
	a, ok := c.Aliases[name]
	if !ok {
		return fmt.Errorf("unknown alias %q", name)
	}
	if a.IsSource {
		return nil
	}
	dst := filepath.Join(a.Path, ".claude.json")
	if _, err := os.Lstat(dst); err == nil {
		return nil
	}
	src := filepath.Join(c.SharedSource, ".claude.json")
	data, err := os.ReadFile(src)
	if err != nil {
		return nil //nolint:nilerr // source absent: nothing to seed, not an error
	}
	scrubbed, err := scrubIdentity(data)
	if err != nil {
		return fmt.Errorf("scrubbing identity from %s: %w", src, err)
	}
	if err := os.MkdirAll(a.Path, 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, scrubbed, 0o644)
}

// EnsureAliasDirs walks the configured alias map and ensures each non-source
// alias's path directory exists.
func EnsureAliasDirs(c *Config) error {
	names := slices.Clone(c.Names())
	for _, n := range names {
		a := c.Aliases[n]
		if a.IsSource {
			continue
		}
		if err := os.MkdirAll(a.Path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

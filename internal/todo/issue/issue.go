package issue

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// ValidateTag checks if a tag is valid (any non-empty string).
func ValidateTag(tag string) error {
	if strings.TrimSpace(tag) == "" {
		return errors.New("tag cannot be empty")
	}
	return nil
}

// NormalizeTag converts a tag to its canonical form (lowercase).
func NormalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

// HasTag returns true if the issue has the specified tag.
func (b *Issue) HasTag(tag string) bool {
	normalized := NormalizeTag(tag)
	return slices.Contains(b.Tags, normalized)
}

// AddTag adds a tag to the issue if it doesn't already exist.
// The tag is stored as given (after trim); dedup uses case-insensitive comparison.
// Returns an error if the tag is invalid.
func (b *Issue) AddTag(tag string) error {
	trimmed := strings.TrimSpace(tag)
	if err := ValidateTag(trimmed); err != nil {
		return err
	}
	if !b.HasTag(trimmed) {
		b.Tags = append(b.Tags, trimmed)
	}
	return nil
}

// RemoveTag removes a tag from the issue.
func (b *Issue) RemoveTag(tag string) {
	normalized := NormalizeTag(tag)
	b.Tags = slices.DeleteFunc(b.Tags, func(s string) bool { return s == normalized })
}

// HasParent returns true if the issue has a parent.
func (b *Issue) HasParent() bool {
	return b.Parent != ""
}

// IsBlocking returns true if this issue is blocking the given issue ID.
func (b *Issue) IsBlocking(id string) bool {
	return slices.Contains(b.Blocking, id)
}

// AddBlocking adds an issue ID to the blocking list if not already present.
func (b *Issue) AddBlocking(id string) {
	if !b.IsBlocking(id) {
		b.Blocking = append(b.Blocking, id)
	}
}

// RemoveBlocking removes an issue ID from the blocking list.
func (b *Issue) RemoveBlocking(id string) {
	b.Blocking = slices.DeleteFunc(b.Blocking, func(s string) bool { return s == id })
}

// IsBlockedBy returns true if this issue is blocked by the given issue ID.
func (b *Issue) IsBlockedBy(id string) bool {
	return slices.Contains(b.BlockedBy, id)
}

// AddBlockedBy adds an issue ID to the blocked-by list if not already present.
func (b *Issue) AddBlockedBy(id string) {
	if !b.IsBlockedBy(id) {
		b.BlockedBy = append(b.BlockedBy, id)
	}
}

// RemoveBlockedBy removes an issue ID from the blocked-by list.
func (b *Issue) RemoveBlockedBy(id string) {
	b.BlockedBy = slices.DeleteFunc(b.BlockedBy, func(s string) bool { return s == id })
}

// HasSync returns true if the issue has sync data for the given name.
func (b *Issue) HasSync(name string) bool {
	if b.Sync == nil {
		return false
	}
	_, ok := b.Sync[name]
	return ok
}

// SetSync sets the sync data for a name (full replacement).
func (b *Issue) SetSync(name string, data map[string]any) {
	if b.Sync == nil {
		b.Sync = make(map[string]map[string]any)
	}
	b.Sync[name] = data
}

// RemoveSync removes the sync data for a name.
func (b *Issue) RemoveSync(name string) {
	if b.Sync == nil {
		return
	}
	delete(b.Sync, name)
	if len(b.Sync) == 0 {
		b.Sync = nil
	}
}

// DueDate is a date-only wrapper around time.Time that serializes as "YYYY-MM-DD".
type DueDate struct {
	time.Time
}

// DueDateFormat is the format used for due dates.
const DueDateFormat = "2006-01-02"

// NewDueDate creates a DueDate from a time.Time, zeroing the time component.
func NewDueDate(t time.Time) *DueDate {
	d := DueDate{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
	return &d
}

// ParseDueDate parses a "YYYY-MM-DD" string into a DueDate.
func ParseDueDate(s string) (*DueDate, error) {
	t, err := time.Parse(DueDateFormat, s)
	if err != nil {
		return nil, fmt.Errorf("invalid due date %q: expected YYYY-MM-DD format", s)
	}
	return NewDueDate(t), nil
}

// MarshalYAML implements yaml.Marshaler to serialize as "YYYY-MM-DD".
func (d DueDate) MarshalYAML() (any, error) {
	return d.Format(DueDateFormat), nil
}

// UnmarshalYAML implements yaml.Unmarshaler to parse "YYYY-MM-DD".
func (d *DueDate) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	t, err := time.Parse(DueDateFormat, s)
	if err != nil {
		return fmt.Errorf("invalid due date %q: expected YYYY-MM-DD format", s)
	}
	d.Time = t
	return nil
}

// MarshalJSON implements json.Marshaler to serialize as "YYYY-MM-DD".
func (d DueDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(DueDateFormat))
}

// UnmarshalJSON implements json.Unmarshaler to parse "YYYY-MM-DD".
func (d *DueDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t, err := time.Parse(DueDateFormat, s)
	if err != nil {
		return fmt.Errorf("invalid due date %q: expected YYYY-MM-DD format", s)
	}
	d.Time = t
	return nil
}

// String returns the date as "YYYY-MM-DD".
func (d DueDate) String() string {
	return d.Format(DueDateFormat)
}

// Issue represents an issue stored as a markdown file with front matter.
type Issue struct {
	// ID is the unique NanoID identifier (from filename).
	ID string `yaml:"-" json:"id"`
	// Slug is the optional human-readable part of the filename.
	Slug string `yaml:"-" json:"slug,omitempty"`
	// Path is the relative path from the issues root (e.g., "a/abc-def--login.md").
	Path string `yaml:"-" json:"path"`

	// Front matter fields
	Title     string `yaml:"title" json:"title"`
	Status    string `yaml:"status" json:"status"`
	Type      string `yaml:"type,omitempty" json:"type,omitempty"`
	Priority  string `yaml:"priority,omitempty" json:"priority,omitempty"`
	Milestone string `yaml:"milestone,omitempty" json:"milestone,omitempty"` // milestone id
	// ExternalID is an optional free-form identifier referencing this issue in
	// another system (e.g. a Jira key or a ticket number in a legacy tracker).
	ExternalID string     `yaml:"external_id,omitempty" json:"external_id,omitempty"`
	Tags       []string   `yaml:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt  *time.Time `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt  *time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
	Due        *DueDate   `yaml:"due,omitempty" json:"due,omitempty"`

	// Body is the markdown content after the front matter.
	Body string `yaml:"-" json:"body,omitempty"`

	// Parent is the optional parent issue ID (milestone, epic, or feature).
	Parent string `yaml:"parent,omitempty" json:"parent,omitempty"`

	// Blocking is a list of issue IDs that this issue is blocking.
	Blocking []string `yaml:"blocking,omitempty" json:"blocking,omitempty"`

	// BlockedBy is a list of issue IDs that are blocking this issue.
	BlockedBy []string `yaml:"blocked_by,omitempty" json:"blocked_by,omitempty"`

	// Sync holds sync integration metadata keyed by integration name.
	Sync map[string]map[string]any `yaml:"sync,omitempty" json:"sync,omitempty"`
}

// frontMatter is the subset of Issue that gets serialized to YAML front matter.
type frontMatter struct {
	Title      string                    `yaml:"title"`
	Status     string                    `yaml:"status"`
	Type       string                    `yaml:"type,omitempty"`
	Priority   string                    `yaml:"priority,omitempty"`
	Milestone  string                    `yaml:"milestone,omitempty"`
	ExternalID string                    `yaml:"external_id,omitempty"`
	Tags       []string                  `yaml:"tags,omitempty"`
	CreatedAt  *time.Time                `yaml:"created_at,omitempty"`
	UpdatedAt  *time.Time                `yaml:"updated_at,omitempty"`
	Due        *DueDate                  `yaml:"due,omitempty"`
	Parent     string                    `yaml:"parent,omitempty"`
	Blocking   []string                  `yaml:"blocking,omitempty"`
	BlockedBy  []string                  `yaml:"blocked_by,omitempty"`
	Sync       map[string]map[string]any `yaml:"sync,omitempty"`
}

// Parse reads an issue from a reader (markdown with YAML front matter).
func Parse(r io.Reader) (*Issue, error) {
	var fm frontMatter
	body, err := frontmatter.Parse(r, &fm)
	if err != nil {
		return nil, fmt.Errorf("parsing front matter: %w", err)
	}

	// Trim trailing newline from body (POSIX files end with newline, but it's not part of content)
	bodyStr := strings.TrimSuffix(string(body), "\n")

	return &Issue{
		Title:      fm.Title,
		Status:     fm.Status,
		Type:       fm.Type,
		Priority:   fm.Priority,
		Milestone:  fm.Milestone,
		ExternalID: fm.ExternalID,
		Tags:       fm.Tags,
		CreatedAt:  fm.CreatedAt,
		UpdatedAt:  fm.UpdatedAt,
		Due:        fm.Due,
		Body:       bodyStr,
		Parent:     fm.Parent,
		Blocking:   fm.Blocking,
		BlockedBy:  fm.BlockedBy,
		Sync:       fm.Sync,
	}, nil
}

// renderFrontMatter is used for YAML output with yaml.v3 (supports custom marshalers).
type renderFrontMatter struct {
	Title      string                    `yaml:"title"`
	Status     string                    `yaml:"status"`
	Type       string                    `yaml:"type,omitempty"`
	Priority   string                    `yaml:"priority,omitempty"`
	Milestone  string                    `yaml:"milestone,omitempty"`
	ExternalID string                    `yaml:"external_id,omitempty"`
	Tags       []string                  `yaml:"tags,omitempty"`
	CreatedAt  *time.Time                `yaml:"created_at,omitempty"`
	UpdatedAt  *time.Time                `yaml:"updated_at,omitempty"`
	Due        *DueDate                  `yaml:"due,omitempty"`
	Parent     string                    `yaml:"parent,omitempty"`
	Blocking   []string                  `yaml:"blocking,omitempty"`
	BlockedBy  []string                  `yaml:"blocked_by,omitempty"`
	Sync       map[string]map[string]any `yaml:"sync,omitempty"`
}

// Render serializes the issue back to markdown with YAML front matter.
func (b *Issue) Render() ([]byte, error) {
	fm := renderFrontMatter{
		Title:      b.Title,
		Status:     b.Status,
		Type:       b.Type,
		Priority:   b.Priority,
		Milestone:  b.Milestone,
		ExternalID: b.ExternalID,
		Tags:       b.Tags,
		CreatedAt:  b.CreatedAt,
		UpdatedAt:  b.UpdatedAt,
		Due:        b.Due,
		Parent:     b.Parent,
		Blocking:   b.Blocking,
		BlockedBy:  b.BlockedBy,
		Sync:       b.Sync,
	}

	fmBytes, err := yaml.Marshal(&fm)
	if err != nil {
		return nil, fmt.Errorf("marshaling front matter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	if b.ID != "" {
		buf.WriteString("# ")
		buf.WriteString(b.ID)
		buf.WriteString("\n")
	}
	buf.Write(fmBytes)
	buf.WriteString("---\n")
	if b.Body != "" {
		// Only add newline separator if body doesn't already start with one
		if !strings.HasPrefix(b.Body, "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString(b.Body)
		// Ensure trailing newline if body doesn't end with one
		if !strings.HasSuffix(b.Body, "\n") {
			buf.WriteString("\n")
		}
	} else {
		// Even without body, add trailing newline for POSIX compliance
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// ETag returns a hash of the issue's rendered content for optimistic concurrency control.
// Uses FNV-1a 64-bit hash, producing a 16-character hex string.
// Returns "0000000000000000" if rendering fails (should never happen for valid issues).
func (b *Issue) ETag() string {
	content, err := b.Render()
	if err != nil {
		// Return a sentinel value that will never match a real ETag,
		// ensuring validation will fail rather than silently passing.
		return "0000000000000000"
	}
	h := fnv.New64a()
	h.Write(content) //nolint:gosec // hash.Write never returns error
	return hex.EncodeToString(h.Sum(nil))
}

// GithubIssueNumber returns the GitHub issue number from the sync metadata,
// or 0 if not set.
func (b *Issue) GithubIssueNumber() int {
	if b.Sync == nil {
		return 0
	}
	gh, ok := b.Sync["github"]
	if !ok {
		return 0
	}
	numStr, ok := gh["issue_number"]
	if !ok {
		return 0
	}
	s, ok := numStr.(string)
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// MarshalJSON implements json.Marshaler to include computed etag and github_issue fields.
func (b *Issue) MarshalJSON() ([]byte, error) {
	type IssueAlias Issue // Avoid infinite recursion

	ghIssue := b.GithubIssueNumber()
	if ghIssue == 0 {
		return json.Marshal(&struct {
			*IssueAlias
			GithubIssue *int   `json:"github_issue"`
			ETag        string `json:"etag"`
		}{
			IssueAlias:  (*IssueAlias)(b),
			GithubIssue: nil,
			ETag:        b.ETag(),
		})
	}
	return json.Marshal(&struct {
		*IssueAlias
		GithubIssue int    `json:"github_issue"`
		ETag        string `json:"etag"`
	}{
		IssueAlias:  (*IssueAlias)(b),
		GithubIssue: ghIssue,
		ETag:        b.ETag(),
	})
}

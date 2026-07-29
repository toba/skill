package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/toba/jig/internal/constants"
	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the name of the config file at project root.
	ConfigFileName = constants.ConfigFileName
	// LegacyTobaConfigFileName is the old .toba.yaml config file name
	LegacyTobaConfigFileName = ".toba.yaml"
	// LegacyConfigFileName is the old config file name (pre-migration)
	LegacyConfigFileName = ".todo.yml"
	// DefaultDataPath is the default directory for storing issues
	DefaultDataPath = ".issues"
)

// JigConfig is the top-level wrapper for the .jig.yaml file format.
// The todo configuration lives under the "todo" key to support
// a shared config format where multiple jig tools each have their own section.
type JigConfig struct {
	Todo Config `yaml:"todo"`
}

// Status name constants.
const (
	StatusInProgress = "in-progress"
	StatusReview     = "review"
	StatusReady      = "ready"
	StatusDraft      = "draft"
	StatusDeferred   = "deferred"
	StatusCompleted  = "completed"
	StatusScrapped   = "scrapped"
)

// MandatoryStatuses are statuses that cannot be disabled via the
// `todo.statuses` map in .jig.yaml. Every project must support these
// because they anchor the open/closed lifecycle.
var MandatoryStatuses = []string{StatusReady, StatusCompleted}

// Type name constants.
const (
	TypeMilestone = "milestone"
	TypeEpic      = "epic"
	TypeBug       = "bug"
	TypeFeature   = "feature"
	TypeTask      = "task"
)

// Priority name constants.
const (
	PriorityCritical = "critical"
	PriorityHigh     = "high"
	PriorityNormal   = "normal"
	PriorityLow      = "low"
	PriorityDeferred = "deferred"
)

// Sort order constants.
const (
	SortDefault = "default"
)

// DefaultStatuses defines the hardcoded status configuration.
// Statuses are not configurable - they are hardcoded like types.
// Order determines sort priority: in-progress first (active work), then review, ready, draft, and done states last.
var DefaultStatuses = []StatusConfig{
	{Name: StatusInProgress, Color: "yellow", Description: "Currently being worked on"},
	{Name: StatusReview, Color: "cyan", Description: "Code complete, awaiting evaluation"},
	{Name: StatusReady, Color: "green", Description: "Ready to be worked on"},
	{Name: StatusDraft, Color: "blue", Description: "Needs refinement before it can be worked on"},
	{Name: StatusDeferred, Color: "pink", Description: "Parked pending further consideration; concerns must be resolved before work can resume"},
	{Name: StatusCompleted, Color: "gray", Archive: true, Description: "Finished successfully"},
	{Name: StatusScrapped, Color: "gray", Archive: true, Description: "Will not be done"},
}

// DefaultTypes defines the default type configuration.
//
// NOTE: "milestone" is intentionally NOT an issue type. Milestones are first-class
// entities (see internal/todo/issue.Milestone); the TypeMilestone constant is retained
// only for backward compatibility with legacy issues and the `milestone migrate` command.
var DefaultTypes = []TypeConfig{
	{Name: TypeEpic, Color: "purple", Description: "A thematic container for related work; should have child issues, not be worked on directly"},
	{Name: TypeBug, Color: "red", Description: "Something that is broken and needs fixing"},
	{Name: TypeFeature, Color: "green", Description: "A user-facing capability or enhancement"},
	{Name: TypeTask, Color: "blue", Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
}

// DefaultPriorities defines the hardcoded priority configuration.
// Priorities are ordered from highest to lowest urgency.
var DefaultPriorities = []PriorityConfig{
	{Name: PriorityCritical, Color: "red", Description: "Urgent, blocking work. When possible, address immediately"},
	{Name: PriorityHigh, Color: "yellow", Description: "Important, should be done before normal work"},
	{Name: PriorityNormal, Color: "white", Description: "Standard priority"},
	{Name: PriorityLow, Color: "gray", Description: "Less important, can be delayed"},
	{Name: PriorityDeferred, Color: "gray", Description: "Explicitly pushed back, avoid doing unless necessary"},
}

// StatusConfig defines a single status with its display color.
type StatusConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Archive     bool   `yaml:"archive,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// TypeConfig defines a single issue type with its display color.
type TypeConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// PriorityConfig defines a single priority level with its display color.
type PriorityConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// TagConfig defines a project tag with an optional description.
type TagConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// Config holds the todo configuration.
// Note: Statuses are no longer stored in config - they are hardcoded like types.
type Config struct {
	// Path is the path to the issues directory (relative to config file location)
	Path           string      `yaml:"path,omitempty"`
	DefaultStatus  string      `yaml:"default_status,omitempty"`
	DefaultType    string      `yaml:"default_type,omitempty"`
	DefaultSort    string      `yaml:"default_sort,omitempty"`
	Editor         string      `yaml:"editor,omitempty"`
	RequireIfMatch bool        `yaml:"require_if_match,omitempty"`
	Tags           []TagConfig `yaml:"tags,omitempty"`
	// ExtraStatuses additively opts non-mandatory statuses into this project.
	// The map is purely additive: an entry of `true` enables that status; an
	// entry of `false` or a missing entry leaves the status disabled.
	// Statuses listed in MandatoryStatuses are always enabled regardless of
	// what this map says.
	ExtraStatuses map[string]bool           `yaml:"extra_statuses,omitempty"`
	Sync          map[string]map[string]any `yaml:"sync,omitempty"`

	// configDir is the directory containing the config file (not serialized)
	// Used to resolve relative paths
	configDir string `yaml:"-"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Path:          DefaultDataPath,
		DefaultStatus: StatusReady,
		DefaultType:   TypeTask,
	}
}

// FindConfig searches upward from the given directory for a .jig.yaml config file,
// falling back to .toba.yaml, then the legacy .todo.yml. If only a .todo.yml is found,
// it is automatically migrated to .jig.yaml (written in the new format, old file removed).
// Returns the absolute path to the config file, or empty string if not found.
func FindConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		// Check .jig.yaml first
		newPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(newPath); err == nil {
			return newPath, nil
		}

		// Check .toba.yaml as legacy fallback
		tobaPath := filepath.Join(dir, LegacyTobaConfigFileName)
		if _, err := os.Stat(tobaPath); err == nil {
			return tobaPath, nil
		}

		// Check .todo.yml as oldest legacy fallback
		legacyPath := filepath.Join(dir, LegacyConfigFileName)
		if _, err := os.Stat(legacyPath); err == nil {
			// Auto-migrate legacy config to new format
			migrated, migrateErr := migrateLegacyConfig(legacyPath, newPath)
			if migrateErr != nil {
				return "", fmt.Errorf("migrating %s to %s: %w", LegacyConfigFileName, ConfigFileName, migrateErr)
			}
			if migrated {
				return newPath, nil
			}
			// If migration failed silently, fall back to legacy
			return legacyPath, nil
		}

		// Check for common typo: .jig.yml instead of .jig.yaml
		typoPath := filepath.Join(dir, ".jig.yml")
		if _, err := os.Stat(typoPath); err == nil {
			return "", errors.New("found .jig.yml but jig expects .jig.yaml — please rename it")
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}

// legacyConfig is used to parse the old .todo.yml format which had
// an "issues:" top-level key containing the issue settings.
type legacyConfig struct {
	Issues struct {
		Path           string `yaml:"path,omitempty"`
		DefaultStatus  string `yaml:"default_status,omitempty"`
		DefaultType    string `yaml:"default_type,omitempty"`
		DefaultSort    string `yaml:"default_sort,omitempty"`
		Editor         string `yaml:"editor,omitempty"`
		RequireIfMatch bool   `yaml:"require_if_match,omitempty"`
	} `yaml:"issues"`
	Sync map[string]map[string]any `yaml:"sync,omitempty"`
}

// migrateLegacyConfig reads a legacy .todo.yml, wraps it in the JigConfig
// format, writes .jig.yaml, and removes the old file.
// Returns true if migration was performed successfully.
func migrateLegacyConfig(legacyPath, newPath string) (bool, error) {
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return false, err
	}

	// Parse the legacy flat format (has "issues:" top-level key)
	var legacy legacyConfig
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		return false, err
	}

	// Flatten into the new Config format
	cfg := Config{
		Path:           legacy.Issues.Path,
		DefaultStatus:  legacy.Issues.DefaultStatus,
		DefaultType:    legacy.Issues.DefaultType,
		DefaultSort:    legacy.Issues.DefaultSort,
		Editor:         legacy.Issues.Editor,
		RequireIfMatch: legacy.Issues.RequireIfMatch,
		Sync:           legacy.Sync,
	}

	// Write in new JigConfig wrapper format
	wrapper := JigConfig{Todo: cfg}
	out, err := yaml.Marshal(&wrapper)
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(newPath, out, 0644); err != nil {
		return false, err
	}

	// Remove legacy file
	if err := os.Remove(legacyPath); err != nil { //nolint:nilerr // non-fatal: new file written, old file just lingers
		return true, nil //nolint:nilerr // non-fatal: new file written, old file just lingers
	}

	return true, nil
}

// Load reads configuration from the given config file path.
// Returns default config if the file doesn't exist.
// Handles both the new .jig.yaml format (with "todo:" wrapper) and
// the legacy .todo.yml flat format.
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	var cfg Config
	if isLegacyConfig(configPath) {
		// Legacy .todo.yml: has "issues:" top-level key
		var legacy legacyConfig
		if err := yaml.Unmarshal(data, &legacy); err != nil {
			return nil, err
		}
		cfg = Config{
			Path:           legacy.Issues.Path,
			DefaultStatus:  legacy.Issues.DefaultStatus,
			DefaultType:    legacy.Issues.DefaultType,
			DefaultSort:    legacy.Issues.DefaultSort,
			Editor:         legacy.Issues.Editor,
			RequireIfMatch: legacy.Issues.RequireIfMatch,
			Sync:           legacy.Sync,
		}
	} else {
		// .jig.yaml or .toba.yaml: wrapped in "todo:" key
		var wrapper JigConfig
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
		cfg = wrapper.Todo
	}

	// Store the config directory for resolving relative paths
	cfg.configDir = filepath.Dir(configPath)

	// Apply defaults for missing values
	cfg.Path = cmp.Or(cfg.Path, DefaultDataPath)
	cfg.DefaultStatus = cmp.Or(cfg.DefaultStatus, StatusReady)
	cfg.DefaultType = cmp.Or(cfg.DefaultType, TypeTask)

	return &cfg, nil
}

// isLegacyConfig returns true if the given path is a legacy .todo.yml file.
func isLegacyConfig(configPath string) bool {
	return filepath.Base(configPath) == LegacyConfigFileName
}

// HasTodoSection reports whether the config file at the given path actually
// contains todo configuration. This is distinct from Load, which always
// returns a defaulted Config even when the file has no todo section — callers
// (e.g. `jig prime`) use this to decide whether todo functionality is
// relevant to the project at all.
//
// The legacy .todo.yml format is todo-only, so it always counts as present.
// For .jig.yaml / .toba.yaml the top-level "todo:" key must exist.
func HasTodoSection(configPath string) bool {
	if configPath == "" {
		return false
	}
	if isLegacyConfig(configPath) {
		return true
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	var top map[string]yaml.Node
	if err := yaml.Unmarshal(data, &top); err != nil {
		return false
	}
	_, ok := top["todo"]
	return ok
}

// LoadFromDirectory finds and loads the config file by searching upward from the given directory.
// If no config file is found, returns a default config anchored at the given directory.
func LoadFromDirectory(startDir string) (*Config, error) {
	configPath, err := FindConfig(startDir)
	if err != nil {
		return nil, err
	}

	if configPath == "" {
		// No config found, return default anchored at startDir
		cfg := Default()
		cfg.configDir = startDir
		return cfg, nil
	}

	return Load(configPath)
}

// ResolveDataPath returns the absolute path to the issues directory.
func (c *Config) ResolveDataPath() string {
	if filepath.IsAbs(c.Path) {
		return c.Path
	}
	if c.configDir == "" {
		// Fallback: use current directory
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, c.Path)
	}
	return filepath.Join(c.configDir, c.Path)
}

// ConfigDir returns the directory containing the config file.
func (c *Config) ConfigDir() string {
	return c.configDir
}

// SetConfigDir sets the config directory (for testing or when creating new configs).
func (c *Config) SetConfigDir(dir string) {
	c.configDir = dir
}

// Save writes the todo configuration to .jig.yaml, preserving other sections.
// Uses yaml.v3 Node API to do a partial update of just the "todo:" key.
// If configDir is set, saves to that directory; otherwise saves to the given directory.
func (c *Config) Save(dir string) error {
	targetDir := c.configDir
	if targetDir == "" {
		targetDir = dir
	}
	path := filepath.Join(targetDir, ConfigFileName)

	// Encode the new todo config as a YAML node
	var newTodo yaml.Node
	if err := newTodo.Encode(c); err != nil {
		return fmt.Errorf("encoding todo config: %w", err)
	}

	// Try to load existing document to preserve other sections
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		var root yaml.Node
		if err := yaml.Unmarshal(data, &root); err == nil {
			if replaceOrAppendKey(&root, "todo", &newTodo) {
				out, err := yaml.Marshal(&root)
				if err != nil {
					return fmt.Errorf("marshaling document: %w", err)
				}
				return os.WriteFile(path, out, 0o644)
			}
		}
	}

	// No existing file or parse error — write fresh
	wrapper := JigConfig{Todo: *c}
	out, err := yaml.Marshal(&wrapper)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// replaceOrAppendKey finds the "key" in a YAML mapping and replaces its value,
// or appends a new key-value pair. Returns true on success.
func replaceOrAppendKey(root *yaml.Node, key string, value *yaml.Node) bool {
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = value
			return true
		}
	}
	// Key not found — append
	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		value,
	)
	return true
}

// named is a constraint for config types that have a Name field.
type named interface {
	StatusConfig | TypeConfig | PriorityConfig | TagConfig
}

func tagName(t *TagConfig) string { return t.Name }

// GetTag returns the TagConfig for a given tag name (case-insensitive), or nil if not found.
func (c *Config) GetTag(name string) *TagConfig {
	lower := strings.ToLower(name)
	for i := range c.Tags {
		if strings.ToLower(c.Tags[i].Name) == lower {
			return &c.Tags[i]
		}
	}
	return nil
}

// TagNames returns a slice of tag names from the tag registry.
func (c *Config) TagNames() []string {
	return configNames(c.Tags, tagName)
}

// configNames extracts name strings from a slice of config items.
func configNames[T named](items []T, getName func(*T) string) []string {
	names := make([]string, len(items))
	for i := range items {
		names[i] = getName(&items[i])
	}
	return names
}

// configList returns a comma-separated list of names.
func configList[T named](items []T, getName func(*T) string) string {
	return strings.Join(configNames(items, getName), ", ")
}

// configFind returns a pointer to the item with the given name, or nil.
func configFind[T named](items []T, name string, getName func(*T) string) *T {
	for i := range items {
		if getName(&items[i]) == name {
			return &items[i]
		}
	}
	return nil
}

// configIsValid returns true if name matches any item.
func configIsValid[T named](items []T, name string, getName func(*T) string) bool {
	return slices.ContainsFunc(items, func(item T) bool {
		return getName(&item) == name
	})
}

func statusName(s *StatusConfig) string     { return s.Name }
func typeName(t *TypeConfig) string         { return t.Name }
func priorityName(p *PriorityConfig) string { return p.Name }

// IsValidStatus returns true if the status is a valid hardcoded status.
// Note: this does not check whether the status is enabled for the current
// project — use IsStatusEnabled for that.
func (c *Config) IsValidStatus(status string) bool {
	return configIsValid(DefaultStatuses, status, statusName)
}

// IsMandatoryStatus reports whether a status cannot be disabled via config.
func IsMandatoryStatus(status string) bool {
	return slices.Contains(MandatoryStatuses, status)
}

// IsStatusEnabled reports whether a status is valid AND enabled for this project.
// Mandatory statuses are always enabled. All other statuses are disabled
// unless explicitly opted into via `todo.extra_statuses` (with value `true`).
// The map is purely additive — `false` and missing entries both mean disabled.
func (c *Config) IsStatusEnabled(status string) bool {
	if !c.IsValidStatus(status) {
		return false
	}
	if IsMandatoryStatus(status) {
		return true
	}
	return c.ExtraStatuses[status]
}

// EnabledStatusNames returns the names of statuses enabled for this project,
// preserving the order of DefaultStatuses.
func (c *Config) EnabledStatusNames() []string {
	names := make([]string, 0, len(DefaultStatuses))
	for _, s := range DefaultStatuses {
		if c.IsStatusEnabled(s.Name) {
			names = append(names, s.Name)
		}
	}
	return names
}

// EnabledStatusList returns a comma-separated list of enabled status names.
func (c *Config) EnabledStatusList() string {
	return strings.Join(c.EnabledStatusNames(), ", ")
}

// StatusList returns a comma-separated list of valid statuses.
func (c *Config) StatusList() string {
	return configList(DefaultStatuses, statusName)
}

// StatusNames returns a slice of valid status names.
func (c *Config) StatusNames() []string {
	return configNames(DefaultStatuses, statusName)
}

// GetStatus returns the StatusConfig for a given status name, or nil if not found.
func (c *Config) GetStatus(name string) *StatusConfig {
	return configFind(DefaultStatuses, name, statusName)
}

// GetDefaultStatus returns the default status name for new issues.
func (c *Config) GetDefaultStatus() string {
	return cmp.Or(c.DefaultStatus, StatusReady)
}

// GetDefaultType returns the default type name for new issues.
func (c *Config) GetDefaultType() string {
	return c.DefaultType
}

// GetEditor returns the configured editor command, or empty string if unset.
func (c *Config) GetEditor() string {
	return c.Editor
}

// GetDefaultSort returns the default sort order for the TUI.
// Returns "default" if not set.
func (c *Config) GetDefaultSort() string {
	return cmp.Or(c.DefaultSort, SortDefault)
}

// IsArchiveStatus returns true if the given status is marked for archiving.
// Statuses are hardcoded and not configurable.
func (c *Config) IsArchiveStatus(name string) bool {
	if s := c.GetStatus(name); s != nil {
		return s.Archive
	}
	return false
}

// IsCompleteStatus reports whether a status represents finished or closed work
// for the purpose of parent-completion gating: completed, scrapped, or deferred.
// A parent issue may only enter one of these statuses once all of its children
// are also in a complete status.
func IsCompleteStatus(name string) bool {
	switch name {
	case StatusCompleted, StatusScrapped, StatusDeferred:
		return true
	}
	return false
}

// GetType returns the TypeConfig for a given type name, or nil if not found.
func (c *Config) GetType(name string) *TypeConfig {
	return configFind(DefaultTypes, name, typeName)
}

// TypeNames returns a slice of valid type names.
func (c *Config) TypeNames() []string {
	return configNames(DefaultTypes, typeName)
}

// IsValidType returns true if the type is a valid hardcoded type.
func (c *Config) IsValidType(name string) bool {
	return configIsValid(DefaultTypes, name, typeName)
}

// TypeList returns a comma-separated list of valid types.
func (c *Config) TypeList() string {
	return configList(DefaultTypes, typeName)
}

// IssueColors holds resolved color information for rendering an issue
type IssueColors struct {
	StatusColor   string
	TypeColor     string
	PriorityColor string
	IsArchive     bool
}

// GetIssueColors returns the resolved colors for an issue based on its status, type, and priority.
func (c *Config) GetIssueColors(status, typeName, priority string) IssueColors {
	colors := IssueColors{
		StatusColor:   "gray",
		TypeColor:     "",
		PriorityColor: "",
		IsArchive:     false,
	}

	if statusCfg := c.GetStatus(status); statusCfg != nil {
		colors.StatusColor = statusCfg.Color
	}
	colors.IsArchive = c.IsArchiveStatus(status)

	if typeCfg := c.GetType(typeName); typeCfg != nil {
		colors.TypeColor = typeCfg.Color
	}

	if priorityCfg := c.GetPriority(priority); priorityCfg != nil {
		colors.PriorityColor = priorityCfg.Color
	}

	return colors
}

// GetPriority returns the PriorityConfig for a given priority name, or nil if not found.
func (c *Config) GetPriority(name string) *PriorityConfig {
	return configFind(DefaultPriorities, name, priorityName)
}

// PriorityNames returns a slice of valid priority names in order from highest to lowest.
func (c *Config) PriorityNames() []string {
	return configNames(DefaultPriorities, priorityName)
}

// IsValidPriority returns true if the priority is a valid hardcoded priority.
// Empty string is valid (means no priority set).
func (c *Config) IsValidPriority(priority string) bool {
	if priority == "" {
		return true
	}
	return configIsValid(DefaultPriorities, priority, priorityName)
}

// SyncConfig returns the configuration data for a named sync integration,
// or nil if the integration has no configuration.
func (c *Config) SyncConfig(name string) map[string]any {
	if c.Sync == nil {
		return nil
	}
	return c.Sync[name]
}

// PriorityList returns a comma-separated list of valid priorities.
func (c *Config) PriorityList() string {
	return configList(DefaultPriorities, priorityName)
}

// DefaultStatusNames returns the names of all default statuses.
func DefaultStatusNames() []string {
	return configNames(DefaultStatuses, statusName)
}

// DefaultTypeNames returns the names of all default types.
func DefaultTypeNames() []string {
	return configNames(DefaultTypes, typeName)
}

// DefaultPriorityNames returns the names of all default priorities.
func DefaultPriorityNames() []string {
	return configNames(DefaultPriorities, priorityName)
}

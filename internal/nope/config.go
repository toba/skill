package nope

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/toba/jig/internal/constants"
	"gopkg.in/yaml.v3"
)

// DefaultToolName is the tool name used when no tool is specified.
const DefaultToolName = "Bash"

// Builtin rule names.
const (
	BuiltinPipe           = "pipe"
	BuiltinChained        = "chained"
	BuiltinRedirect       = "redirect"
	BuiltinSubshell       = "subshell"
	BuiltinCredentialRead = "credential-read" //nolint:gosec // not a credential, just a category name
	BuiltinNetwork        = "network"
	BuiltinExfiltration   = "exfiltration"
	BuiltinEnvHijack      = "env-hijack"
	BuiltinInlineSecrets  = "inline-secrets"
	BuiltinVarCommand     = "var-command"
)

// NopeConfig is the nope section of .jig.yaml.
type NopeConfig struct {
	Debug string    `yaml:"debug"` // path to debug log file (empty = disabled)
	Rules []RuleDef `yaml:"rules"`
}

// RuleDef is a single rule from the config file.
type RuleDef struct {
	Name    string   `yaml:"name"`
	Pattern string   `yaml:"pattern"` // regex, mutually exclusive with Builtin
	Builtin string   `yaml:"builtin"`
	Message string   `yaml:"message"`
	Tools   []string `yaml:"tools"` // tool names this rule applies to; empty defaults to ["Bash"]
}

// CompiledRule is a rule ready for matching.
type CompiledRule struct {
	Name      string
	Check     func(input string) bool
	ToolMatch func(toolName string) bool
	Message   string
}

// LoadConfig reads .jig.yaml, finds the nope: node, and decodes it into NopeConfig.
func LoadConfig(path string) (*NopeConfig, error) {
	data, err := os.ReadFile(path) //nolint:gosec // config path from trusted walk-up search
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// doc is a Document node; its first child is the mapping.
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("config %s: empty or invalid document", path)
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config %s: expected mapping at top level", path)
	}

	// Find the "nope" key in the mapping.
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == "nope" {
			var cfg NopeConfig
			if err := mapping.Content[i+1].Decode(&cfg); err != nil {
				return nil, fmt.Errorf("decoding nope section in %s: %w", path, err)
			}
			return &cfg, nil
		}
	}

	return nil, fmt.Errorf("config %s: no 'nope' section found", path)
}

// FindConfigPath locates .jig.yaml by walking up from cwd.
func FindConfigPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for {
		candidate := filepath.Join(dir, constants.ConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("no config found (create .jig.yaml with a nope: section or run `jigo nope init`)")
}

// FindAndLoadConfig locates and parses the nope section of .jig.yaml.
// Returns the config and the project root (directory containing .jig.yaml).
func FindAndLoadConfig() (*NopeConfig, string, error) {
	path, err := FindConfigPath()
	if err != nil {
		return nil, "", err
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, "", err
	}
	// Project root is the directory containing .jig.yaml.
	root := filepath.Dir(path)
	return cfg, root, nil
}

// buildToolMatcher returns a function that reports whether a given tool name
// matches the rule's tool scope. An empty tools list defaults to Bash-only.
// A single entry of "*" matches all tools.
func buildToolMatcher(tools []string) func(string) bool {
	if len(tools) == 0 {
		return func(name string) bool { return name == DefaultToolName }
	}
	if slices.Contains(tools, "*") {
		return func(string) bool { return true }
	}
	set := make(map[string]struct{}, len(tools))
	for _, t := range tools {
		set[t] = struct{}{}
	}
	return func(name string) bool {
		_, ok := set[name]
		return ok
	}
}

// CompileRules converts RuleDefs into CompiledRules, failing fast on bad patterns.
func CompileRules(defs []RuleDef) ([]CompiledRule, error) {
	rules := make([]CompiledRule, 0, len(defs))
	for _, d := range defs {
		if d.Pattern != "" && d.Builtin != "" {
			return nil, fmt.Errorf("rule %q: pattern and builtin are mutually exclusive", d.Name)
		}
		if d.Pattern == "" && d.Builtin == "" {
			return nil, fmt.Errorf("rule %q: must have pattern or builtin", d.Name)
		}
		if d.Message == "" {
			return nil, fmt.Errorf("rule %q: message is required", d.Name)
		}

		var check func(string) bool
		if d.Builtin != "" {
			// Builtins are Bash-only — reject non-Bash tool scoping
			for _, t := range d.Tools {
				if t != DefaultToolName {
					return nil, fmt.Errorf("rule %q: builtin rules only support Bash tool, got %q", d.Name, t)
				}
			}
			switch d.Builtin {
			case BuiltinPipe:
				check = CheckPipe
			case BuiltinChained:
				check = CheckChained
			case BuiltinRedirect:
				check = CheckRedirect
			case BuiltinSubshell:
				check = CheckSubshell
			case BuiltinCredentialRead:
				check = CheckCredentialRead
			case BuiltinNetwork:
				check = CheckNetwork
			case BuiltinExfiltration:
				check = CheckExfiltration
			case BuiltinEnvHijack:
				check = CheckEnvHijack
			case BuiltinInlineSecrets:
				check = CheckInlineSecrets
			case BuiltinVarCommand:
				check = CheckVarCommand
			default:
				return nil, fmt.Errorf("rule %q: unknown builtin %q", d.Name, d.Builtin)
			}
		} else {
			// Prepend (?s) so . matches newlines — multiline commands
			// (e.g. shell line continuations) should not escape pattern rules.
			re, err := regexp.Compile("(?s)" + d.Pattern)
			if err != nil {
				hint := ""
				if strings.Contains(d.Pattern, "(?<") || strings.Contains(d.Pattern, "(?=") || strings.Contains(d.Pattern, "(?!") {
					hint = " (Go regex uses RE2, which does not support lookahead/lookbehind — rewrite using negated character classes or word boundaries)"
				}
				return nil, fmt.Errorf("rule %q: bad pattern: %w%s", d.Name, err, hint)
			}
			check = re.MatchString
		}

		rules = append(rules, CompiledRule{
			Name:      d.Name,
			Check:     check,
			ToolMatch: buildToolMatcher(d.Tools),
			Message:   d.Message,
		})
	}
	return rules, nil
}

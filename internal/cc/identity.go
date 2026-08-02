package cc

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// IdentityFields are the keys in .claude.json bound to a specific account or
// install. They must never be shared or copied between aliases: each account
// regenerates its own on first launch/login. Seeding a new alias's
// .claude.json from another account's file strips these so two distinct
// accounts don't end up reporting the same identifiers — which reads as
// cross-account linkage. This is account isolation, NOT policy evasion: the
// goal is for each account to look like the separate install it actually is.
var IdentityFields = []string{
	"machineID",
	"userID",
	"oauthAccount",
}

// identityScalars are the stable string identifiers `cc doctor` compares
// across aliases to catch accidental cross-account linkage.
var identityScalars = []string{"machineID", "userID"}

// scrubIdentity removes IdentityFields from a .claude.json byte payload and
// returns re-serialized JSON. An empty payload is treated as an empty object.
func scrubIdentity(data []byte) ([]byte, error) {
	m := map[string]json.RawMessage{}
	if strings.TrimSpace(string(data)) != "" {
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
	}
	for _, k := range IdentityFields {
		delete(m, k)
	}
	return json.MarshalIndent(m, "", "  ")
}

// IdentityCollision reports two or more aliases sharing the same identity
// value — the fingerprint that makes distinct accounts look like one install.
type IdentityCollision struct {
	Field   string   `json:"field"`
	Value   string   `json:"value"`
	Aliases []string `json:"aliases"`
}

// CheckIdentity scans every alias's .claude.json for shared machineID/userID
// values. Any value held by more than one alias is reported as a collision.
// Missing or unreadable files are skipped (nothing to compare).
func CheckIdentity(c *Config) ([]IdentityCollision, error) {
	// field -> value -> aliases holding it
	seen := make(map[string]map[string][]string, len(identityScalars))
	for _, field := range identityScalars {
		seen[field] = map[string][]string{}
	}

	for _, name := range c.Names() {
		a := c.Aliases[name]
		data, err := os.ReadFile(filepath.Join(a.Path, ".claude.json"))
		if err != nil {
			continue
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		for _, field := range identityScalars {
			raw, ok := m[field]
			if !ok {
				continue
			}
			var v string
			if err := json.Unmarshal(raw, &v); err != nil || v == "" {
				continue
			}
			seen[field][v] = append(seen[field][v], name)
		}
	}

	var out []IdentityCollision
	for _, field := range identityScalars {
		for v, aliases := range seen[field] {
			if len(aliases) < 2 {
				continue
			}
			slices.Sort(aliases)
			out = append(out, IdentityCollision{Field: field, Value: v, Aliases: aliases})
		}
	}
	slices.SortFunc(out, func(a, b IdentityCollision) int {
		return cmp.Or(
			cmp.Compare(a.Field, b.Field),
			cmp.Compare(a.Value, b.Value),
		)
	})
	return out, nil
}

---
# 26k-met
title: Strip account identity fields when seeding a new cc alias's .claude.json
status: completed
type: bug
priority: normal
created_at: 2026-08-02T17:28:24Z
updated_at: 2026-08-02T17:35:38Z
---

## Problem

`jig cc` runs multiple Claude Code profiles from one machine on a "share tooling, isolate identity" model: shared config (agents/skills/commands/CLAUDE.md/projects) is symlinked from one source dir, while each alias keeps its own real copies of the `DefaultPrivate` files.

The bug: `SeedClaudeJSON` (internal/cc/symlinks.go) copies the source account's `.claude.json` verbatim into a fresh alias so Claude can launch. But `.claude.json` embeds stable identity fields — `machineID`, `userID`, `oauthAccount`, and it sits alongside `statsig`/`telemetry` device IDs. On login, `oauthAccount` is overwritten but `machineID`/`userID` persist. Result: two distinct paid accounts report the identical machineID+userID from one install, which reads as cross-account linkage / fraud to Anthropic.

This is an isolation defect, NOT policy evasion — the fix makes each account keep its own distinct identity (what a separate install would do).

## Todo

- [x] Write a failing test: seeding an alias's .claude.json from a source file must NOT carry over machineID, userID, oauthAccount
- [x] Fix SeedClaudeJSON to strip identity fields (machineID, userID, oauthAccount) when seeding — write a scrubbed .claude.json so Claude regenerates its own on first login
- [x] Ensure statsig/telemetry dirs are never seeded/copied from another account (verify CopyPrivateFiles + init flow)
- [x] Add a `jig cc doctor` check that flags any two aliases sharing machineID or userID
- [x] Run scripts/lint.sh; go test ./...; go vet ./...

## Notes

CLAUDE.md was updated to document cc intent and codify the share-tooling / isolate-identity invariant. This issue implements the enforcement in code.

## Summary of Changes

- **internal/cc/identity.go** (new): `IdentityFields` (`machineID`, `userID`, `oauthAccount`) as the canonical list of account/install-bound keys; `scrubIdentity()` removes them from a `.claude.json` payload; `CheckIdentity()` scans all aliases and reports `IdentityCollision`s where two+ aliases share a `machineID`/`userID` (the cross-account fingerprint).
- **internal/cc/symlinks.go**: `SeedClaudeJSON` now reads the source `.claude.json`, scrubs identity fields, and writes the result — instead of a verbatim `copyFile`. Fresh aliases created by `cc add` / `cc login` no longer inherit the source account's identity. Doc comment updated to state the isolation intent.
- **cmd/cc_doctor.go**: `jig cc doctor` now runs `CheckIdentity` in addition to symlink health, prints any collisions (identifier value masked), exits non-zero when found, and includes `identity_collisions` in `--json` output. Added `maskID` helper.
- **internal/cc/symlinks_test.go**: added `TestSeedClaudeJSONStripsIdentity` (identity stripped, benign fields preserved), `TestCheckIdentityDetectsSharedIDs`, and `TestCheckIdentityCleanWhenDistinct`.

### Verification of statsig/telemetry (checklist item 3)

`CopyPrivateFiles` copies the full private list and is used only by (a) `cc clone` — an explicit same-account duplicate — and (b) `Init` for already-distinct detected `~/.claude-*` dirs, where each dir's own data moves into its own managed alias (same account, no cross-contamination). The fresh-alias paths (`cc add`, `cc login`) never copy `statsig`/`telemetry`, so those regenerate per account. The new `cc doctor` collision check is the backstop that catches any accidental sharing regardless of path.

Build, `go vet ./...`, `go test ./...`, and `scripts/lint.sh` all clean.

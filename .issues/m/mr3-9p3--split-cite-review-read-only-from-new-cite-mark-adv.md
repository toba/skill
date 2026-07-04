---
# mr3-9p3
title: Split cite review (read-only) from new cite mark (advances marker)
status: completed
type: bug
priority: normal
created_at: 2026-07-04T23:27:11Z
updated_at: 2026-07-04T23:30:52Z
sync:
    github:
        issue_number: "124"
        synced_at: "2026-07-04T23:33:10Z"
---

`jig cite review` prints changes AND advances `last_checked_sha`/`last_checked_date` in the same call (cmd/check.go:116-137). Agents treat review as idempotent and re-run it, or lose the output to a pipe; on the second run the marker has moved to HEAD so they get an empty result and fall back to fetching from GitHub directly.

Fix: make `review` read-only/repeatable (never advances) and add `jig cite mark [source]` to advance the marker explicitly. Update the cite skill and CLAUDE.md design notes.

- [ ] Write failing test
- [ ] Make review read-only (drop the Save block)
- [ ] Add `jig cite mark [source]` command
- [ ] Update SKILL.md + CLAUDE.md



## Summary of Changes

- `cmd/check.go`: `review` (`runCheck`) is now read-only — dropped the block that advanced markers and saved config; simplified `checkResult` to just the display field.
- `cmd/cite_mark.go` (new): `jig cite mark [source]` command + `markSource` helper. Fetches current HEAD (or latest release tag+SHA for release-tracked sources), advances `last_checked_sha`/`last_checked_date`/`last_checked_tag`, saves config once. Supports `--json`.
- `cmd/cite_mark_test.go` (new): failing-first tests for `markSource` (commit + release paths) and mark command registration.
- Docs: updated cite SKILL.md (Review now read-only/repeatable; Mark = explicit step), CLAUDE.md architecture + design notes, README.md.

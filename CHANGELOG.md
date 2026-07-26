# Changelog

## Week of Jul 19 – Jul 25, 2026

### ✨ Features

- Issues can now store an optional `external_id` in frontmatter referencing another system (e.g. a Jira key); wired through the model, GraphQL schema/resolvers, the `--external-id` flag on `create`/`update`, and the `show`/TUI headers ([#127](https://github.com/toba/jig/issues/127))
- The TUI issue detail header now shows the created and updated dates as a muted line beneath the ID/status row ([#126](https://github.com/toba/jig/issues/126))

### 🗜️ Tweaks

- Update all Go dependencies to latest; pin the `bleve` stack to the versions `bleve/v2 v2.6.0` declares (a blanket upgrade broke compilation via a `roaring` `.Value` arity change) and regenerate the `gqlgen` output for v0.17.94 ([#125](https://github.com/toba/jig/issues/125))

## Week of Jun 28 – Jul 4, 2026

### 🐞 Fixes

- `--body-file -` and `--replace-body-file -` now read stdin like the inline `--body -` flags do; previously `resolveContent` treated `-` as a literal filename and failed with `reading file: open -: no such file or directory`, which repeatedly tripped up agents piping issue bodies in ([#123](https://github.com/toba/jig/issues/123))
- `jig cite review` is now read-only and repeatable instead of advancing `last_checked_sha`/`last_checked_date` as a side effect; a new `jig cite mark [source]` command records sources as reviewed, so agents can re-run review (or recover from output lost to a pipe) without silently skipping the upstream changes ([#124](https://github.com/toba/jig/issues/124))

## Week of Jun 21 – Jun 27, 2026

### ✨ Features

- Block moving a parent issue into a complete status (`completed`, `scrapped`, `deferred`) while any child is still active; `updateIssue` rejects the transition and names the blocking children, covering the CLI, GraphQL, and TUI ([#119](https://github.com/toba/jig/issues/119))

### 🐞 Fixes

- `jig todo show` no longer emits ANSI escape codes when stdout is piped; output now routes through a color-profile writer that strips color for non-TTY destinations (and honors `NO_COLOR`/`CLICOLOR`), so agents reading via `| cat` get clean text instead of falling back to the raw issue file ([#120](https://github.com/toba/jig/issues/120))

### 🗜️ Tweaks

- ClickUp sync no longer pays a fixed prefetch tax on every run; `SyncIssues` now skips the authorized-user, list, and space-tag prefetches on dry runs, fetches them only when actually needed (creates and tagged issues respectively), and runs the survivors concurrently, collapsing ~3 serial round-trips to ~1; the per-issue cost was already incremental, so this keeps a typical few-issue sync fast regardless of total project size ([#121](https://github.com/toba/jig/issues/121))

## Week of Jun 14 – Jun 20, 2026

### ✨ Features

- Add `jig todo comment <id> "..."`; a discoverable verb (agents reach for it by analogy with `gh`/`git`) that appends a note to an issue body via the same path as `update --append-body`, so etag checks, the `updated` timestamp, and sync all run; accepts `-` for stdin and `--json`; the agent guide and docs now steer agents to it and warn against editing `.issues/*.md` files directly ([#118](https://github.com/toba/jig/issues/118))

## Week of Jun 7 – Jun 13, 2026

### ✨ Features

- TUI filter matches partial issue IDs; typing `vfj` surfaces issue `vfj-jop`, case-insensitive substring matching across the title and ID (regression test added)

### 🐞 Fixes

- `jig cite review` no longer re-emits the latest release every run for `track: releases` sources; when `LastCheckedTag` matches `release.tag_name` the release is omitted from output while `last_checked_*` still refreshes

## Week of May 24 – May 30, 2026

### ✨ Features

- Add first-class milestones; milestone entities stored in `.issues/milestones/`, an `issue.milestone` reference field, `jig todo milestone` commands, GraphQL surface, TUI badge/picker/filter, GitHub milestone sync, and a `migrate` command that retires the legacy `milestone` issue type ([#108](https://github.com/toba/jig/issues/108))
- Render the milestone short name as a gray `<short>:` prefix glued to the front of the issue ID in TUI rows (including tree view); the ID stays purple ([#109](https://github.com/toba/jig/issues/109))
- Child issues now inherit their parent's milestone when none is set explicitly; applies on create with `--parent` and on reparent via update or the TUI parent picker

### 🐞 Fixes

- Render the milestone short-name prefix on child and linked issues in the TUI detail view, matching the main list rows
- Fix the TUI file watcher leaking milestone files into the issue list as empty-title tasks; route milestone-file events to the milestone map instead of `c.issues`

### 🗜️ Tweaks

- Make `deferred` status indicator more subtle; change default color from orange to a muted pink so parked items don't grab attention ([#107](https://github.com/toba/jig/issues/107))
- Retire `--body`/`--body-file` on `jig todo update` (they silently replaced the entire body); add explicit `--replace-body`/`--replace-body-file` and `--append-body`, and point agents to the safe verbs

## Week of May 3 – May 9, 2026

### ✨ Features

- Add `deferred` issue status; per-project status opt-in via additive `extra_statuses` map; `jig update` migrates existing sync-configured projects
- `cite review --json` front-loads commit message bodies and release notes; add `--with-diffs` flag for inline unified diffs (capped at 500 lines / 50 KB per file)

## Week of Apr 26 – May 2, 2026

### ✨ Features

- Add `jig cc` command; orchestrate multiple Claude Code profiles with shared `~/.claude` symlinks and isolated credentials per alias

### 🗜️ Tweaks

- `jig cc` launcher now always passes `--dangerously-skip-permissions` to `claude` ([#104](https://github.com/toba/jig/issues/104))

## Week of Apr 12 – Apr 18, 2026

### ✨ Features

- Block auto-propagation when parent has incomplete checklist; parent issues with `- [ ]` items manage their own workflow

## Week of Apr 5 – Apr 11, 2026

### ✨ Features

- Add `cite update` command for modifying existing citation fields

## Week of Mar 15 – Mar 21, 2026

### ✨ Features

- Consolidate `brew init` to push formulae to shared `owner/homebrew-tap` repo; remove per-project tap creation
- Consolidate `scoop init` to push manifests to shared `owner/scoop-bucket` repo; manifests at repo root matching charmbracelet model
- Replace `companions` config with `packages: [brew, scoop]` list and top-level `zed_extension` key

### 🐞 Fixes

- Add `--body-check`/`--body-uncheck` flags; toggle checkboxes by substring match instead of fragile exact-text replacement
- Fix TUI list truncating titles after Bubble Tea v2 migration; `Width(N)` semantics changed to include borders ([#100](https://github.com/toba/jig/issues/100))
- Fix `todo init` dropping existing `.jig.yaml` fields like `sync.github.repo`; preserve config on re-init ([#98](https://github.com/toba/jig/issues/98))
- Add changelog config documentation to `jig prime` agent prompt; agents now know about `changelog:` key in `.jig.yaml` ([#99](https://github.com/toba/jig/issues/99))

## Week of Mar 8 – Mar 14, 2026

### ✨ Features

- Auto-propagate parent issue status from children; completing all children marks parent completed
- Collapse/expand children in TUI list view; `z` toggles single root, `Z` toggles all

### 🐞 Fixes

- Fix `blockedBy` GraphQL resolver ignoring `blocked_by` frontmatter links; combine both blocking sources with dedup
- Fix file watcher silently dropping events and discarding errors; log warnings for slow subscribers and fsnotify errors
- Fix `loadFromDisk` loading `.md` files from dot-prefixed subdirectories in `.issues/`; skip with `filepath.SkipDir`
- Add `-f`/`--file` flag to `jig todo query`; avoids zsh shell escaping when GraphQL mutations contain backticks
- Compact ID-to-leaf-count spacing in TUI collapsed view; recalculate column width from visible items

### 🗜️ Tweaks

- Remove `--markdown` changelog output; agents should use `--json` to avoid merge churn

## Week of Mar 1 – Mar 7, 2026

### ✨ Features

- `changelog --markdown`; produce ready-to-paste formatted output with categorization, GitHub links, and dedup
- Add `scope` field to citation sources for specifying which local area a citation pertains to
- Add citation release tracking; `track: releases` monitors GitHub releases instead of branch commits
- Auto-promote parent to epic when adding child to non-container type ([#82](https://github.com/toba/jig/issues/82))

### 🐞 Fixes

- Fix `commit apply` leaving dirty working tree when pre-commit hooks reformat files; auto-amend hook changes into commit
- Fix `commit apply --push` printing usage text on push failure; add retry with exponential backoff for transient network errors
- Fix `cite review` saving oldest commit SHA instead of newest; use correct index based on API response order ([#72](https://github.com/toba/jig/issues/72))
- Fix brew/scoop/zed init failing when companion repo already exists; skip creation and proceed to push content
- Fix scoop init looking for wrong architecture asset (`arm64` instead of `amd64`); make ARM64 optional
- Fix brew/scoop doctor requiring goreleaser for projects with manual builds; downgrade to warning
- Fix brew/scoop doctor not accepting goreleaser v2 `format` (singular) field
- Fix `cite add` adding duplicate entry when URL already cited; skip with message
- Fix changelog excluding `review`-status issues; include alongside `completed` in changelog output
- Fix `changelog --commits 1` returning empty when no issues are completed; widen single-commit time range and auto-include git commits

### 🗜️ Tweaks

- Improve sync configuration discovery; show example YAML config in error messages, detect `.jig.yml` typo, add sync doctor check ([#71](https://github.com/toba/jig/issues/71))

## Week of Feb 23 – Mar 1, 2026

### ✨ Features

- Add `changelog` command for gathering recent issues and commits by time range ([#69](https://github.com/toba/jig/issues/69))
- Add Scoop bucket companion support; init, doctor, and CI workflow generation for Windows distribution
- Add `review` status for code-complete issues awaiting evaluation ([#65](https://github.com/toba/jig/issues/65))
- Color due date hourglass by urgency; red ≤24h, orange ≤3d, yellow ≤7d, green beyond
- Add status and priority sort options with newest-created tiebreaker

### 🐞 Fixes

- Fix TUI filter breaking tree hierarchy; preserve ancestor chain when filtering ([#64](https://github.com/toba/jig/issues/64))
- Fix TUI layout; use TypeAbbrev for dimmed rows to prevent lipgloss word-wrap ([#68](https://github.com/toba/jig/issues/68))
- Fix `jig commit` leaving dirty files after sync metadata updates ([#70](https://github.com/toba/jig/issues/70))
- Fix release workflow; add GoReleaser replace mode, parallelize scoop job

### 🗜️ Tweaks

- Fix all golangci-lint issues; add config, fix syntax errors, suppress test false positives ([#67](https://github.com/toba/jig/issues/67))
- Modernize Go idioms; extract helpers, add constants, bound sync concurrency ([#66](https://github.com/toba/jig/issues/66))
- Shorten TUI type column to two-letter abbreviations

## Week of Feb 16 – Feb 22, 2026

### ✨ Features

- Add data exfiltration detection; sensitive file uploads over network ([#14](https://github.com/toba/jig/issues/14))
- Add environment self-defense built-in check ([#27](https://github.com/toba/jig/issues/27))
- Add inline secret detection built-in ([#19](https://github.com/toba/jig/issues/19))
- Detect `$var` in command position as evasion ([#16](https://github.com/toba/jig/issues/16))
- Fail closed on malformed stdin ([#11](https://github.com/toba/jig/issues/11))
- Strip wrapper commands (`sudo`, `timeout`, `env`, etc.) in CheckNetwork ([#9](https://github.com/toba/jig/issues/9))
- Split compound commands into segments for independent rule checking ([#26](https://github.com/toba/jig/issues/26))
- Rename `cite check` → `cite review`; add `cite add` command ([#10](https://github.com/toba/jig/issues/10))
- Add `cite doctor` subcommand to verify license attribution ([#29](https://github.com/toba/jig/issues/29))
- Brew doctor; detect project language and adjust diagnostics ([#3](https://github.com/toba/jig/issues/3))
- Enhance GitHub sync to fully preserve issue relationships; parent/child via sub-issues API, footer links for blocks/blocked-by ([#20](https://github.com/toba/jig/issues/20))
- Sync milestones and blocking natively; replace footer links with GitHub milestones API and dependencies API ([#12](https://github.com/toba/jig/issues/12))
- Add tag registry; import GitHub labels as project tags with relaxed validation
- Add file-based issue tracking (todo) with Go idiom modernization
- Add top-level `jig init` command to run all sub-inits in sequence
- Upload local images during sync
- TUI auto-refresh when issues change on disk
- Add sync footer note to externally created issues

### 🐞 Fixes

- Fix brew doctor false positive on workflow asset reference check ([#30](https://github.com/toba/jig/issues/30))
- Fix commit push; push tags in version order, allow push-only without staged changes ([#17](https://github.com/toba/jig/issues/17), [#28](https://github.com/toba/jig/issues/28))
- Fix sub-issue sync; pass GitHub issue ID instead of number to sub-issues API
- Fix GitHub sync 422 on milestone clear; serialize as null instead of 0
- Fix TUI detail view selection resets on file watcher refresh
- Fix sync to update parent/subtask relationships on existing ClickUp tasks
- Fix ClickUp sync not setting parent on tasks whose parent isn't in the sync batch

### 🗜️ Tweaks

- Rename project; skill/ja → jig ([#5](https://github.com/toba/jig/issues/5))
- Rename config file; .toba.yaml → .jig.yaml ([#18](https://github.com/toba/jig/issues/18))
- Rename upstream → cite with flattened config ([#24](https://github.com/toba/jig/issues/24))
- Call todo sync in-process instead of shelling out to subprocess ([#15](https://github.com/toba/jig/issues/15))
- Go optimization sweep; extract shared utilities, add constants, parallelize doctor, expand test coverage across ~30 sub-tasks ([#7](https://github.com/toba/jig/issues/7))
- Skip brew/zed doctor gracefully when companions not configured
- Add all jig tools to `prime` command output
- Simplify GraphQL schema for agentic use
- Apply all goptimize findings

## Week of Feb 9 – Feb 15, 2026

### ✨ Features

- Implement `migrate` subcommand for importing from beans format
- Integrate ClickUp sync into todo
- Integrate GitHub sync into todo
- Import ClickUp config during migration
- Remove configurable prefix; adopt fixed xxx-xxx ID format with hash subfolders
- Add due field and due-date sorting
- Support OS default app for markdown editing in TUI
- Provide a public Go client package for external tools

### 🐞 Fixes

- Fix GitHub sync; use native types, remove label abuse
- Fix concurrent map write crash in GitHub sync

### 🗜️ Tweaks

- Optimize codebase; update to Go 1.26 and apply goptimize analysis
- Update prime command prompt template with all fork features
- Show status icon and label in status picker
- Display blockedBy relationships in show output
- Cherry-pick upstream atomic relationship updates

## Week of Feb 2 – Feb 8, 2026

### ✨ Features

- Add deep search; `//` to filter by title and body
- Add sort picker to TUI
- Replace fuzzy filter with substring filter in TUI
- Add external integration metadata to issues
- Support editor config field from config file

### 🐞 Fixes

- Fix deep search pointer invalidation bug

### 🗜️ Tweaks

- Configure GoReleaser for toba/todo fork
- Add build number to help modal and version output

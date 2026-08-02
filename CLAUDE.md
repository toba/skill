# jig

Multi-tool CLI combining file-based issue tracking, citation monitoring, Claude Code security guard, two-phase commit workflow, and Homebrew/Zed companion repo scaffolding.

## Rules

- ALWAYS write a failing test before fixing bugs
- ALWAYS create or find an issue (`jig todo`) before starting work
- Run `scripts/lint.sh` after editing Go files

## Build & Test

```bash
go build -o jig .
go test ./...
go vet ./...
scripts/lint.sh        # golangci-lint with auto-fix, then report remaining issues
```

## Architecture

- `cmd/` — Cobra commands
  - `todo` parent with `init`, `create`, `list`, `show`, `update`, `comment`, `delete`, `archive`, `roadmap`, `graphql` (alias `query`), `doctor`, `sync` (with `check`, `link`, `unlink` subcommands), `milestone` (alias `ms`; with `create`, `list`, `show`, `update`, `delete`, `migrate` subcommands), `refry`, `tui` subcommands — issue tracking
  - `commit` parent with `gather`, `apply` subcommands — two-phase commit workflow
  - `cite` parent with `init`, `review` (alias `check`), `mark`, `add`, `update` subcommands — citation monitoring
  - `nope` parent with `init`, `doctor`, `help` subcommands — security guard
  - `brew` parent with `init`, `doctor` subcommands — Homebrew tap management
  - `scoop` parent with `init`, `doctor` subcommands — Scoop bucket management
  - `zed` parent with `init`, `doctor` subcommands — Zed extension management
  - `cc` parent (`init`, `add`, `list`, `login`, `<alias>` launch) — run multiple Claude Code profiles from one machine, sharing agents/skills/commands while keeping each profile's credentials and identity separate
  - `prime` — output instructions for AI coding agents
  - `doctor` — run all doctor checks (nope, brew, scoop, zed)
  - `help-all` — show all commands and flags in agent-friendly format
  - `tui` — top-level alias for `todo tui`; bare `jig` (no subcommand) also opens the TUI
  - `sync` — top-level alias for `todo sync` (with `check`, `link`, `unlink` subcommands)
  - `update`, `version` — top-level utilities
- `internal/config/` — `.jig.yaml` partial read/write via yaml.v3 Node API (citations section)
- `internal/cite/` — repo inspection and path suggestion for `cite add`
- `internal/github/` — GitHub API client wrapping `gh` CLI
- `internal/classify/` — Glob-based file classification (high/medium/low)
- `internal/display/` — Lipgloss-styled terminal output
- `internal/commit/` — commit workflow logic (gather/apply phases)
- `internal/companion/` — companion repo management (GitHub ops, CI workflow injection)
- `internal/nope/` — PreToolUse guard (reads `nope:` section from `.jig.yaml`)
- `internal/brew/` — Homebrew tap init and doctor logic
- `internal/scoop/` — Scoop bucket init and doctor logic
- `internal/zed/` — Zed extension init and doctor logic
- `internal/cc/` — multi-profile Claude Code management: detect `~/.claude*` dirs, symlink shared config, isolate per-alias private files
- `internal/update/` — migration logic for legacy config files
- `internal/todo/config/` — todo config (reads `todo:` section from `.jig.yaml`, Node API for partial writes)
- `internal/todo/core/` — issue CRUD, archive, link checking, file watcher
- `internal/todo/graph/` — GraphQL schema and resolvers (gqlgen)
- `internal/todo/integration/` — sync integrations (ClickUp, GitHub Issues)
- `internal/todo/issue/` — issue model, frontmatter parsing, sorting
- `internal/todo/output/` — JSON output helpers
- `internal/todo/refry/` — migration from hmans/beans format
- `internal/todo/search/` — Bleve full-text search index
- `internal/todo/tui/` — Bubble Tea interactive TUI
- `internal/todo/ui/` — Lipgloss styles, tree rendering
- `pkg/client/` — GraphQL client library

## Key Design Decisions

- Config uses yaml.v3 Node API for partial read/write to avoid clobbering other sections in `.jig.yaml`
- GitHub calls shell out to `gh` CLI (no API token management needed)
- `cite review` (alias `check`) is read-only and repeatable — it never advances `last_checked_sha`/`last_checked_date`, so an agent can re-run it (or recover from lost output) without losing changes. `cite mark` advances the marker (and `last_checked_tag` for release-tracked sources) to current HEAD as a separate explicit step
- `cite add` inspects a repo via GitHub API or git clone and suggests path classification globs
- `commit` uses a two-phase gather/apply workflow so the agent can review before committing
- `brew init` and `scoop init` push to existing shared repos (`owner/homebrew-tap`, `owner/scoop-bucket`); `zed init` uses `internal/companion/` for repo creation
- `brew init` and `scoop init` auto-add to `packages` list in `.jig.yaml`
- Config uses `packages: [brew, scoop]` for convention-based repos (no URLs needed)
- Config uses `zed_extension: owner/repo` for per-project Zed extension repos
- Uses `doublestar` for `**` glob support since Go's `path.Match` lacks it
- `nope` guard reads rules from `nope:` key in `.jig.yaml` (not a separate file)
- `nope` uses instance-based `DebugLogger` (nil-safe) instead of global state
- Guard mode runs via `RunE` on the parent cobra command; exit codes use `ExitError` sentinel
- Each command group (`cite`, `nope`, `brew`, `zed`, `todo`) has its own `PersistentPreRunE`; root's is a no-op
- `nope init` writes to `.jig.yaml` and `.claude/settings.json`; migrates old `nogo`/`skill nope`/`ja nope` hooks to `jig nope`
- `todo` config uses yaml.v3 Node API for `Save()` to avoid clobbering other `.jig.yaml` sections
- `todo` stores issues as markdown files with YAML frontmatter in `.issues/`
- `todo` milestones are first-class entities (NOT an issue type), stored as files in `.issues/milestones/` (skipped by the issue loader); an issue references one via its `milestone:` frontmatter field (a milestone ID). The TUI shows the milestone short name as a badge, `m` reassigns, `g m` filters. GitHub sync maps milestone entities ↔ GitHub milestones (number stored on the milestone file's `sync.github`). The legacy `milestone` issue *type* is retired; `jig todo milestone migrate` converts old `type: milestone` issues into entities
- `todo` supports GraphQL queries/mutations via embedded gqlgen schema
- `cc` exists to let one developer run several *legitimately-held* Claude Code accounts (e.g. personal + work) from a single machine without re-installing agents/skills/commands for each. It is a convenience/DRY tool, NOT a mechanism to circumvent Anthropic usage policies, rate limits, or anti-fraud systems. Design principle: **share tooling, isolate identity.** Shared config (`agents`, `skills`, `commands`, `CLAUDE.md`, `projects`) is symlinked from one source dir; each alias keeps its own *real* copies of the `DefaultPrivate` files (`.credentials.json`, `.claude.json`, `statsig`, `telemetry`, caches). Anything that identifies or authenticates a specific account MUST stay per-alias and MUST NOT be shared or copied between accounts — most importantly the stable `machineID` and `userID` fields inside `.claude.json`, plus `oauthAccount` and the `statsig`/`telemetry` device IDs. Seeding a new alias's `.claude.json` from another account's file (see `SeedClaudeJSON`) must strip these identity fields so each account regenerates its own; sharing them makes two distinct accounts look like one install, which is exactly the cross-account linkage this tool must avoid
- `tui` and `sync` have top-level aliases that call `initTodoCore()` in their own PreRunE; the root command's `RunE` opens the TUI for bare `jig` (guarded by `cobra.NoArgs` so unknown subcommands still error)

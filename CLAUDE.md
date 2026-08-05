# jig

Multi-tool CLI combining citation monitoring, Claude Code security guard, two-phase commit workflow, multi-profile Claude Code management, and Homebrew/Scoop/Zed companion repo scaffolding.

The module is `github.com/toba/jig` but the built binary and distributed package are named **`jigo`** (`brew install toba/tap/jigo`) — the `jig` name was already taken.

## Rules

- ALWAYS write a failing test before fixing bugs
- Run `scripts/lint.sh` after editing Go files

## Build & Test

```bash
go build -o jigo .
go test ./...
go vet ./...
scripts/lint.sh        # golangci-lint with auto-fix, then report remaining issues
```

## Architecture

- `cmd/` — Cobra commands
  - `commit` parent with `gather`, `apply` subcommands — two-phase commit workflow
  - `cite` parent with `init`, `review` (alias `check`), `mark`, `add`, `update`, `doctor` subcommands — citation monitoring
  - `nope` parent with `init`, `doctor`, `help` subcommands — security guard
  - `brew` parent with `init`, `doctor` subcommands — Homebrew tap management
  - `scoop` parent with `init`, `doctor` subcommands — Scoop bucket management
  - `zed` parent with `init`, `doctor` subcommands — Zed extension management
  - `cc` parent (`init`, `add`, `list`, `login`, `<alias>` launch) — run multiple Claude Code profiles from one machine, sharing agents/skills/commands while keeping each profile's credentials and identity separate
  - `init` — run every init subcommand (nope, cite, brew, zed)
  - `doctor` — run all doctor checks (nope, brew, scoop, zed, cite, cc)
  - `help-all` — show all commands and flags in agent-friendly format
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

## Key Design Decisions

- The binary is `jigo`, but the Go module path, GitHub repo, and config file (`.jig.yaml`) all keep the `jig` name — only the distributed package and command were renamed
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
- Each command group (`cite`, `nope`, `brew`, `zed`) has its own `PersistentPreRunE`; root's is a no-op
- `nope init` writes to `.jig.yaml` and `.claude/settings.json`; migrates old `jig nope`/`nogo`/`skill nope`/`ja nope` hooks to `jigo nope` (see `legacyHookCommands`)
- Bare `jigo` (no subcommand) prints help, guarded by `cobra.NoArgs` so unknown subcommands still error
- `cc` exists to let one developer run several *legitimately-held* Claude Code accounts (e.g. personal + work) from a single machine without re-installing agents/skills/commands for each. It is a convenience/DRY tool, NOT a mechanism to circumvent Anthropic usage policies, rate limits, or anti-fraud systems. Design principle: **share tooling, isolate identity.** Shared config (`agents`, `skills`, `commands`, `CLAUDE.md`, `projects`) is symlinked from one source dir; each alias keeps its own *real* copies of the `DefaultPrivate` files (`.credentials.json`, `.claude.json`, `statsig`, `telemetry`, caches). Anything that identifies or authenticates a specific account MUST stay per-alias and MUST NOT be shared or copied between accounts — most importantly the stable `machineID` and `userID` fields inside `.claude.json`, plus `oauthAccount` and the `statsig`/`telemetry` device IDs. Seeding a new alias's `.claude.json` from another account's file (see `SeedClaudeJSON`) must strip these identity fields so each account regenerates its own; sharing them makes two distinct accounts look like one install, which is exactly the cross-account linkage this tool must avoid

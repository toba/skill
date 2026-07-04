# 👾 Jig

<img src="./assets/dad.jpg" align="right" width="100"/>

My dad was a cabinet maker. His perpetually sawdusted workshop was dotted with contrivances that I sometimes mistook for junk (some stories there!) that actually were thoughtful, if scrappy, efforts to make some task simpler or safer.

Jig is a multi-tool CLI that bundles repo monitoring, a file-based issue tracker, a Claude Code security guard, and Homebrew/Scoop/Zed companion repo scaffolding. The issue tracker is derived from [hmans/beans](https://github.com/hmans/beans), itself inspired by [steveyegge/beads](https://github.com/steveyegge/beads).

- [Install](#install)
- [Configuration](#configuration)
- [Requirements](#requirements)

## Commands

- **`jig`**
   - **`tui`**: alias for `todo tui`
   - **`sync`**: alias for `todo sync`
   - **`doctor`**: run all doctor checks (brew, scoop, zed, nope)
   - **`prime`**: output agent instructions for issue tracking
   - **`version`**: print version info
   - **[`todo`](#todo)**: file-based issue tracker for AI-first workflows
      - **`tui`**: interactive terminal UI that displays `todo` issues
      - **`init`**: create `.issues/` directory and config
      - **`create`**: create a new issue
      - **`list`**: list issues with filters
      - **`show`**: display issue details
      - **`update`**: modify an issue
      - **`delete`**: remove an issue
      - **`archive`**: archive completed/scrapped issues
      - **`roadmap`**: render issue tree
      - **`query`**: run GraphQL queries and mutations
      - **`doctor`**: validate issue links and references
      - **`sync`**: sync issues to external trackers
      - **`refry`**: migrate from [beans](https://github.com/hmans/beans) format
   - **[`cite`](#cite)**: monitor cited repositories for changes
      - **`init`**: add starter citations section to `.jig.yaml`
      - **`add`**: add the given URL to citations
      - **`review`**: fetch and display changes (read-only; repeatable)
      - **`mark`**: record sources as reviewed by advancing `last_checked_sha`
   - **[`nope`](#nope)**: Claude Code `PreToolUse` guard (reads JSON from stdin, exits 0 or 2)
      - **`init`**: scaffold nope rules in `.jig.yaml` and hook in `.claude/settings.json`
      - **`doctor`**: validate nope configuration
      - **`help`**: show nope guard reference
   - **[`changelog`](#changelog)**: gather recent issues and commits for changelog generation
   - **[`commit`](#commit)**: stage changes, check for gitignore candidates, signal push intent
   - **[`brew`](#brew)**: Homebrew tap management
      - **`init`**: create tap repo, push initial formula, inject `update-homebrew` CI job
      - **`doctor`**: verify brew tap setup is healthy
   - **[`scoop`](#scoop)**: Scoop bucket management
      - **`init`**: create bucket repo, push initial manifest, inject `update-scoop` CI job
      - **`doctor`**: verify scoop bucket setup is healthy
   - **[`zed`](#zed)**: Zed extension management
      - **`init`**: create extension repo, push scaffold, inject `sync-extension` CI job
      - **`doctor`**: verify Zed extension setup is healthy
  

## Install

```bash
brew install toba/tap/jig
```

On Windows:

```powershell
scoop bucket add toba https://github.com/toba/scoop-jig
scoop install jig
```

Or build from source:

```bash
go install github.com/toba/jig@latest
```

## Todo

A git-diffable issue tracker that lives in your project. Issues are markdown files with YAML frontmatter stored in `.issues/`. Unlike similar tools, jig can sync bidirectionally with external trackers, and it's designed to be driven by LLM agents.

```bash
jig todo init                                  # create .issues/ and config
jig todo create "Fix login bug" -t bug -s ready
jig todo list                                  # list all issues
jig todo show abc-def                          # view an issue
jig todo tui                                   # interactive terminal UI
jig todo sync                                  # sync to ClickUp or GitHub Issues
```

### What's in it

[Beans](https://github.com/hmans/beans) things and ...

- **External sync**: bidirectional sync with ClickUp and GitHub Issues (`jig todo sync`)
- **Due dates**: date field with sort support
- **TUI improvements**
    - Status icons instead of text labels
    - Sort picker (`o` key)
    - Substring search instead of fuzzy match
    - Tap `/` twice to search descriptions too
    - Due date indicators

![tui](assets/tui.png)

### Issue Types

| Type | Purpose |
|------|---------|
| `milestone` | A target release or checkpoint |
| `epic` | A thematic container for related work |
| `feature` | A user-facing capability or enhancement |
| `bug` | Something that is broken and needs fixing |
| `task` | A concrete piece of work (chore, sub-task) |

### Agent Configuration

The most basic way to teach your agent about jig's issue tracker is to add the following to your `AGENTS.md`, `CLAUDE.md`, or equivalent:

```
**IMPORTANT**: before you do anything else, run the `jig prime` command and heed its output.
```

The `prime` output is designed to be token-efficient — about 680 words — so it doesn't eat your context window every time a session starts or compacts.

#### Claude Code Hooks

Add the following hooks to your project's `.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "jig prime" }] }
    ],
    "PreCompact": [
      { "hooks": [{ "type": "command", "command": "jig prime" }] }
    ]
  }
}
```

### Agent Workflows

The real power of jig's issue tracker comes from letting your coding agent manage tasks. With the hooks above, you can use natural language:

```
Are there any tasks we should be tracking for this project? If so, please create issues for them.
```

```
What should we work on next?
```

```
Please inspect this project's issues and reorganize them into epics and milestones.
```

### Syncing with External Trackers

jig syncs issues bidirectionally with **ClickUp** and **GitHub Issues**. Configure the integration in `.jig.yaml` under `todo.sync`, then run:

```bash
jig todo sync                  # Sync all issues
jig todo sync abc-def xyz-123  # Sync specific issues
jig todo sync --dry-run        # Preview changes without applying
jig todo sync --force          # Force update even if unchanged
```

Per-issue sync state is stored in frontmatter:

```yaml
---
title: Fix login bug
status: ready
sync:
  clickup:
    task_id: "868h4hd05"
    synced_at: "2026-01-18T00:07:02Z"
  github:
    issue_number: "42"
    synced_at: "2026-01-18T00:07:02Z"
---
```

#### ClickUp

Requires `CLICKUP_TOKEN` environment variable. Syncs statuses, priorities, types, and blocking relationships as ClickUp task dependencies.

```yaml
todo:
  sync:
    clickup:
      list_id: "123456789"
      assignee: 42
      status_mapping:
        draft: "backlog"
        ready: "to do"
        in-progress: "in progress"
        completed: "complete"
        scrapped: "closed"
      priority_mapping:
        critical: 1
        high: 2
        normal: 3
        low: 4
```

#### GitHub Issues

Requires `GITHUB_TOKEN` environment variable (or `gh` CLI auth). Maps statuses, priorities, and types to GitHub labels (e.g., `status:in-progress`, `priority:high`, `type:bug`). Blocking relationships are rendered as text in the issue body.

```yaml
todo:
  sync:
    github:
      repo: "owner/repo"
```

## Cite

This arose as a new pattern (to me) while working with agents. The agent makes it easy to fork a repo and make a bunch of updates. Great. But it was quickly obvious that these changes didn't constitute a proper contribution back to the source. There were too many changes, too specific to my use-case. I also began combining sources, further impeding formal contribution.

The `cite` subcommand addresses a couple things. It will help check your license for proper attribution even when there's not a formal dependency or fork in place. And it can be run to notify you of changes within those cited sources that might be important to factor into your own project.

```bash
jig cite init
jig cite add
jig cite review   # read-only: shows changes, repeatable
jig cite mark     # record as reviewed (advances last_checked_sha)
jig cite doctor
```

Configure cited sources in `.jig.yaml`:

```yaml
citations:
  - repo: owner/repo
    branch: main
    paths:
      high:
        - "src/**/*.go"
      medium:
        - "go.mod"
      low:
        - "README.md"
```

Files are classified as high, medium, or low relevance based on glob patterns. `**` works — we use [doublestar](https://github.com/bmatcuk/doublestar) because Go's `path.Match` stubbornly refuses to support it.

## Nope

When the agent wears you down with incessant prompts you've tried fruitlessly to *always-allow*, and you decide to go YOLO (`dangerously-skip-permissions`), a little *nope* remains prudent to prevent personal apocalypse.

This command applies Claude's PreToolUse guard so that even when allowed to run wild, you can say "nope" if it tries to erase your thesis, send bomb threats or wire funds to your many enemies.

```bash
jig nope init
```

This adds a `nope:` section to `.jig.yaml` with starter rules and wires up the hook in `.claude/settings.json`. Claude Code pipes a JSON payload to `jig nope` on stdin before each tool call. If a rule matches, the tool is blocked (exit 2). If nothing matches, it's allowed (exit 0).

### Structure

The `nope:` section contains a `rules` list and an optional `debug` log path:

```yaml
nope:
  debug: .claude/nope.log   # optional JSONL debug log (omit to disable)
  rules:
    # Regex pattern — matched against the tool_input JSON
    - name: git-push
      pattern: 'git\s+push'
      message: "git push not allowed — only user should push"

    # Built-in check — structural analysis, not just string matching
    - name: pipe-commands
      builtin: pipe
      message: "piped commands not allowed — run commands separately"

    # Scope rules to specific tools (default is Bash only)
    - name: no-write-env
      pattern: '"file_path"\s*:\s*"[^"]*\.env"'
      tools: ["Write", "Edit"]
      message: "writing to .env files not allowed"
```

Rules are either regex patterns or built-in checks. Each rule has a `name`, a `message`, and either a `pattern` (regex) or `builtin` (structural check). The optional `tools` array scopes which tools the rule applies to (defaults to `["Bash"]`; use `["*"]` for all tools).

### Built-in Checks

| Name | What it catches |
|------|----------------|
| `pipe` | Pipe operators outside quotes |
| `chained` | `&&`, `\|\|`, `;` outside quotes |
| `redirect` | `>`, `>>` outside quotes |
| `subshell` | `$()`, backticks outside single quotes |
| `credential-read` | Reading `.env`, `.pem`, `.key`, SSH keys, etc. |
| `network` | `curl`, `wget`, `ssh`, etc. in command position |

Built-ins use proper shell tokenization — they understand quoting, so `grep "foo|bar"` won't trigger the pipe check. Regex patterns get `(?s)` prepended automatically so `.` matches newlines.

## Commit

I am all for having the agent write nice commit messages but my eyes bleed a little very time I see tokens ticking away while it runs the same wrong commands three times before getting it right.

This command uses other configuration in `.jig.yaml` to determine whether there are issues to synchronize or a companion brew, scoop, or Zed extension repo to consider so the agent isn't always evaluating such things.

Instead, the agent is given the list of changes to summarize, along with the last tag, if any, and asked to respond with a description and likely next tag (version).

```bash
jig commit
```

## Changelog

Agents are great at writing changelogs but terrible at gathering the raw material — they'll spend forty turns poking around git history and issue files before producing anything useful. This command collects recent issues (created, updated, completed) and optionally git commits into a single structured dump the agent can actually work with.

```bash
jig changelog --json                    # last 7 days of issues
jig changelog --json --days 30          # last 30 days
jig changelog --json --days 14 --git    # with git commits
jig changelog --json --since 2026-01-01 # explicit start date
jig changelog --commits 50 --json       # time range from last 50 commits
```

Issues are bucketed into `created`, `updated`, and `completed` — an issue lands in exactly one bucket based on priority: completed > created > updated. The `--git` flag adds a `commits` array with hash, subject, and date. Without `--json` you get a plain text summary that's still agent-friendly but won't offend human eyes.

## Brew

I just got tired of re-figuring-out how to set up the companion repository for homebrew releases. At first I used an agent skill, which helped but I ended up with three different approaches for three repositories.

```bash
jig brew init --tap toba/homebrew-todo
```

It auto-detects the source repo, latest release tag, description, and license via `gh`. The formula SHA256 is resolved using the same three-strategy approach (`.sha256` sidecar, `checksums.txt`, direct download). After running, tap updates happen automatically via CI.

```bash
jig brew init --tap toba/homebrew-todo --tag v1.2.3 --repo toba/todo --desc "My tool" --license MIT
```

Use `--dry-run` to preview without creating anything. Use `--json` for machine-readable output.

**After running**, add a `HOMEBREW_TAP_TOKEN` secret to the source repo — a GitHub PAT with Contents write access to the tap repo.

## Scoop

Same idea as brew, but for Windows. Creates a companion Scoop bucket repo with a JSON manifest covering both amd64 and arm64, and injects an `update-scoop` CI job into the release workflow.

```bash
jig scoop init --bucket toba/scoop-jig
```

It auto-detects the source repo, latest release tag, description, and license via `gh`. SHA256 hashes are resolved for both `_windows_amd64.zip` and `_windows_arm64.zip` archives. The manifest includes `checkver` and `autoupdate` sections so Scoop's tooling can pick up new versions automatically.

```bash
jig scoop init --bucket toba/scoop-jig --tag v1.2.3 --repo toba/jig --desc "My tool" --license MIT
```

Use `--dry-run` to preview without creating anything. Use `--json` for machine-readable output.

**After running**, add a `HOMEBREW_TAP_TOKEN` secret to the source repo — a GitHub PAT with Contents write access to the bucket repo (reuses the same token as Homebrew).

## Zed

One-time setup for Zed extension automation. Creates a companion extension repo on GitHub with the full scaffold (extension.toml, Cargo.toml, src/lib.rs, bump-version script and workflow, LICENSE, README), and injects a `sync-extension` job into the source repo's `release.yml`.

```bash
jig zed init --ext toba/gozer --languages "Go Text Template,Go HTML Template"
```

It auto-detects the source repo, latest release tag, and description via `gh`. The `--languages` flag is required — it sets which languages the extension provides LSP support for. After running, extension updates happen automatically via CI.

```bash
jig zed init --ext toba/gozer --languages "CSS" --tag v1.0.0 --repo toba/go-css-lsp --desc "CSS LSP" --lsp-name go-css-lsp
```

Use `--dry-run` to preview all generated files without creating anything. Use `--json` for machine-readable output.

**After running**, add an `EXTENSION_PAT` secret to the source repo — a GitHub PAT with Contents write access to the extension repo. Also run `cargo generate-lockfile` in the extension repo to create the initial `Cargo.lock`.

## Configuration

Everything lives in `.jig.yaml`. Sections are independent — you can use any subset.

```yaml
todo:
  issues:
    path: .issues
    editor: "code --wait"
  sync:
    github:
      repo: "owner/repo"

upstream:
  sources: [...]

nope:
  debug: nope.log    # optional JSONL debug log
  rules: [...]
```

Config reading uses the yaml.v3 Node API for partial read/write, so no section clobbers another.

A [JSON Schema](https://raw.githubusercontent.com/toba/jig/main/schema.json) is available for editor autocomplete and validation. Add this modeline to the top of your `.jig.yaml`:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/toba/jig/main/schema.json
```

## Requirements

- macOS, Linux, or Windows
- `gh` CLI for upstream monitoring, brew, scoop, zed, and sync commands (nope guard and todo core have no external dependencies)

## License

Apache-2.0

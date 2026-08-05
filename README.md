# 👾 Jig

<img src="./assets/dad.jpg" align="right" width="100"/>

My dad was a cabinet maker. His perpetually sawdusted workshop was dotted with contrivances that I sometimes mistook for junk (some stories there!) that actually were thoughtful, if scrappy, efforts to make some task simpler or safer.

Jig is a multi-tool CLI that bundles repo monitoring, a Claude Code security guard, multi-profile Claude Code management, and Homebrew/Scoop/Zed companion repo scaffolding.

The binary is named `jigo` (the `jig` name was already taken in Homebrew).

- [Install](#install)
- [Configuration](#configuration)
- [Requirements](#requirements)

## Commands

- **`jigo`**
   - **`init`**: run every init subcommand (nope, cite, brew, zed)
   - **`doctor`**: run all doctor checks (nope, brew, scoop, zed, cite, cc)
   - **`update`**: migrate legacy config files into `.jig.yaml`
   - **`help-all`**: show all commands and flags in agent-friendly format
   - **`version`**: print version info
   - **[`cite`](#cite)**: monitor cited repositories for changes
      - **`init`**: add starter citations section to `.jig.yaml`
      - **`add`**: add the given URL to citations
      - **`review`**: fetch and display changes (read-only; repeatable)
      - **`mark`**: record sources as reviewed by advancing `last_checked_sha`
      - **`update`**: modify an existing citation
      - **`doctor`**: validate citation configuration
   - **[`nope`](#nope)**: Claude Code `PreToolUse` guard (reads JSON from stdin, exits 0 or 2)
      - **`init`**: scaffold nope rules in `.jig.yaml` and hook in `.claude/settings.json`
      - **`doctor`**: validate nope configuration
      - **`help`**: show nope guard reference
   - **[`commit`](#commit)**: stage changes, check for gitignore candidates, signal push intent
      - **`gather`**: stage changes and output context for commit message authoring
      - **`apply`**: commit staged changes with optional tag and push
   - **`cc`**: run multiple Claude Code profiles from one machine
      - **`init`**: auto-detect `~/.claude*` dirs
      - **`add`**: create a new alias
      - **`list`**: list aliases
      - **`login`**: re-authenticate an alias
      - **`doctor`**: verify profile isolation is intact
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
brew install toba/tap/jigo
```

On Windows:

```powershell
scoop bucket add toba https://github.com/toba/scoop-bucket
scoop install jigo
```

Or build from source:

```bash
go install github.com/toba/jig@latest
```

## Cite

This arose as a new pattern (to me) while working with agents. The agent makes it easy to fork a repo and make a bunch of updates. Great. But it was quickly obvious that these changes didn't constitute a proper contribution back to the source. There were too many changes, too specific to my use-case. I also began combining sources, further impeding formal contribution.

The `cite` subcommand addresses a couple things. It will help check your license for proper attribution even when there's not a formal dependency or fork in place. And it can be run to notify you of changes within those cited sources that might be important to factor into your own project.

```bash
jigo cite init
jigo cite add
jigo cite review   # read-only: shows changes, repeatable
jigo cite mark     # record as reviewed (advances last_checked_sha)
jigo cite doctor
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
jigo nope init
```

This adds a `nope:` section to `.jig.yaml` with starter rules and wires up the hook in `.claude/settings.json`. Claude Code pipes a JSON payload to `jigo nope` on stdin before each tool call. If a rule matches, the tool is blocked (exit 2). If nothing matches, it's allowed (exit 0).

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

This command uses other configuration in `.jig.yaml` to determine whether there is a companion brew, scoop, or Zed extension repo to consider so the agent isn't always evaluating such things.

Instead, the agent is given the list of changes to summarize, along with the last tag, if any, and asked to respond with a description and likely next tag (version).

```bash
jigo commit
```

## Brew

I just got tired of re-figuring-out how to set up the companion repository for homebrew releases. At first I used an agent skill, which helped but I ended up with three different approaches for three repositories.

```bash
jigo brew init --tap toba/homebrew-todo
```

It auto-detects the source repo, latest release tag, description, and license via `gh`. The formula SHA256 is resolved using the same three-strategy approach (`.sha256` sidecar, `checksums.txt`, direct download). After running, tap updates happen automatically via CI.

```bash
jigo brew init --tap toba/homebrew-todo --tag v1.2.3 --repo toba/todo --desc "My tool" --license MIT
```

Use `--dry-run` to preview without creating anything. Use `--json` for machine-readable output.

**After running**, add a `HOMEBREW_TAP_TOKEN` secret to the source repo — a GitHub PAT with Contents write access to the tap repo.

## Scoop

Same idea as brew, but for Windows. Creates a companion Scoop bucket repo with a JSON manifest covering both amd64 and arm64, and injects an `update-scoop` CI job into the release workflow.

```bash
jigo scoop init --bucket toba/scoop-jig
```

It auto-detects the source repo, latest release tag, description, and license via `gh`. SHA256 hashes are resolved for both `_windows_amd64.zip` and `_windows_arm64.zip` archives. The manifest includes `checkver` and `autoupdate` sections so Scoop's tooling can pick up new versions automatically.

```bash
jigo scoop init --bucket toba/scoop-jig --tag v1.2.3 --repo toba/jig --desc "My tool" --license MIT
```

Use `--dry-run` to preview without creating anything. Use `--json` for machine-readable output.

**After running**, add a `HOMEBREW_TAP_TOKEN` secret to the source repo — a GitHub PAT with Contents write access to the bucket repo (reuses the same token as Homebrew).

## Zed

One-time setup for Zed extension automation. Creates a companion extension repo on GitHub with the full scaffold (extension.toml, Cargo.toml, src/lib.rs, bump-version script and workflow, LICENSE, README), and injects a `sync-extension` job into the source repo's `release.yml`.

```bash
jigo zed init --ext toba/gozer --languages "Go Text Template,Go HTML Template"
```

It auto-detects the source repo, latest release tag, and description via `gh`. The `--languages` flag is required — it sets which languages the extension provides LSP support for. After running, extension updates happen automatically via CI.

```bash
jigo zed init --ext toba/gozer --languages "CSS" --tag v1.0.0 --repo toba/go-css-lsp --desc "CSS LSP" --lsp-name go-css-lsp
```

Use `--dry-run` to preview all generated files without creating anything. Use `--json` for machine-readable output.

**After running**, add an `EXTENSION_PAT` secret to the source repo — a GitHub PAT with Contents write access to the extension repo. Also run `cargo generate-lockfile` in the extension repo to create the initial `Cargo.lock`.

## Configuration

Everything lives in `.jig.yaml`. Sections are independent — you can use any subset.

```yaml
citations:
  - repo: owner/repo
    branch: main
    paths: {...}

nope:
  debug: nope.log    # optional JSONL debug log
  rules: [...]

packages: [brew, scoop]
zed_extension: owner/repo
```

Config reading uses the yaml.v3 Node API for partial read/write, so no section clobbers another.

A [JSON Schema](https://raw.githubusercontent.com/toba/jig/main/schema.json) is available for editor autocomplete and validation. Add this modeline to the top of your `.jig.yaml`:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/toba/jig/main/schema.json
```

## Requirements

- macOS, Linux, or Windows
- `gh` CLI for citation monitoring and the brew, scoop, and zed commands (the nope guard has no external dependencies)

## License

Apache-2.0

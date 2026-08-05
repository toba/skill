package nope

import "fmt"

// HelpText is the full help reference for the nope guard.
const HelpText = `jigo nope — Claude Code PreToolUse guard

USAGE
  jigo nope              Run as hook guard (reads JSON from stdin)
  jigo nope init         Scaffold nope: section in .jig.yaml and hook in .claude/settings.json
  jigo nope doctor       Validate configuration
  jigo nope help         Show this help

CONFIGURATION
  jigo nope reads rules from the nope: section of .jig.yaml, found by
  walking up from the current directory. Each rule blocks tool usage when
  its pattern matches the tool_input field from the hook's stdin JSON payload.

  Config file structure (.jig.yaml):

    nope:
      rules:
        - name: <identifier>          # required — unique rule name
          pattern: <go-regex>         # regex matched against tool_input JSON
          builtin: <builtin-name>     # OR use a built-in check (mutually exclusive with pattern)
          message: <block-message>    # required — shown when the rule blocks
          tools: [<tool>, ...]        # optional — tool names this rule applies to (default: ["Bash"])

RULE FIELDS
  name       Unique identifier for the rule.
  pattern    Go regular expression tested against the tool_input JSON object.
             Use this for custom rules matching command text or tool parameters.
  builtin    Name of a built-in check (see BUILTINS). Mutually exclusive with pattern.
             Builtins are Bash-only and cannot be scoped to other tools.
  message    Human-readable reason shown when the rule blocks a tool call.
  tools      List of Claude Code tool names this rule applies to.
             Defaults to ["Bash"] if omitted. Use ["*"] to match all tools.
             Common tool names: Bash, Read, Write, Edit, Glob, Grep, WebFetch.

BUILTINS
  pipe              Block pipe operators (|) outside quotes.
  chained           Block chained operators (&&, ||, ;) outside quotes.
  redirect          Block output redirection (>, >>) outside quotes.
  subshell          Block subshell expansion ($(), backticks) outside quotes.
  credential-read   Block reading sensitive files (.env, .pem, .key, SSH keys, etc.).
  network           Block network tools (curl, wget, ssh, etc.) in command position.
  exfiltration      Block uploading sensitive files over the network (curl -d @.env, scp keys, /dev/tcp).
  env-hijack        Block dangerous environment variable assignments (LD_PRELOAD, DYLD_INSERT_LIBRARIES, NODE_OPTIONS, etc.).
  inline-secrets    Block commands containing inline secrets (AWS keys, GitHub tokens, API keys, passwords).
  var-command       Block variable references ($var, ${var}) in command position (potential evasion).

TOOL SCOPING
  By default, rules only match the Bash tool. Use the tools field to guard
  other Claude Code tools:

    - name: no-write-env
      pattern: '"file_path"\s*:\s*"[^"]*\.env"'
      tools: ["Write", "Edit"]
      message: "writing to .env files not allowed"

  The pattern matches against the tool_input JSON object, so for non-Bash
  tools you typically match JSON field names and values (e.g., "file_path": "...").

EXIT CODES
  0   Allow (no rule matched)
  1   Configuration error
  2   Blocked (rule matched, message printed to stderr)

EXAMPLES
  Block git push:
    - name: git-push
      pattern: 'git\s+push'
      message: "git push not allowed"

  Block force-delete on broad paths:
    - name: destructive-rm
      pattern: 'rm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+|--force\s+).*(/|~|\$HOME)'
      message: "destructive rm on broad paths not allowed"

  Block writing to credential files (Write and Edit tools):
    - name: no-write-credentials
      pattern: '"file_path"\s*:\s*"[^"]*(\.pem|\.key|credentials\.json)"'
      tools: ["Write", "Edit"]
      message: "writing to credential files not allowed"

  Block all tools from matching a pattern:
    - name: no-secrets
      pattern: '(?i)(password|secret|api.key)\s*[:=]'
      tools: ["*"]
      message: "secrets in tool input not allowed"
`

// RunHelp prints the help text.
func RunHelp() int {
	fmt.Print(HelpText)
	return 0
}

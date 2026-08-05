#!/usr/bin/env bash
# Build the current source and install it as the global `jigo`, shadowing the
# Homebrew release. Targets the Homebrew symlink (never its realpath) so the
# versioned Cellar binary is left intact. Restore the release at any time with:
#   brew link --overwrite jigo
set -euo pipefail

target="$(brew --prefix)/bin/jigo"
ver="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
commit="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
built="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

go build -ldflags "\
  -X github.com/toba/jig-go/v4/cmd.ver=${ver} \
  -X github.com/toba/jig-go/v4/cmd.commit=${commit} \
  -X github.com/toba/jig-go/v4/cmd.date=${built}" -o "${tmp}" .

# `install` follows symlinks, which would write through to the Cellar binary and
# corrupt the Homebrew install. Remove the symlink first, then drop in our build.
rm -f "${target}"
install -m 755 "${tmp}" "${target}"

echo "Installed ${ver} (${commit}) to ${target}"
echo "Restore the Homebrew release with: brew link --overwrite jigo"

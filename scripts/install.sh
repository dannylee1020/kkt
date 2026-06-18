#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
kkt installer

Usage:
  scripts/install.sh [installer options]
  curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash
  curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash -s -- [installer options]
  curl -fsSL <install.sh-url> | KKT_INSTALL_URL=<archive-url> bash -s -- [installer options]

Installer options are passed to bin/kkt-skills.mjs.

Common options:
  --target <name>   default | codex | claude | pi | opencode | all
  --local [path]    Install to project-local skill directories. Defaults to cwd.
  --dir <path>      Install to an explicit skill root directory.
  --force           Overwrite existing KKT skill directories.
  --dry-run         Print operations without writing files.

Default install writes to:
  ~/.agents/skills
  ~/.claude/skills
EOF
}

log() {
  printf '%s\n' "$*" >&2
}

fail() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

need_command() {
  command -v "$1" >/dev/null 2>&1 || fail "Required command not found: $1"
}

script_dir() {
  cd -- "$(dirname -- "$0")" && pwd -P
}

download_archive() {
  local url="$1"
  local dest="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest" || fail "Failed to download kkt archive from $url."
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url" || fail "Failed to download kkt archive from $url."
  else
    fail "curl or wget is required to download KKT from KKT_INSTALL_URL."
  fi
}

default_archive_url() {
  printf '%s\n' "https://github.com/dannylee1020/kkt/archive/refs/heads/main.tar.gz"
}

extract_archive() {
  local archive="$1"
  local dest="$2"

  need_command tar
  mkdir -p "$dest"
  tar -xzf "$archive" -C "$dest" --strip-components 1
}

resolve_root() {
  if [ -f "$0" ] && [ "$0" != "bash" ] && [ "$0" != "sh" ]; then
    local dir
    dir="$(script_dir)"
    for root in "$dir" "$dir/.."; do
      if [ -f "$root/bin/kkt-skills.mjs" ] && [ -d "$root/skills" ]; then
        cd -- "$root"
        pwd -P
        return
      fi
    done
  fi

  local install_url tmp_dir archive
  install_url="${KKT_INSTALL_URL:-$(default_archive_url)}"
  tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/kkt-install.XXXXXX")"
  archive="$tmp_dir/kkt.tar.gz"
  log "Downloading kkt from $install_url"
  download_archive "$install_url" "$archive"
  extract_archive "$archive" "$tmp_dir/src"
  printf '%s\n' "$tmp_dir/src"
  return

  fail "Could not locate kkt source."
}

main() {
  if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
    usage
    exit 0
  fi

  need_command node
  local root
  root="$(resolve_root)"
  node "$root/bin/kkt-skills.mjs" install "$@"
}

main "$@"

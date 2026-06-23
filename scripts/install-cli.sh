#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
kkt CLI installer

Usage:
  scripts/install-cli.sh [options]
  curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install-cli.sh | bash
  curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install-cli.sh | bash -s -- [options]
  curl -fsSL <install-cli.sh-url> | KKT_INSTALL_URL=<archive-url> bash -s -- [options]

Installs the Go kkt CLI. Add coding-agent instructions separately with
`kkt init <agent>`.

Options:
  --bin-dir <path>          Install the kkt CLI here. Defaults to ~/.local/bin.
  --dry-run                 Print operations without writing files.
  --help, -h                Show this help.

Environment:
  KKT_INSTALL_URL           Source archive URL. Defaults to main branch archive.
  KKT_BINARY_URL            Optional prebuilt kkt binary URL.
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

download_file() {
  local url="$1"
  local dest="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest" || fail "Failed to download $url."
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url" || fail "Failed to download $url."
  else
    fail "curl or wget is required to download KKT."
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
      if [ -f "$root/cmd/kkt/main.go" ] && [ -d "$root/internal/workflow" ]; then
        cd -- "$root"
        pwd -P
        return
      fi
    done
  fi

  local install_url tmp_dir archive
  install_url="${KKT_INSTALL_URL:-$(default_archive_url)}"
  tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/kkt-cli-install.XXXXXX")"
  archive="$tmp_dir/kkt.tar.gz"
  log "Downloading kkt source from $install_url"
  download_file "$install_url" "$archive"
  extract_archive "$archive" "$tmp_dir/src"
  printf '%s\n' "$tmp_dir/src"
}

expand_home() {
  case "$1" in
    "~") printf '%s\n' "$HOME" ;;
    "~/"*) printf '%s/%s\n' "$HOME" "${1#~/}" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

parse_args() {
  bin_dir="${KKT_BIN_DIR:-$HOME/.local/bin}"
  dry_run="false"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --bin-dir)
        [ "$#" -ge 2 ] || fail "--bin-dir requires a value."
        bin_dir="$(expand_home "$2")"
        shift 2
        ;;
      --dry-run)
        dry_run="true"
        shift
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        fail "Unknown argument: $1"
        ;;
    esac
  done
}

install_cli() {
  local root="$1"
  kkt_command="$(expand_home "$bin_dir")/kkt"

  if [ "$dry_run" = "true" ]; then
    log "[dry-run] cli: install kkt to $kkt_command"
    return
  fi

  mkdir -p "$(dirname "$kkt_command")"
  if [ -n "${KKT_BINARY_URL:-}" ]; then
    log "Downloading kkt CLI from $KKT_BINARY_URL"
    download_file "$KKT_BINARY_URL" "$kkt_command"
    chmod +x "$kkt_command"
  elif command -v go >/dev/null 2>&1; then
    log "Building kkt CLI to $kkt_command"
    (cd "$root" && go build -o "$kkt_command" ./cmd/kkt)
  else
    fail "Could not install kkt CLI: Go is not installed and KKT_BINARY_URL is not set."
  fi

  log "Installed kkt CLI: $kkt_command"
}

main() {
  parse_args "$@"
  local root
  root="$(resolve_root)"
  install_cli "$root"
}

main "$@"

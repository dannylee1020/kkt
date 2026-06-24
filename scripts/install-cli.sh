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

Installs the Go kkt CLI. KKT skills can use this binary for deterministic
.kkt state scaffolding, status, next-step hints, and validation.

Options:
  --bin-dir <path>          Install the kkt CLI here. Defaults to ~/.local/bin.
  --help                    Show this help.

Environment:
  KKT_INSTALL_URL           Source archive URL. Defaults to main branch archive.
  KKT_BINARY_URL            Explicit prebuilt kkt binary URL.
  KKT_VERSION               GitHub Release tag to install. Defaults to latest.
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

try_download_file() {
  local url="$1"
  local dest="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  else
    fail "curl or wget is required to download KKT."
  fi
}

download_file() {
  local url="$1"
  local dest="$2"

  try_download_file "$url" "$dest" || fail "Failed to download $url."
}

github_release_base_url() {
  printf '%s\n' "https://github.com/dannylee1020/kkt/releases"
}

detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Darwin) os="darwin" ;;
    Linux) os="linux" ;;
    *) return 1 ;;
  esac

  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) return 1 ;;
  esac

  printf '%s-%s\n' "$os" "$arch"
}

default_binary_url() {
  local platform asset base
  platform="$(detect_platform)" || return 1
  asset="kkt-$platform"
  base="$(github_release_base_url)"

  if [ -n "${KKT_VERSION:-}" ]; then
    printf '%s/download/%s/%s\n' "$base" "$KKT_VERSION" "$asset"
  else
    printf '%s/latest/download/%s\n' "$base" "$asset"
  fi
}

default_archive_url() {
  if [ -n "${KKT_VERSION:-}" ]; then
    printf '%s\n' "https://github.com/dannylee1020/kkt/archive/refs/tags/${KKT_VERSION}.tar.gz"
    return
  fi
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

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --bin-dir)
        [ "$#" -ge 2 ] || fail "--bin-dir requires a value."
        bin_dir="$(expand_home "$2")"
        shift 2
        ;;
      --help)
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
  local kkt_command binary_url root
  kkt_command="$(expand_home "$bin_dir")/kkt"
  binary_url="${KKT_BINARY_URL:-}"
  if [ -z "$binary_url" ]; then
    binary_url="$(default_binary_url || true)"
  fi

  mkdir -p "$(dirname "$kkt_command")"

  if [ -n "$binary_url" ]; then
    log "Downloading kkt CLI from $binary_url"
    if try_download_file "$binary_url" "$kkt_command"; then
      chmod +x "$kkt_command"
      log "Installed kkt CLI: $kkt_command"
      return
    fi
    rm -f "$kkt_command"
    log "Could not download prebuilt kkt CLI from $binary_url"
  fi

  if command -v go >/dev/null 2>&1; then
    root="$(resolve_root)"
    log "Building kkt CLI to $kkt_command"
    (cd "$root" && go build -ldflags="-X github.com/dannylee1020/kkt/internal/workflow.Version=${KKT_VERSION:-dev}" -o "$kkt_command" ./cmd/kkt)
  else
    fail "Could not install kkt CLI: prebuilt binary unavailable and Go is not installed."
  fi

  log "Installed kkt CLI: $kkt_command"
}

main() {
  parse_args "$@"
  install_cli
}

main "$@"

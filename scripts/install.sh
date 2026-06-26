#!/usr/bin/env bash
set -euo pipefail

skill_names=("kkt" "kkt-loop" "kkt-model")
legacy_skill_names=("kkt-intent" "kkt-discovery" "kkt-modeling" "kkt-execution" "kkt-validation")
native_targets=("codex" "claude" "pi" "opencode")

usage() {
  cat <<'EOF'
kkt installer

Usage:
  scripts/install.sh [installer options]
  scripts/install.sh upgrade [installer options]
  scripts/install.sh uninstall [installer options]
  scripts/install.sh doctor
  curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash
  curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash -s -- [installer options]
  curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash -s -- upgrade [installer options]
  curl -fsSL <install.sh-url> | KKT_INSTALL_URL=<archive-url> bash -s -- [installer options]

Installs KKT skills and the companion kkt CLI used for durable state.

Common options:
  --target <name>   auto | codex | claude | pi | opencode | all
  --local [path]    Install to project-local skill directories. Defaults to cwd.
  --dir <path>      Install to an explicit skill root directory.
  --bin-dir <path>  Install the kkt CLI here. Defaults to ~/.local/bin.
  --force           Overwrite existing KKT skill directories during install.
  --dry-run         Print operations without writing files.
  --help, -h        Show this help.

Commands:
  install           Install missing skills and CLI; keep existing skills unchanged.
  upgrade           Remove known old KKT skill directories, then install the latest skills and CLI.
  uninstall         Remove KKT skill directories and CLI.
  doctor            Check that source skills are present.

Default install auto-detects supported coding agents and writes to:
  ~/.agents/skills   (Codex, Pi, OpenCode)
  ~/.claude/skills   (Claude Code)
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
      if [ -d "$root/skills" ] && [ -f "$root/skills/kkt/SKILL.md" ]; then
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
  download_archive "$install_url" "$archive"
  log "source: downloaded $install_url"
  extract_archive "$archive" "$tmp_dir/src"
  printf '%s\n' "$tmp_dir/src"
}

expand_home() {
  case "$1" in
    \~) printf '%s\n' "$HOME" ;;
    \~/*) printf '%s/%s\n' "$HOME" "${1#~/}" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

absolute_path() {
  local value
  value="$(expand_home "$1")"
  if [ -d "$value" ]; then
    cd -- "$value"
    pwd -P
    return
  fi
  case "$value" in
    /*) printf '%s\n' "$value" ;;
    *) printf '%s/%s\n' "$(pwd -P)" "$value" ;;
  esac
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

agent_detected() {
  case "$1" in
    codex) command_exists codex || [ -d "$HOME/.codex" ] ;;
    claude) command_exists claude || [ -d "$HOME/.claude" ] ;;
    pi) command_exists pi || [ -d "$HOME/.pi" ] ;;
    opencode) command_exists opencode || [ -d "$HOME/.config/opencode" ] ;;
    *) return 1 ;;
  esac
}

contains_value() {
  local needle="$1"
  shift
  local value
  for value in "$@"; do
    [ "$value" = "$needle" ] && return 0
  done
  return 1
}

target_root() {
  local target="$1"
  local local_mode="$2"
  local local_path="$3"
  local project_root
  project_root="$(absolute_path "$local_path")"

  case "$target" in
    codex|pi|opencode)
      if [ "$local_mode" = "true" ]; then
        printf '%s/.agents/skills\n' "$project_root"
      else
        printf '%s/.agents/skills\n' "$HOME"
      fi
      ;;
    claude)
      if [ "$local_mode" = "true" ]; then
        printf '%s/.claude/skills\n' "$project_root"
      else
        printf '%s/.claude/skills\n' "$HOME"
      fi
      ;;
    *)
      fail "Unsupported target: $target"
      ;;
  esac
}

add_root() {
  local label="$1"
  local root="$2"
  local existing
  if [ "${#root_paths[@]}" -gt 0 ]; then
    for existing in "${root_paths[@]}"; do
      [ "$existing" = "$root" ] && return
    done
  fi
  root_labels+=("$label")
  root_paths+=("$root")
}

resolve_targets() {
  resolved_targets=()
  case "$target" in
    default|auto)
      local native
      for native in "${native_targets[@]}"; do
        if agent_detected "$native"; then
          resolved_targets+=("$native")
        fi
      done
      if [ "${#resolved_targets[@]}" -eq 0 ]; then
        fail "No supported coding agent was detected. Rerun with --target codex, --target claude, --target pi, --target opencode, or --target all."
      fi
      ;;
    all)
      resolved_targets=("${native_targets[@]}")
      ;;
    codex|claude|pi|opencode)
      resolved_targets=("$target")
      ;;
    *)
      fail "Unsupported target: $target"
      ;;
  esac
}

resolve_roots() {
  root_labels=()
  root_paths=()
  if [ -n "$explicit_dir" ]; then
    local label
    label="$target"
    if [ "$label" = "auto" ] || [ "$label" = "default" ]; then
      label="explicit"
    fi
    add_root "$label" "$(absolute_path "$explicit_dir")"
    return
  fi

  resolve_targets
  local native root
  for native in "${resolved_targets[@]}"; do
    root="$(target_root "$native" "$local_install" "$local_path")"
    add_root "$native" "$root"
  done
}

parse_args() {
  command_name="${1:-install}"
  case "$command_name" in
    install|upgrade|uninstall|doctor)
      shift || true
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      command_name="install"
      ;;
  esac

  target="auto"
  local_install="false"
  local_path="$(pwd -P)"
  explicit_dir=""
  bin_dir="${KKT_BIN_DIR:-$HOME/.local/bin}"
  force="false"
  dry_run="false"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --target)
        [ "$#" -ge 2 ] || fail "--target requires a value."
        target="$2"
        shift 2
        ;;
      --local)
        local_install="true"
        if [ "$#" -ge 2 ] && [[ "$2" != -* ]]; then
          local_path="$2"
          shift 2
        else
          shift
        fi
        ;;
      --dir)
        [ "$#" -ge 2 ] || fail "--dir requires a value."
        explicit_dir="$2"
        shift 2
        ;;
      --bin-dir)
        [ "$#" -ge 2 ] || fail "--bin-dir requires a value."
        bin_dir="$(expand_home "$2")"
        shift 2
        ;;
      --force)
        force="true"
        shift
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

ensure_source_skills() {
  source_skills_root="$root/skills"
  [ -d "$source_skills_root" ] || fail "Source skills directory not found: $source_skills_root"
  local name
  for name in "${skill_names[@]}"; do
    [ -f "$source_skills_root/$name/SKILL.md" ] || fail "Missing source skill: $source_skills_root/$name/SKILL.md"
  done
}

directories_equal() {
  local left="$1"
  local right="$2"
  [ -d "$left" ] && [ -d "$right" ] || return 1
  diff -qr "$left" "$right" >/dev/null 2>&1
}

record_operation() {
  operation_actions+=("$1")
  operation_names+=("$2")
  operation_targets+=("$3")
}

reset_operations() {
  operation_actions=()
  operation_names=()
  operation_targets=()
}

copy_skill() {
  local name="$1"
  local root_dir="$2"
  local source="$source_skills_root/$name"
  local target_dir="$root_dir/$name"

  if [ -e "$target_dir" ]; then
    if directories_equal "$source" "$target_dir"; then
      record_operation "skip" "$name" "$target_dir"
      return
    fi
    if [ "$force" != "true" ]; then
      record_operation "keep" "$name" "$target_dir"
      return
    fi
    record_operation "overwrite" "$name" "$target_dir"
    return
  fi

  record_operation "copy" "$name" "$target_dir"
}

install_root() {
  local root_dir="$1"
  reset_operations

  local name
  for name in "${skill_names[@]}"; do
    copy_skill "$name" "$root_dir"
  done

  if [ "$dry_run" != "true" ]; then
    local i source target_dir
    for i in "${!operation_actions[@]}"; do
      case "${operation_actions[$i]}" in
        copy)
          source="$source_skills_root/${operation_names[$i]}"
          target_dir="${operation_targets[$i]}"
          cp -R -- "$source" "$target_dir"
          ;;
        overwrite)
          source="$source_skills_root/${operation_names[$i]}"
          target_dir="${operation_targets[$i]}"
          rm -rf -- "$target_dir"
          cp -R -- "$source" "$target_dir"
          ;;
      esac
    done
  fi
}

upgrade_root() {
  local root_dir="$1"
  reset_operations

  local name target_dir
  for name in "${skill_names[@]}" "${legacy_skill_names[@]}"; do
    target_dir="$root_dir/$name"
    if [ -e "$target_dir" ]; then
      record_operation "remove" "$name" "$target_dir"
      if [ "$dry_run" != "true" ]; then
        rm -rf -- "$target_dir"
      fi
    else
      record_operation "skip" "$name" "$target_dir"
    fi
  done

  for name in "${skill_names[@]}"; do
    target_dir="$root_dir/$name"
    record_operation "copy" "$name" "$target_dir"
    if [ "$dry_run" != "true" ]; then
      cp -R -- "$source_skills_root/$name" "$target_dir"
    fi
  done
}

uninstall_root() {
  local root_dir="$1"
  reset_operations

  local name target_dir
  for name in "${skill_names[@]}" "${legacy_skill_names[@]}"; do
    target_dir="$root_dir/$name"
    if [ -e "$target_dir" ]; then
      record_operation "remove" "$name" "$target_dir"
      if [ "$dry_run" != "true" ]; then
        rm -rf -- "$target_dir"
      fi
    else
      record_operation "skip" "$name" "$target_dir"
    fi
  done
}

names_for_action() {
  local action="$1"
  local names=()
  local i
  for i in "${!operation_actions[@]}"; do
    if [ "${operation_actions[$i]}" = "$action" ]; then
      names+=("${operation_names[$i]}")
    fi
  done
  if [ "${#names[@]}" -gt 0 ]; then
    local joined=""
    local name
    for name in "${names[@]}"; do
      if [ -n "$joined" ]; then
        joined="$joined, $name"
      else
        joined="$name"
      fi
    done
    printf '%s\n' "$joined"
  fi
}

print_summary_line() {
  local label="$1"
  local summary="$2"
  [ -n "$summary" ] || return 0
  printf '  - %s: %s\n' "$label" "$summary"
}

print_operations() {
  local command="$1"
  local label="$2"
  local root_dir="$3"
  local suffix=""
  if [ "$dry_run" = "true" ]; then
    suffix=" (dry-run)"
  fi

  printf 'skills %s%s: %s\n' "$label" "$suffix" "$root_dir"

  if [ "$command" = "upgrade" ]; then
    print_summary_line "installed" "$(names_for_action copy || true)"
    local removed_legacy=()
    local i
    for i in "${!operation_actions[@]}"; do
      if [ "${operation_actions[$i]}" = "remove" ] && contains_value "${operation_names[$i]}" "${legacy_skill_names[@]}"; then
        removed_legacy+=("${operation_names[$i]}")
      fi
    done
    if [ "${#removed_legacy[@]}" -gt 0 ]; then
      local joined=""
      local name
      for name in "${removed_legacy[@]}"; do
        if [ -n "$joined" ]; then
          joined="$joined, $name"
        else
          joined="$name"
        fi
      done
      print_summary_line "removed legacy" "$joined"
    fi
    return
  fi

  if [ "$command" = "uninstall" ]; then
    local removed current
    removed="$(names_for_action remove || true)"
    current="$(names_for_action skip || true)"
    print_summary_line "removed" "$removed"
    if [ -z "$removed" ]; then
      print_summary_line "not installed" "$current"
    fi
    return
  fi

  local installed updated current kept
  installed="$(names_for_action copy || true)"
  updated="$(names_for_action overwrite || true)"
  kept="$(names_for_action keep || true)"
  current="$(names_for_action skip || true)"
  print_summary_line "installed" "$installed"
  print_summary_line "updated" "$updated"
  print_summary_line "current" "$current"
  print_summary_line "kept existing" "$kept"
  if [ -n "$kept" ]; then
    print_summary_line "next" "run scripts/install.sh upgrade to replace existing skills"
  fi
}

doctor() {
  ensure_source_skills
  printf 'ok: %s\n' "$source_skills_root"
}

install_cli() {
  local kkt_command cli_installer
  kkt_command="$(expand_home "$bin_dir")/kkt"
  cli_installer="$root/scripts/install-cli.sh"

  if [ "$dry_run" = "true" ]; then
    printf 'cli: would install %s\n' "$kkt_command"
    return
  fi

  [ -f "$cli_installer" ] || fail "CLI installer not found: $cli_installer"
  bash "$cli_installer" --bin-dir "$bin_dir"
}

uninstall_cli() {
  local kkt_command
  kkt_command="$(expand_home "$bin_dir")/kkt"

  if [ "$dry_run" = "true" ]; then
    printf 'cli: would remove %s\n' "$kkt_command"
    return
  fi

  if [ -e "$kkt_command" ]; then
    rm -f -- "$kkt_command"
    printf 'cli: removed %s\n' "$kkt_command"
  else
    printf 'cli: not installed %s\n' "$kkt_command"
  fi
}

main() {
  parse_args "$@"
  root="$(resolve_root)"
  ensure_source_skills

  if [ "$command_name" = "doctor" ]; then
    doctor
    return
  fi

  need_command diff
  resolve_roots
  local i root_dir label
  for i in "${!root_paths[@]}"; do
    root_dir="${root_paths[$i]}"
    label="${root_labels[$i]}"
    if [ "$dry_run" != "true" ]; then
      mkdir -p "$root_dir"
    fi
    case "$command_name" in
      install) install_root "$root_dir" ;;
      upgrade) upgrade_root "$root_dir" ;;
      uninstall) uninstall_root "$root_dir" ;;
      *) fail "Unsupported command: $command_name" ;;
    esac
    print_operations "$command_name" "$label" "$root_dir"
  done

  case "$command_name" in
    install|upgrade) install_cli ;;
    uninstall) uninstall_cli ;;
  esac
}

main "$@"

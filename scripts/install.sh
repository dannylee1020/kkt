#!/usr/bin/env bash
set -euo pipefail

skill_names=("kkt" "kkt-loop" "kkt-model" "kkt-run")
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

Installs KKT skills, ast-grep, and the companion kkt CLI used for optional and durable workflow state.

Common options:
  --target <name>   auto | codex | claude | pi | opencode | all
  --local [path]    Install to project-local skill directories. Defaults to cwd.
  --dir <path>      Install to an explicit skill root directory.
  --bin-dir <path>  Install the kkt CLI here. Defaults to ~/.local/bin.
  --hooks           Install inert agent hook adapters. Enforcement stays off until kkt hooks arm.
  --force           Overwrite existing KKT skill directories during install.
  --dry-run         Print operations without writing files.
  --help, -h        Show this help.

Commands:
  install           Install missing skills and CLI; keep existing skills unchanged.
  upgrade           Remove known old KKT skill directories, then install the latest skills and CLI.
  uninstall         Remove KKT skill directories and CLI.
  doctor            Check that source skills are present.

Environment:
  KKT_BUILD_FROM_SOURCE  Build the CLI from source without downloading a release binary.

Hook adapters are opt-in and no-op by default. They call `kkt hook ...`, which only enforces
when the current project has an armed `.kkt/hooks.json`.

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
  install_hooks="false"

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
      --hooks)
        install_hooks="true"
        shift
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

install_ast_grep_with() {
  local installer="$1"
  shift
  log "dependency: installing ast-grep with $installer"
  "$@"
}

ensure_ast_grep() {
  local path
  if path="$(command -v ast-grep 2>/dev/null)"; then
    log "dependency: ast-grep already installed at $path"
    return
  fi

  if [ "$dry_run" = "true" ]; then
    printf 'dependency: would install ast-grep if missing\n'
    return
  fi

  if command_exists brew; then
    install_ast_grep_with "brew" brew install ast-grep
    return
  fi

  if command_exists cargo; then
    install_ast_grep_with "cargo" cargo install ast-grep --locked
    return
  fi

  if command_exists npm; then
    install_ast_grep_with "npm" npm i @ast-grep/cli -g
    return
  fi

  fail "ast-grep is required for KKT structural discovery. Install it with: brew install ast-grep, cargo install ast-grep --locked, or npm i @ast-grep/cli -g."
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

hooks_root_for_target() {
  local target_name="$1"
  local project_root
  project_root="$(absolute_path "$local_path")"
  case "$target_name" in
    pi)
      if [ "$local_install" = "true" ]; then
        printf '%s/.pi/extensions\n' "$project_root"
      else
        printf '%s/.pi/agent/extensions\n' "$HOME"
      fi
      ;;
    opencode)
      if [ "$local_install" = "true" ]; then
        printf '%s/.opencode/plugins\n' "$project_root"
      else
        printf '%s/.config/opencode/plugins\n' "$HOME"
      fi
      ;;
    claude)
      if [ "$local_install" = "true" ]; then
        printf '%s/.claude/settings.local.json\n' "$project_root"
      else
        printf '%s/.claude/settings.json\n' "$HOME"
      fi
      ;;
    codex)
      if [ "$local_install" = "true" ]; then
        printf '%s/.codex/hooks.json\n' "$project_root"
      else
        printf '%s/.codex/hooks.json\n' "$HOME"
      fi
      ;;
    *) fail "Unsupported hook target: $target_name" ;;
  esac
}

copy_hook_adapter() {
  local source="$1"
  local dest="$2"
  local label="$3"
  if [ "$dry_run" = "true" ]; then
    printf 'hooks %s: would install %s\n' "$label" "$dest"
    return
  fi
  mkdir -p "$(dirname "$dest")"
  cp -- "$source" "$dest"
  printf 'hooks %s: installed %s\n' "$label" "$dest"
}

install_claude_hooks() {
  local settings_path="$1"
  if [ "$dry_run" = "true" ]; then
    printf 'hooks claude: would update %s\n' "$settings_path"
    return
  fi
  command_exists python3 || fail "python3 is required to merge Claude hook settings."
  mkdir -p "$(dirname "$settings_path")"
  python3 - "$settings_path" <<'PY'
import json
import os
import sys

path = sys.argv[1]
try:
    with open(path, "r", encoding="utf-8") as handle:
        data = json.load(handle)
except FileNotFoundError:
    data = {}
if not isinstance(data, dict):
    raise SystemExit(f"Claude settings must be a JSON object: {path}")

def add_hook(event, matcher, command):
    hooks = data.setdefault("hooks", {})
    groups = hooks.setdefault(event, [])
    for group in groups:
        if not isinstance(group, dict):
            continue
        if group.get("matcher") != matcher:
            continue
        for handler in group.get("hooks", []):
            if isinstance(handler, dict) and handler.get("type") == "command" and handler.get("command") == command:
                return
    groups.append({"matcher": matcher, "hooks": [{"type": "command", "command": command}]})

matcher = "Bash|Edit|MultiEdit|Write|NotebookEdit"
add_hook("PreToolUse", matcher, "kkt hook pre-tool --agent claude")
add_hook("PostToolUse", matcher, "kkt hook post-tool --agent claude")
with open(path, "w", encoding="utf-8") as handle:
    json.dump(data, handle, indent=2)
    handle.write("\n")
PY
  printf 'hooks claude: installed %s\n' "$settings_path"
}

uninstall_claude_hooks() {
  local settings_path="$1"
  if [ "$dry_run" = "true" ]; then
    printf 'hooks claude: would remove KKT hooks from %s\n' "$settings_path"
    return
  fi
  [ -f "$settings_path" ] || return
  command_exists python3 || fail "python3 is required to merge Claude hook settings."
  python3 - "$settings_path" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as handle:
    data = json.load(handle)
if not isinstance(data, dict):
    raise SystemExit(0)
commands = {"kkt hook pre-tool --agent claude", "kkt hook post-tool --agent claude"}
hooks = data.get("hooks")
if isinstance(hooks, dict):
    for event, groups in list(hooks.items()):
        if not isinstance(groups, list):
            continue
        next_groups = []
        for group in groups:
            if not isinstance(group, dict):
                next_groups.append(group)
                continue
            handlers = group.get("hooks")
            if not isinstance(handlers, list):
                next_groups.append(group)
                continue
            group["hooks"] = [h for h in handlers if not (isinstance(h, dict) and h.get("command") in commands)]
            if group["hooks"]:
                next_groups.append(group)
        if next_groups:
            hooks[event] = next_groups
        else:
            hooks.pop(event, None)
    if not hooks:
        data.pop("hooks", None)
with open(path, "w", encoding="utf-8") as handle:
    json.dump(data, handle, indent=2)
    handle.write("\n")
PY
  printf 'hooks claude: removed KKT hooks from %s\n' "$settings_path"
}

codex_config_has_inline_hooks() {
  local config_path="$1"
  [ -f "$config_path" ] && grep -Eq '^[[:space:]]*\[\[?hooks(\.|\])' "$config_path"
}

install_codex_toml_hooks() {
  local config_path="$1"
  if [ "$dry_run" = "true" ]; then
    printf 'hooks codex: would update %s\n' "$config_path"
    return
  fi
  command_exists python3 || fail "python3 is required to merge Codex hook settings."
  mkdir -p "$(dirname "$config_path")"
  python3 - "$config_path" <<'PY'
import re
import sys

path = sys.argv[1]
try:
    text = open(path, "r", encoding="utf-8").read()
except FileNotFoundError:
    text = ""
text = re.sub(r"\n?# kkt-hooks:start\n.*?# kkt-hooks:end\n?", "\n", text, flags=re.S)
block = '''# kkt-hooks:start
[[hooks.PreToolUse]]
matcher = "Bash|apply_patch|Edit|Write"

[[hooks.PreToolUse.hooks]]
type = "command"
command = "kkt hook pre-tool --agent codex"
statusMessage = "Checking KKT guardrails"

[[hooks.PostToolUse]]
matcher = "Bash|apply_patch|Edit|Write"

[[hooks.PostToolUse.hooks]]
type = "command"
command = "kkt hook post-tool --agent codex"
statusMessage = "Checking KKT guardrails"
# kkt-hooks:end
'''
text = text.rstrip() + "\n\n" + block
open(path, "w", encoding="utf-8").write(text)
PY
  printf 'hooks codex: installed %s\n' "$config_path"
}

uninstall_codex_toml_hooks() {
  local config_path="$1"
  [ -f "$config_path" ] || return
  command_exists python3 || fail "python3 is required to merge Codex hook settings."
  python3 - "$config_path" <<'PY'
import re
import sys
path = sys.argv[1]
text = open(path, "r", encoding="utf-8").read()
text = re.sub(r"\n?# kkt-hooks:start\n.*?# kkt-hooks:end\n?", "\n", text, flags=re.S)
open(path, "w", encoding="utf-8").write(text.rstrip() + "\n")
PY
  printf 'hooks codex: removed KKT hooks from %s\n' "$config_path"
}

install_codex_hooks() {
  local hooks_path="$1"
  local config_path
  config_path="$(dirname "$hooks_path")/config.toml"
  if codex_config_has_inline_hooks "$config_path"; then
    install_codex_toml_hooks "$config_path"
    return
  fi
  if [ "$dry_run" = "true" ]; then
    printf 'hooks codex: would update %s\n' "$hooks_path"
    return
  fi
  command_exists python3 || fail "python3 is required to merge Codex hook settings."
  mkdir -p "$(dirname "$hooks_path")"
  python3 - "$hooks_path" <<'PY'
import json
import sys

path = sys.argv[1]
try:
    with open(path, "r", encoding="utf-8") as handle:
        data = json.load(handle)
except FileNotFoundError:
    data = {}
if not isinstance(data, dict):
    raise SystemExit(f"Codex hooks file must be a JSON object: {path}")

def add_hook(event, matcher, command):
    hooks = data.setdefault("hooks", {})
    groups = hooks.setdefault(event, [])
    for group in groups:
        if not isinstance(group, dict) or group.get("matcher") != matcher:
            continue
        for handler in group.get("hooks", []):
            if isinstance(handler, dict) and handler.get("type") == "command" and handler.get("command") == command:
                return
    groups.append({
        "matcher": matcher,
        "hooks": [{
            "type": "command",
            "command": command,
            "statusMessage": "Checking KKT guardrails",
        }],
    })

matcher = "Bash|apply_patch|Edit|Write"
add_hook("PreToolUse", matcher, "kkt hook pre-tool --agent codex")
add_hook("PostToolUse", matcher, "kkt hook post-tool --agent codex")
with open(path, "w", encoding="utf-8") as handle:
    json.dump(data, handle, indent=2)
    handle.write("\n")
PY
  printf 'hooks codex: installed %s\n' "$hooks_path"
}

uninstall_codex_hooks() {
  local hooks_path="$1"
  local config_path
  config_path="$(dirname "$hooks_path")/config.toml"
  if [ "$dry_run" = "true" ]; then
    printf 'hooks codex: would remove KKT hooks from %s\n' "$hooks_path"
    printf 'hooks codex: would remove KKT hooks from %s\n' "$config_path"
    return
  fi
  if [ -f "$hooks_path" ]; then
    command_exists python3 || fail "python3 is required to merge Codex hook settings."
    python3 - "$hooks_path" <<'PY'
import json
import os
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as handle:
    data = json.load(handle)
if not isinstance(data, dict):
    raise SystemExit(0)
commands = {"kkt hook pre-tool --agent codex", "kkt hook post-tool --agent codex"}
hooks = data.get("hooks")
if isinstance(hooks, dict):
    for event, groups in list(hooks.items()):
        if not isinstance(groups, list):
            continue
        next_groups = []
        for group in groups:
            if not isinstance(group, dict):
                next_groups.append(group)
                continue
            handlers = group.get("hooks")
            if not isinstance(handlers, list):
                next_groups.append(group)
                continue
            group["hooks"] = [h for h in handlers if not (isinstance(h, dict) and h.get("command") in commands)]
            if group["hooks"]:
                next_groups.append(group)
        if next_groups:
            hooks[event] = next_groups
        else:
            hooks.pop(event, None)
    if not hooks:
        data.pop("hooks", None)
if data:
    with open(path, "w", encoding="utf-8") as handle:
        json.dump(data, handle, indent=2)
        handle.write("\n")
else:
    os.remove(path)
PY
    printf 'hooks codex: removed KKT hooks from %s\n' "$hooks_path"
  fi
  uninstall_codex_toml_hooks "$config_path"
}

install_hook_adapters() {
  resolve_targets
  local native dest
  for native in "${resolved_targets[@]}"; do
    dest="$(hooks_root_for_target "$native")"
    case "$native" in
      pi) copy_hook_adapter "$root/adapters/pi/kkt-hooks.ts" "$dest/kkt-hooks.ts" "pi" ;;
      opencode) copy_hook_adapter "$root/adapters/opencode/kkt-hooks.ts" "$dest/kkt-hooks.ts" "opencode" ;;
      claude) install_claude_hooks "$dest" ;;
      codex) install_codex_hooks "$dest" ;;
    esac
  done
}

uninstall_hook_adapters() {
  resolve_targets
  local native dest
  for native in "${resolved_targets[@]}"; do
    dest="$(hooks_root_for_target "$native")"
    case "$native" in
      pi)
        if [ "$dry_run" = "true" ]; then
          printf 'hooks pi: would remove %s\n' "$dest/kkt-hooks.ts"
        else
          rm -f -- "$dest/kkt-hooks.ts"
          printf 'hooks pi: removed %s\n' "$dest/kkt-hooks.ts"
        fi
        ;;
      opencode)
        if [ "$dry_run" = "true" ]; then
          printf 'hooks opencode: would remove %s\n' "$dest/kkt-hooks.ts"
        else
          rm -f -- "$dest/kkt-hooks.ts"
          printf 'hooks opencode: removed %s\n' "$dest/kkt-hooks.ts"
        fi
        ;;
      claude) uninstall_claude_hooks "$dest" ;;
      codex) uninstall_codex_hooks "$dest" ;;
    esac
  done
}

main() {
  parse_args "$@"
  root="$(resolve_root)"
  ensure_source_skills

  if [ "$command_name" = "doctor" ]; then
    doctor
    return
  fi

  case "$command_name" in
    install|upgrade) ensure_ast_grep ;;
  esac

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
    install|upgrade)
      install_cli
      if [ "$install_hooks" = "true" ]; then
        install_hook_adapters
      fi
      ;;
    uninstall)
      if [ "$install_hooks" = "true" ]; then
        uninstall_hook_adapters
      fi
      uninstall_cli
      ;;
  esac
}

main "$@"

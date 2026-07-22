#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/kkt-install-hooks.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

FAKE_BIN="$TMP_DIR/bin"
FAKE_KKT="$TMP_DIR/fake-kkt"
mkdir -p "$FAKE_BIN"

cat >"$FAKE_BIN/ast-grep" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$FAKE_BIN/ast-grep"

cat >"$FAKE_KKT" <<'EOF'
#!/usr/bin/env bash
printf 'fake kkt\n'
EOF
chmod +x "$FAKE_KKT"

export PATH="$FAKE_BIN:$PATH"
export KKT_BINARY_URL="file://$FAKE_KKT"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

pass() {
  printf 'ok: %s\n' "$*"
}

assert_file() {
  [ -f "$1" ] || fail "expected file: $1"
}

assert_not_file() {
  [ ! -e "$1" ] || fail "expected path to be absent: $1"
}

assert_contains() {
  grep -Fq -- "$2" "$1" || fail "expected $1 to contain: $2"
}

assert_not_contains() {
  ! grep -Fq -- "$2" "$1" || fail "expected $1 not to contain: $2"
}

run_installer() {
  local home="$1"
  shift
  HOME="$home" "$ROOT/scripts/install.sh" "$@"
}

assert_json_contracts() {
  local project="$1"
  PROJECT="$project" python3 - <<'PY'
import json
import os
from pathlib import Path

project = Path(os.environ["PROJECT"])


def load(relative):
    with (project / relative).open(encoding="utf-8") as handle:
        return json.load(handle)


def handlers(data, event):
    return [
        handler
        for group in data.get("hooks", {}).get(event, [])
        for handler in group.get("hooks", [])
        if isinstance(handler, dict)
    ]

codex = load(".codex/hooks.json")
pre = handlers(codex, "PreToolUse")
post = handlers(codex, "PostToolUse")
assert any(item.get("command") == "custom-before" for item in pre)
assert any(item.get("command") == "custom-after" for item in post)
assert sum(item.get("command") == "kkt hook pre-tool --agent codex" for item in pre) == 1
assert sum(item.get("command") == "kkt hook post-tool --agent codex" for item in post) == 1

claude = load(".claude/settings.local.json")
pre = handlers(claude, "PreToolUse")
post = handlers(claude, "PostToolUse")
assert any(item.get("command") == "custom-claude-before" for item in pre)
assert any(item.get("command") == "custom-claude-after" for item in post)
assert not any(item.get("command", "").startswith("kkt hook ") for item in pre + post)
for event, name in (("PreToolUse", "pre-tool"), ("PostToolUse", "post-tool")):
    matches = [
        item for item in handlers(claude, event)
        if item.get("command") == "kkt" and item.get("args") == ["hook", name, "--agent", "claude"]
    ]
    assert len(matches) == 1, (event, matches)
    assert matches[0].get("timeout") == 30
    assert matches[0].get("statusMessage") == "Checking KKT guardrails"

assert "custom" in codex
assert claude["permissions"] == {"allow": ["Read(*)"]}
PY
}

assert_global_install() {
  local home="$1"
  assert_file "$home/.agents/skills/kkt/SKILL.md"
  assert_file "$home/.claude/skills/kkt/SKILL.md"
  assert_file "$home/.codex/hooks.json"
  assert_file "$home/.claude/settings.json"
  assert_file "$home/.pi/agent/extensions/kkt-hooks.ts"
  assert_file "$home/.config/opencode/plugins/kkt-hooks.ts"
}

LOCAL_HOME="$TMP_DIR/local-home"
LOCAL_PROJECT="$TMP_DIR/local-project"
LOCAL_BIN="$TMP_DIR/local-bin"
mkdir -p "$LOCAL_HOME" "$LOCAL_PROJECT/.codex" "$LOCAL_PROJECT/.claude"

cat >"$LOCAL_PROJECT/.codex/hooks.json" <<'EOF'
{
  "custom": "keep",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "custom-before"}]
      },
      {
        "matcher": "Bash|apply_patch|Edit|Write",
        "hooks": [{"type": "command", "command": "kkt hook pre-tool --agent codex"}]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{"type": "command", "command": "custom-after"}]
      },
      {
        "matcher": "Bash|apply_patch|Edit|Write",
        "hooks": [{"type": "command", "command": "kkt hook post-tool --agent codex"}]
      }
    ]
  }
}
EOF

cat >"$LOCAL_PROJECT/.claude/settings.local.json" <<'EOF'
{
  "permissions": {"allow": ["Read(*)"]},
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          {"type": "command", "command": "custom-claude-before"},
          {"type": "command", "command": "kkt hook pre-tool --agent claude"}
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          {"type": "command", "command": "custom-claude-after"},
          {"type": "command", "command": "kkt hook post-tool --agent claude"}
        ]
      }
    ]
  }
}
EOF

cat >"$LOCAL_PROJECT/.codex/config.toml" <<'EOF'
model = "keep-me"

# kkt-hooks:start
legacy = true
# kkt-hooks:end
EOF

run_installer "$LOCAL_HOME" install --target all --local "$LOCAL_PROJECT" --hooks --bin-dir "$LOCAL_BIN"

assert_file "$LOCAL_PROJECT/.agents/skills/kkt/SKILL.md"
assert_file "$LOCAL_PROJECT/.claude/skills/kkt/SKILL.md"
assert_file "$LOCAL_PROJECT/.pi/extensions/kkt-hooks.ts"
assert_file "$LOCAL_PROJECT/.opencode/plugins/kkt-hooks.ts"
assert_file "$LOCAL_PROJECT/.codex/hooks.json"
assert_file "$LOCAL_PROJECT/.claude/settings.local.json"
assert_file "$LOCAL_BIN/kkt"
assert_not_contains "$LOCAL_PROJECT/.codex/config.toml" '# kkt-hooks:start'
cmp "$ROOT/adapters/pi/kkt-hooks.ts" "$LOCAL_PROJECT/.pi/extensions/kkt-hooks.ts"
cmp "$ROOT/adapters/opencode/kkt-hooks.ts" "$LOCAL_PROJECT/.opencode/plugins/kkt-hooks.ts"
assert_json_contracts "$LOCAL_PROJECT"
pass "local all-target installation merges and preserves hook configuration"

mkdir -p "$LOCAL_PROJECT/.agents/skills/kkt" "$LOCAL_PROJECT/.pi/extensions" "$LOCAL_PROJECT/.opencode/plugins"
printf 'stale skill\n' >"$LOCAL_PROJECT/.agents/skills/kkt/stale.txt"
printf 'stale pi adapter\n' >"$LOCAL_PROJECT/.pi/extensions/kkt-hooks.ts"
printf 'stale opencode adapter\n' >"$LOCAL_PROJECT/.opencode/plugins/kkt-hooks.ts"
cat >>"$LOCAL_PROJECT/.codex/config.toml" <<'EOF'

# kkt-hooks:start
legacy = true
# kkt-hooks:end
EOF

PROJECT="$LOCAL_PROJECT" python3 - <<'PY'
import json
import os
from pathlib import Path

path = Path(os.environ["PROJECT"]) / ".claude/settings.local.json"
data = json.loads(path.read_text(encoding="utf-8"))
for event, name in (("PreToolUse", "pre-tool"), ("PostToolUse", "post-tool")):
    data["hooks"].setdefault(event, []).append({
        "matcher": "legacy",
        "hooks": [{"type": "command", "command": f"kkt hook {name} --agent claude"}],
    })
path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
PY

run_installer "$LOCAL_HOME" upgrade --target all --local "$LOCAL_PROJECT" --hooks --bin-dir "$LOCAL_BIN"
assert_not_file "$LOCAL_PROJECT/.agents/skills/kkt/stale.txt"
assert_not_contains "$LOCAL_PROJECT/.codex/config.toml" '# kkt-hooks:start'
cmp "$ROOT/adapters/pi/kkt-hooks.ts" "$LOCAL_PROJECT/.pi/extensions/kkt-hooks.ts"
cmp "$ROOT/adapters/opencode/kkt-hooks.ts" "$LOCAL_PROJECT/.opencode/plugins/kkt-hooks.ts"
assert_json_contracts "$LOCAL_PROJECT"
pass "upgrade removes legacy state, replaces stale adapters, and remains idempotent"

run_installer "$LOCAL_HOME" install --target all --local "$LOCAL_PROJECT" --hooks --bin-dir "$LOCAL_BIN"
assert_json_contracts "$LOCAL_PROJECT"
pass "repeated install does not duplicate KKT hooks"

run_installer "$LOCAL_HOME" uninstall --target all --local "$LOCAL_PROJECT" --hooks --bin-dir "$LOCAL_BIN"
assert_not_file "$LOCAL_PROJECT/.agents/skills/kkt"
assert_not_file "$LOCAL_PROJECT/.claude/skills/kkt"
assert_not_file "$LOCAL_PROJECT/.pi/extensions/kkt-hooks.ts"
assert_not_file "$LOCAL_PROJECT/.opencode/plugins/kkt-hooks.ts"
assert_not_file "$LOCAL_BIN/kkt"
assert_file "$LOCAL_PROJECT/.codex/hooks.json"
assert_file "$LOCAL_PROJECT/.claude/settings.local.json"
assert_contains "$LOCAL_PROJECT/.codex/hooks.json" 'custom-before'
assert_contains "$LOCAL_PROJECT/.claude/settings.local.json" 'custom-claude-before'
assert_not_contains "$LOCAL_PROJECT/.codex/hooks.json" 'kkt hook pre-tool'
assert_not_contains "$LOCAL_PROJECT/.claude/settings.local.json" '"command": "kkt"'
pass "uninstall removes only KKT adapters and preserves unrelated settings"

EMPTY_PROJECT="$TMP_DIR/empty-project"
EMPTY_HOME="$TMP_DIR/empty-home"
EMPTY_BIN="$TMP_DIR/empty-bin"
mkdir -p "$EMPTY_PROJECT" "$EMPTY_HOME"
run_installer "$EMPTY_HOME" uninstall --target all --local "$EMPTY_PROJECT" --hooks --bin-dir "$EMPTY_BIN"
pass "uninstall is a no-op for absent hook configuration"

DRY_PROJECT="$TMP_DIR/dry-project"
DRY_HOME="$TMP_DIR/dry-home"
DRY_BIN="$TMP_DIR/dry-bin"
run_installer "$DRY_HOME" install --target all --local "$DRY_PROJECT" --hooks --dry-run --bin-dir "$DRY_BIN" >/dev/null
assert_not_file "$DRY_PROJECT"
assert_not_file "$DRY_BIN"
pass "dry-run does not create skill, adapter, or CLI files"

GLOBAL_HOME="$TMP_DIR/global-home"
GLOBAL_BIN="$TMP_DIR/global-bin"
mkdir -p "$GLOBAL_HOME"
run_installer "$GLOBAL_HOME" install --target all --hooks --bin-dir "$GLOBAL_BIN"
assert_global_install "$GLOBAL_HOME"
assert_file "$GLOBAL_BIN/kkt"
pass "global target paths are isolated and correct"

printf 'PASS: installer hook smoke tests\n'

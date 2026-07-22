#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd -P)"
RUNTIME="${1:-}"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/kkt-runtime-hooks.XXXXXX")"
cleanup() {
  if [ "${KKT_KEEP_TEMP:-}" = "1" ]; then
    printf 'kept runtime test directory: %s\n' "$TMP_DIR" >&2
  else
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT

if [ "$RUNTIME" != "pi" ] && [ "$RUNTIME" != "opencode" ]; then
  printf 'Usage: %s pi|opencode\n' "$0" >&2
  exit 2
fi

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

require_command git
require_command python3

BIN_DIR="$TMP_DIR/bin"
mkdir -p "$BIN_DIR"
if command -v go >/dev/null 2>&1; then
  (cd "$ROOT" && go build -trimpath -o "$BIN_DIR/kkt" ./cmd/kkt)
elif command -v kkt >/dev/null 2>&1; then
  cp "$(command -v kkt)" "$BIN_DIR/kkt"
else
  fail "go or kkt is required to build the runtime test CLI"
fi
export PATH="$BIN_DIR:$PATH"

if [ "$RUNTIME" = "pi" ]; then
  require_command pi
else
  if command -v opencode >/dev/null 2>&1; then
    OPENCODE_MODE="installed"
  elif command -v npx >/dev/null 2>&1; then
    OPENCODE_MODE="npx"
  else
    fail "opencode or npx is required; install opencode $(${OPENCODE_VERSION:-1.18.4})"
  fi
fi

setup_workspace() {
  local project="$1"
  mkdir -p "$project/src"
  git init -q "$project"
  git -C "$project" config user.email kkt-runtime-test@example.com
  git -C "$project" config user.name kkt-runtime-test
  : >"$project/.gitkeep"
  git -C "$project" add .gitkeep
  git -C "$project" commit -qm initial

  kkt_in "$project" start run 'Validate runtime hook adapter enforcement'
  kkt_in "$project" intent 'Run an isolated runtime hook test.'
  kkt_in "$project" discovery 'The adapter must invoke kkt before and after tool execution.'
  kkt_in "$project" model '## Objective Function
Validate runtime hook enforcement.

## Decision Variables and Affected Surfaces
- Decision variables: runtime test matrix.
- Affected surfaces: src/** and adapter execution.

## Constraint Functions
- Hard: preserve runtime behavior and constrain mutations to src/**.
- Soft: deterministic local validation.

## Candidate Feasibility
- Feasible: isolated runtime with a local model server.
- Rejected: external model credentials.

## Selected Optimum
Use deterministic local provider traffic and the installed adapter.

## Binding Constraints
The runtime may mutate only src/**.

## Validation Plan and Certificate
Run the adapter in the selected agent runtime and inspect filesystem and hook results.'
  kkt_in "$project" plan 'Run allowed, pre-tool blocked, and post-tool drift scenarios using a deterministic local model server. Acceptance criteria: allowed mutations complete; blocked mutations do not occur; post-tool drift is reported.'
  kkt_in "$project" guardrails configure --allowed 'src/**' --blocked '.env*' --command 'true'
  kkt_in "$project" approve 'runtime adapter test'
  kkt_in "$project" hooks arm --mode enforce --ttl 1h
}

kkt_in() {
  local project="$1"
  shift
  (cd "$project" && kkt "$@" >/dev/null)
}

start_server() {
  local case_dir="$1"
  local target="$2"
  local tool="$3"
  local ready="$case_dir/server.ready"
  local log="$case_dir/server.jsonl"
  python3 "$ROOT/tests/runtime-hooks/fake-openai-server.py" 0 "$ready" "$target" "$tool" "$log" &
  SERVER_PID=$!
  for _ in $(seq 1 100); do
    if [ -s "$ready" ]; then
      SERVER_PORT="$(cat "$ready")"
      SERVER_LOG="$log"
      return
    fi
    sleep 0.05
  done
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
  fail "local model server did not become ready"
}

stop_server() {
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
}

assert_runtime_result() {
  local case_dir="$1"
  local target="$2"
  local outcome="$3"
  local output="$case_dir/agent-output.txt"

  case "$outcome" in
    allowed)
      [ -e "$case_dir/project/$target" ] || fail "allowed mutation did not occur: $target"
      ;;
    blocked)
      [ ! -e "$case_dir/project/$target" ] || fail "blocked mutation occurred: $target"
      ;;
    post-drift)
      [ -e "$case_dir/project/$target" ] || fail "post-tool mutation did not occur: $target"
      ;;
    *)
      fail "unknown expected outcome: $outcome"
      ;;
  esac

  case "$outcome" in
    blocked|post-drift)
      local expected
      if [ "$outcome" = "blocked" ]; then
        if grep -Fq "tool target violates KKT guardrails" "$case_dir/server.jsonl" || \
          grep -Fq "shell command targets a blocked path" "$case_dir/server.jsonl"; then
          return
        fi
        expected="tool target violates KKT guardrails or shell command targets a blocked path"
      else
        expected="changed paths violate KKT hook baseline"
      fi
      grep -Fq "$expected" "$case_dir/server.jsonl" || {
        printf '%s\n' "--- agent output ---" >&2
        cat "$output" >&2 || true
        printf '%s\n' "--- server log ---" >&2
        cat "$case_dir/server.jsonl" >&2 || true
        fail "runtime did not surface expected hook result: $expected"
      }
      ;;
  esac
}

run_pi() {
  local project="$1"
  local port="$2"
  local output="$3"
  local home="$4"
  (
    cd "$project"
    HOME="$home" \
      PI_CODING_AGENT_DIR="$home/.pi/agent" \
      PI_TELEMETRY=0 \
      KKT_TEST_PORT="$port" \
      pi --print --no-session --approve \
        --provider kkt-test --model kkt-test/test \
        --extension "$ROOT/tests/runtime-hooks/pi-provider.ts" \
        "Use the available tool exactly once, then report the result."
  ) >"$output" 2>&1
}

run_opencode() {
  local project="$1"
  local port="$2"
  local output="$3"
  local home="$4"
  local args=(run --dir "$project" --model kkt-test/test --auto --format json "Use the available bash tool exactly once, then report the result.")
  (
    cd "$project"
    export HOME="$home"
    export OPENCODE_DISABLE_AUTOUPDATE=1
    export OPENCODE_DISABLE_MODELS_FETCH=1
    export OPENCODE_CONFIG="$project/opencode.json"
    export KKT_TEST_PORT="$port"
    if [ "$OPENCODE_MODE" = "installed" ]; then
      opencode "${args[@]}"
    else
      npx --yes --package "opencode-ai@${OPENCODE_VERSION:-1.18.4}" opencode "${args[@]}"
    fi
  ) >"$output" 2>&1
}

run_case() {
  local name="$1"
  local tool="$2"
  local target="$3"
  local outcome="$4"
  local case_dir="$TMP_DIR/$name"
  local project="$case_dir/project"
  local home="$case_dir/home"
  mkdir -p "$case_dir" "$home"
  setup_workspace "$project"
  start_server "$case_dir" "$target" "$tool"

  if [ "$RUNTIME" = "pi" ]; then
    mkdir -p "$project/.pi/extensions"
    cp "$ROOT/adapters/pi/kkt-hooks.ts" "$project/.pi/extensions/kkt-hooks.ts"
    set +e
    run_pi "$project" "$SERVER_PORT" "$case_dir/agent-output.txt" "$home"
    status=$?
    set -e
  else
    mkdir -p "$project/.opencode/plugins"
    cp "$ROOT/adapters/opencode/kkt-hooks.ts" "$project/.opencode/plugins/kkt-hooks.ts"
    cat >"$project/opencode.json" <<EOF
{
  "\$schema": "https://opencode.ai/config.json",
  "model": "kkt-test/test",
  "provider": {
    "kkt-test": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "KKT Runtime Test",
      "options": {
        "baseURL": "http://127.0.0.1:${SERVER_PORT}/v1",
        "apiKey": "kkt-test"
      },
      "models": {
        "test": {"name": "KKT Runtime Test"}
      }
    }
  },
  "permission": {"bash": "allow"}
}
EOF
    set +e
    run_opencode "$project" "$SERVER_PORT" "$case_dir/agent-output.txt" "$home"
    status=$?
    set -e
  fi

  stop_server
  if [ "$status" -ne 0 ]; then
    printf '%s\n' "--- agent output ($name) ---" >&2
    cat "$case_dir/agent-output.txt" >&2 || true
    fail "$RUNTIME runtime exited with status $status in $name"
  fi
  assert_runtime_result "$case_dir" "$target" "$outcome"
  printf 'ok: %s %s\n' "$RUNTIME" "$name"
}

if [ "$RUNTIME" = "pi" ]; then
  run_case allowed write src/allowed.txt allowed
  run_case blocked write outside.txt blocked
else
  run_case allowed bash src/allowed.txt allowed
  run_case blocked bash .env blocked
fi
run_case post-drift bash outside.txt post-drift

printf 'PASS: %s runtime hook tests\n' "$RUNTIME"

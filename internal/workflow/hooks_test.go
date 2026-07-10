package workflow

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookInactiveAllowsWithoutWorkspace(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root)

	payload := `{"tool_name":"write","tool_input":{"path":"outside.txt"}}`
	result := runHookJSON(t, "pre-tool", payload)
	if result.Verdict != "allow" || result.Mode != "inactive" {
		t.Fatalf("inactive hook should allow, got %#v", result)
	}
}

func TestHooksArmAndPreToolPathEnforcement(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root)
	initGit(t, root)
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "unrelated.md"), []byte("preexisting unrelated work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	startValidationRunWorkspaceWithBounds(t, nil, []string{"src/**"}, []string{".env*", ".git/**", "dist/**"})
	armHooksForTest(t, "enforce")

	statePayload, err := os.ReadFile(filepath.Join(root, ".kkt", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(statePayload); !strings.Contains(text, `"armed": true`) || !strings.Contains(text, `"docs/unrelated.md"`) {
		t.Fatalf("hook state missing armed baseline:\n%s", text)
	}

	allowed := runHookJSON(t, "pre-tool", `{"tool_name":"write","tool_input":{"path":"src/main.go"}}`)
	if allowed.Verdict != "allow" {
		t.Fatalf("allowed path should pass: %#v", allowed)
	}

	outside := runHookJSON(t, "pre-tool", `{"tool_name":"write","tool_input":{"path":"docs/out.md"}}`)
	if outside.Verdict != "block" || !strings.Contains(strings.Join(outside.Evidence, "\n"), "outside allowed") {
		t.Fatalf("outside path should block: %#v", outside)
	}

	blocked := runHookJSON(t, "pre-tool", `{"tool_name":"write","tool_input":{"path":".env"}}`)
	if blocked.Verdict != "block" || !strings.Contains(strings.Join(blocked.Evidence, "\n"), "blocked") {
		t.Fatalf("blocked path should block: %#v", blocked)
	}
}

func TestCodexApplyPatchPreToolParsesPatchPaths(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root)
	initGit(t, root)
	startValidationRunWorkspaceWithBounds(t, nil, []string{"src/**"}, []string{".env*"})
	armHooksForTest(t, "enforce")

	allowedPatch := "*** Begin Patch\n*** Add File: src/main.go\n+package main\n*** End Patch"
	allowed := runHookJSON(t, "pre-tool", codexPatchPayload(t, allowedPatch))
	if allowed.Verdict != "allow" {
		t.Fatalf("codex apply_patch inside allowed paths should pass: %#v", allowed)
	}

	outsidePatch := "*** Begin Patch\n*** Add File: docs/out.md\n+out\n*** End Patch"
	outside := runHookJSON(t, "pre-tool", codexPatchPayload(t, outsidePatch))
	if outside.Verdict != "block" || !strings.Contains(strings.Join(outside.Evidence, "\n"), "docs/out.md") {
		t.Fatalf("codex apply_patch outside allowed paths should block: %#v", outside)
	}
}

func TestPostToolBaselineAllowsPreexistingDirtyAndBlocksNewOutOfScope(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root)
	initGit(t, root)
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "unrelated.md"), []byte("preexisting unrelated work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	startValidationRunWorkspaceWithBounds(t, nil, []string{"src/**"}, []string{".env*", ".git/**", "dist/**"})
	armHooksForTest(t, "enforce")

	unchanged := runHookJSON(t, "post-tool", `{}`)
	if unchanged.Verdict != "allow" {
		t.Fatalf("unchanged preexisting dirty path should pass: %#v", unchanged)
	}

	if err := os.WriteFile(filepath.Join(root, "docs", "new.md"), []byte("new out-of-scope work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed := runHookJSON(t, "post-tool", `{}`)
	if changed.Verdict != "block" || !strings.Contains(strings.Join(changed.Evidence, "\n"), "docs/new.md") {
		t.Fatalf("new out-of-scope change should block: %#v", changed)
	}
}

func TestHooksObserveWarnsInsteadOfBlocking(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root)
	initGit(t, root)
	startValidationRunWorkspaceWithBounds(t, nil, []string{"src/**"}, []string{".env*"})
	armHooksForTest(t, "observe")

	payload := `{"tool_name":"write","tool_input":{"path":"docs/out.md"}}`
	result := runHookJSON(t, "pre-tool", payload)
	if result.Verdict != "warn" || result.Mode != "observe" {
		t.Fatalf("observe mode should warn instead of block: %#v", result)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"hook", "pre-tool", payload}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "KKT hook warning") {
		t.Fatalf("observe mode should emit an agent warning, got %q", stdout.String())
	}
}

func TestExpiredHooksAutoDisarmAndAllow(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root)
	initGit(t, root)
	startValidationRunWorkspaceWithBounds(t, nil, []string{"src/**"}, nil)
	var stdout bytes.Buffer
	if err := Run([]string{"hooks", "arm", "--ttl", "1ns", "--json"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	result := runHookJSON(t, "pre-tool", `{"tool_name":"write","tool_input":{"path":"docs/out.md"}}`)
	if result.Verdict != "allow" || !strings.Contains(result.Reason, "expired") {
		t.Fatalf("expired hook should auto-disarm and allow: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, ".kkt", "hooks.json")); !os.IsNotExist(err) {
		t.Fatalf("expired hooks should remove hooks.json, stat err=%v", err)
	}
}

func TestDoneDisarmsHooks(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root)
	initGit(t, root)
	startValidationRunWorkspaceWithBounds(t, nil, []string{"src/**"}, nil)
	armHooksForTest(t, "enforce")

	if err := Run([]string{"done", "complete"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".kkt", "hooks.json")); !os.IsNotExist(err) {
		t.Fatalf("hooks.json should be removed on done, stat err=%v", err)
	}
}

func armHooksForTest(t *testing.T, mode string) HookState {
	t.Helper()
	var stdout bytes.Buffer
	if err := Run([]string{"hooks", "arm", "--mode", mode, "--ttl", "1h", "--json"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var state HookState
	if err := json.Unmarshal(stdout.Bytes(), &state); err != nil {
		t.Fatalf("parse hook state: %v\n%s", err, stdout.String())
	}
	return state
}

func runHookJSON(t *testing.T, event, payload string) HookResult {
	t.Helper()
	var stdout bytes.Buffer
	if err := Run([]string{"hook", event, "--json", payload}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var result HookResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse hook result: %v\n%s", err, stdout.String())
	}
	return result
}

func codexPatchPayload(t *testing.T, command string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"tool_name":  "apply_patch",
		"tool_input": map[string]any{"command": command},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func withCwd(t *testing.T, root string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	})
}

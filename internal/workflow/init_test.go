package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveInstructionPreservesExistingContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	existing := strings.Join([]string{
		"# Existing",
		"",
		instructionStart,
		"# KKT Workflow",
		instructionEnd,
		"",
		"Keep this.",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := RemoveInstruction(path)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("RemoveInstruction should report changed")
	}
	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(result)
	if strings.Contains(text, instructionStart) || strings.Contains(text, instructionEnd) {
		t.Fatal("KKT markers were not removed")
	}
	if !strings.Contains(text, "# Existing") || !strings.Contains(text, "Keep this.") {
		t.Fatal("existing content was not preserved")
	}
}

func TestRemoveInstructionIgnoresMissingOrUnmanagedFiles(t *testing.T) {
	root := t.TempDir()
	missingChanged, err := RemoveInstruction(filepath.Join(root, "missing.md"))
	if err != nil {
		t.Fatal(err)
	}
	if missingChanged {
		t.Fatal("missing file should not report changed")
	}

	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unmanagedChanged, err := RemoveInstruction(path)
	if err != nil {
		t.Fatal(err)
	}
	if unmanagedChanged {
		t.Fatal("unmanaged file should not report changed")
	}
}

func TestUninstallPlansUsesGlobalAgentLocations(t *testing.T) {
	home := t.TempDir()
	plans, err := UninstallPlansWithHome("all", home)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, plan := range plans {
		if !plan.Remove {
			t.Fatalf("uninstall plan should only contain remove actions: %s", plan.Path)
		}
		paths[plan.Path] = true
	}
	expectedPaths := []string{
		filepath.Join(home, ".codex", "AGENTS.md"),
		filepath.Join(home, ".codex", "KKT.md"),
		filepath.Join(home, ".claude", "CLAUDE.md"),
		filepath.Join(home, ".claude", "KKT.md"),
		filepath.Join(home, ".pi", "agent", "AGENTS.md"),
		filepath.Join(home, ".config", "opencode", "AGENTS.md"),
		filepath.Join(home, ".agents", "AGENTS.md"),
		filepath.Join(home, ".agents", "KKT.md"),
	}
	for _, path := range expectedPaths {
		if !paths[path] {
			t.Fatalf("missing uninstall path: %s", path)
		}
	}
}

func TestUninstallPlansDedupesInstructionPaths(t *testing.T) {
	home := t.TempDir()
	plans, err := UninstallPlansWithHome("all", home)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, plan := range plans {
		if seen[plan.Path] {
			t.Fatalf("duplicate instruction path: %s", plan.Path)
		}
		seen[plan.Path] = true
	}
}

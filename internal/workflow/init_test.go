package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteInstructionPreservesExistingContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	existing := "# Existing\n\nKeep this.\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := WriteInstruction(path, instructionContent("codex", "kkt"))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("WriteInstruction should report changed")
	}
	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(result)
	if !strings.Contains(text, "Keep this.") {
		t.Fatal("existing content was not preserved")
	}
	if !strings.Contains(text, instructionStart) || !strings.Contains(text, instructionEnd) {
		t.Fatal("KKT markers were not written")
	}
}

func TestInitPlansDedupesSharedInstructionPath(t *testing.T) {
	home := t.TempDir()
	plans, err := InitPlansWithHome("all", home, "kkt")
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

func TestInitPlansCombinesSharedAgentInstructions(t *testing.T) {
	home := t.TempDir()
	plans, err := InitPlansWithHome("all", home, "/tmp/kkt")
	if err != nil {
		t.Fatal(err)
	}
	sharedPath := filepath.Join(home, ".agents", "AGENTS.md")
	for _, plan := range plans {
		if plan.Path == sharedPath {
			if !strings.Contains(plan.Agent, "codex") || !strings.Contains(plan.Agent, "opencode") || !strings.Contains(plan.Agent, "pi") {
				t.Fatalf("shared AGENTS.md plan did not include all shared agents: %q", plan.Agent)
			}
			if !strings.Contains(plan.Content, "/tmp/kkt classify") {
				t.Fatal("custom command was not rendered")
			}
			return
		}
	}
	t.Fatal("missing shared AGENTS.md plan")
}

func TestInitPlansUsesGlobalAgentLocations(t *testing.T) {
	home := t.TempDir()
	plans, err := InitPlansWithHome("all", home, "kkt")
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, plan := range plans {
		paths[plan.Path] = true
	}
	if !paths[filepath.Join(home, ".agents", "AGENTS.md")] {
		t.Fatal("missing shared ~/.agents/AGENTS.md plan")
	}
	if !paths[filepath.Join(home, ".claude", "CLAUDE.md")] {
		t.Fatal("missing ~/.claude/CLAUDE.md plan")
	}
}

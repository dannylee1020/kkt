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

func TestInitPlansDedupesInstructionPaths(t *testing.T) {
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

func TestInitPlansUsesInlineInstructionsForPiAndOpenCode(t *testing.T) {
	home := t.TempDir()
	tests := []struct {
		agent string
		path  string
	}{
		{"pi", filepath.Join(home, ".pi", "agent", "AGENTS.md")},
		{"opencode", filepath.Join(home, ".config", "opencode", "AGENTS.md")},
	}

	for _, test := range tests {
		t.Run(test.agent, func(t *testing.T) {
			plans, err := InitPlansWithHome(test.agent, home, "/tmp/kkt")
			if err != nil {
				t.Fatal(err)
			}
			plan := planByPath(plans, test.path)
			if plan == nil {
				t.Fatalf("missing %s instruction plan", test.agent)
			}
			if !strings.Contains(plan.Content, "# KKT Workflow") || !strings.Contains(plan.Content, "/tmp/kkt classify") {
				t.Fatalf("%s instruction plan did not contain inline workflow", test.agent)
			}
			if strings.Contains(plan.Content, "@"+filepath.Join(home, ".agents", "KKT.md")) {
				t.Fatalf("%s instruction plan should not reference shared KKT.md", test.agent)
			}
		})
	}
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
	if !paths[filepath.Join(home, ".codex", "AGENTS.md")] {
		t.Fatal("missing ~/.codex/AGENTS.md plan")
	}
	if !paths[filepath.Join(home, ".codex", "KKT.md")] {
		t.Fatal("missing ~/.codex/KKT.md plan")
	}
	if !paths[filepath.Join(home, ".pi", "agent", "AGENTS.md")] {
		t.Fatal("missing ~/.pi/agent/AGENTS.md plan")
	}
	if !paths[filepath.Join(home, ".config", "opencode", "AGENTS.md")] {
		t.Fatal("missing ~/.config/opencode/AGENTS.md plan")
	}
	if !paths[filepath.Join(home, ".claude", "CLAUDE.md")] {
		t.Fatal("missing ~/.claude/CLAUDE.md plan")
	}
	if !paths[filepath.Join(home, ".claude", "KKT.md")] {
		t.Fatal("missing ~/.claude/KKT.md plan")
	}
}

func TestInitPlansUsesReferenceFiles(t *testing.T) {
	home := t.TempDir()
	tests := []struct {
		agent string
		entry string
		full  string
	}{
		{"codex", filepath.Join(home, ".codex", "AGENTS.md"), filepath.Join(home, ".codex", "KKT.md")},
		{"claude", filepath.Join(home, ".claude", "CLAUDE.md"), filepath.Join(home, ".claude", "KKT.md")},
	}

	for _, test := range tests {
		t.Run(test.agent, func(t *testing.T) {
			plans, err := InitPlansWithHome(test.agent, home, "kkt")
			if err != nil {
				t.Fatal(err)
			}
			if writePlanCount(plans) != 2 {
				t.Fatalf("expected %s to create two write plans, got %d", test.agent, writePlanCount(plans))
			}

			entry := planByPath(plans, test.entry)
			if entry == nil {
				t.Fatalf("missing %s entry plan", test.agent)
			}
			if !strings.Contains(entry.Content, "@"+test.full) {
				t.Fatalf("%s entry plan did not reference KKT.md", test.agent)
			}
			if strings.Contains(entry.Content, "kkt classify") {
				t.Fatalf("%s entry plan should only contain the KKT.md reference", test.agent)
			}

			full := planByPath(plans, test.full)
			if full == nil {
				t.Fatalf("missing %s KKT.md plan", test.agent)
			}
			if !strings.Contains(full.Content, "# KKT Workflow") || !strings.Contains(full.Content, "kkt classify") {
				t.Fatalf("%s KKT.md plan did not contain full workflow instructions", test.agent)
			}
		})
	}
}

func TestInitPlansAddsLegacyCleanupForMovedAgents(t *testing.T) {
	home := t.TempDir()
	plans, err := InitPlansWithHome("opencode", home, "kkt")
	if err != nil {
		t.Fatal(err)
	}
	if !removePlanExists(plans, filepath.Join(home, ".agents", "AGENTS.md")) {
		t.Fatal("missing legacy ~/.agents/AGENTS.md cleanup")
	}
	if !removePlanExists(plans, filepath.Join(home, ".agents", "KKT.md")) {
		t.Fatal("missing legacy ~/.agents/KKT.md cleanup")
	}
}

func TestInitPlansSkipsLegacyCleanupForClaude(t *testing.T) {
	home := t.TempDir()
	plans, err := InitPlansWithHome("claude", home, "kkt")
	if err != nil {
		t.Fatal(err)
	}
	for _, plan := range plans {
		if plan.Remove {
			t.Fatalf("claude should not create cleanup plan: %s", plan.Path)
		}
	}
}

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

func planByPath(plans []InitPlan, path string) *InitPlan {
	for i := range plans {
		if plans[i].Path == path {
			return &plans[i]
		}
	}
	return nil
}

func removePlanExists(plans []InitPlan, path string) bool {
	for _, plan := range plans {
		if plan.Path == path && plan.Remove {
			return true
		}
	}
	return false
}

func writePlanCount(plans []InitPlan) int {
	count := 0
	for _, plan := range plans {
		if !plan.Remove {
			count++
		}
	}
	return count
}

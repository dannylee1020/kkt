package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartWorkflowCreatesPlanState(t *testing.T) {
	root := t.TempDir()
	workspace, err := StartWorkflow(root, "implement KKT workflow CLI", "plan")
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Path != filepath.Join(root, ".kkt") {
		t.Fatalf("Path = %q, want %q", workspace.Path, filepath.Join(root, ".kkt"))
	}
	required := []string{"kkt.yaml"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(workspace.Path, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	current, err := os.ReadFile(filepath.Join(root, ".kkt", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if len(current) == 0 {
		t.Fatal("current pointer is empty")
	}
	state, err := os.ReadFile(filepath.Join(workspace.Path, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(state); !strings.Contains(text, "workspace_type: plan") || !strings.Contains(text, "profile: plan") {
		t.Fatalf("plan state did not use plan profile:\n%s", text)
	}
}

func TestStartWorkflowCreatesModelWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace, err := StartWorkflow(root, "choose API shape", "model")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.Dir(workspace.Path), filepath.Join(root, ".kkt", "model"); got != want {
		t.Fatalf("workspace parent = %q, want %q", got, want)
	}
	required := []string{"kkt.yaml", "intent.md", "discovery.md", "model.md"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(workspace.Path, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace.Path, "plan.md")); !os.IsNotExist(err) {
		t.Fatalf("plan.md should not exist for model workspace: %v", err)
	}
}

func TestStartWorkflowCreatesLoopWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace, err := StartWorkflow(root, "run multi-step migration", "loop")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.Dir(workspace.Path), filepath.Join(root, ".kkt", "loop"); got != want {
		t.Fatalf("workspace parent = %q, want %q", got, want)
	}
	required := []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "plan.md", "progress.md", "evidence.md", "notes.md"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(workspace.Path, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestResolveWorkspaceUsesCurrentPointer(t *testing.T) {
	root := t.TempDir()
	created, err := StartWorkflow(root, "choose API shape", "model")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != created.Path {
		t.Fatalf("resolved = %q, want %q", resolved, created.Path)
	}
}

func TestValidateWorkspaceFailsPendingEvidence(t *testing.T) {
	root := t.TempDir()
	workspace, err := StartWorkflow(root, "implement KKT workflow CLI", "loop")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ValidateWorkspace(workspace.Path)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("ValidateWorkspace should fail while evidence is pending")
	}
}

func TestValidateModelWorkspaceDoesNotRequireExecutionFiles(t *testing.T) {
	root := t.TempDir()
	workspace, err := StartWorkflow(root, "choose API shape", "model")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ValidateWorkspace(workspace.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("ValidateWorkspace should accept model workspace, got issues: %v", result.Issues)
	}
}

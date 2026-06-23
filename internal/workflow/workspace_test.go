package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartWorkflowCreatesWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace, err := StartWorkflow(root, "implement KKT workflow CLI", "daily")
	if err != nil {
		t.Fatal(err)
	}
	required := []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "plan.md", "progress.md", "evidence.md", "notes.md"}
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
}

func TestValidateWorkspaceFailsPendingEvidence(t *testing.T) {
	root := t.TempDir()
	workspace, err := StartWorkflow(root, "implement KKT workflow CLI", "daily")
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

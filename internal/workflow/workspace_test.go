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
	for _, want := range []string{
		"planning_contract:",
		"objective_function:",
		"files_to_modify:",
		"constraint_functions:",
		"decision_variables:",
		"validation_proof:",
	} {
		if !strings.Contains(string(state), want) {
			t.Fatalf("plan state missing %q:\n%s", want, state)
		}
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
	required := []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "guardrails.json"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(workspace.Path, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace.Path, "plan.md")); !os.IsNotExist(err) {
		t.Fatalf("plan.md should not exist for model workspace: %v", err)
	}
	state, err := os.ReadFile(filepath.Join(workspace.Path, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(state); !strings.Contains(text, "active_layer: intent") || !strings.Contains(text, "method: pending") {
		t.Fatalf("model workspace should start with pending intent:\n%s", text)
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
	required := []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "guardrails.json", "plan.md", "progress.md", "evidence.md", "notes.md", "events.jsonl"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(workspace.Path, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	state, err := os.ReadFile(filepath.Join(workspace.Path, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(state); !strings.Contains(text, "loop_state:") || !strings.Contains(text, "acceptance_criteria:") {
		t.Fatalf("loop state block missing from kkt.yaml:\n%s", text)
	}
	if text := string(state); !strings.Contains(text, "active_layer: intent") || !strings.Contains(text, "method_invocations: []") {
		t.Fatalf("loop workspace should start with pending method selection:\n%s", text)
	}
}

func TestStartWorkflowCreatesRunWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace, err := StartWorkflow(root, "execute selected model", "run")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.Dir(workspace.Path), filepath.Join(root, ".kkt", "run"); got != want {
		t.Fatalf("workspace parent = %q, want %q", got, want)
	}
	required := []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "guardrails.json", "plan.md", "progress.md", "evidence.md", "notes.md"}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(workspace.Path, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace.Path, "events.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("events.jsonl should not exist for run workspace: %v", err)
	}
	state, err := os.ReadFile(filepath.Join(workspace.Path, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(state); !strings.Contains(text, "workspace_type: run") || !strings.Contains(text, "guardrails: guardrails.json") {
		t.Fatalf("run state missing expected contract:\n%s", text)
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

func TestStartWorkflowUsesNearestGitRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "packages", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	workspace, err := StartWorkflow(nested, "anchor workspace at project root", "plan")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := workspace.Path, filepath.Join(root, ".kkt"); got != want {
		t.Fatalf("workspace = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(nested, ".kkt")); !os.IsNotExist(err) {
		t.Fatalf("nested .kkt should not exist: %v", err)
	}
}

func TestResolveWorkspaceUsesProjectRootFromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	firstNested := filepath.Join(root, "packages", "app")
	secondNested := filepath.Join(root, "cmd", "tool")
	if err := os.MkdirAll(firstNested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secondNested, 0o755); err != nil {
		t.Fatal(err)
	}

	created, err := StartWorkflow(firstNested, "choose API shape", "model")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveWorkspace(secondNested, "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != created.Path {
		t.Fatalf("resolved = %q, want %q", resolved, created.Path)
	}
}

func TestProjectRootAcceptsGitFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: ../.git/worktrees/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "src")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	workspace, err := StartWorkflow(nested, "support worktree roots", "plan")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := workspace.Path, filepath.Join(root, ".kkt"); got != want {
		t.Fatalf("workspace = %q, want %q", got, want)
	}
}

func TestProjectRootFallsBackToStartOutsideGit(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "src")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	resolved, err := projectRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != nested {
		t.Fatalf("projectRoot = %q, want %q", resolved, nested)
	}
}

func TestResolveWorkspaceUsesExplicitPathOutsideProjectRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "src")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	created, err := StartWorkflow(external, "external explicit workspace", "model")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveWorkspace(nested, created.Path)
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

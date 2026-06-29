package workflow

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsVersion(t *testing.T) {
	previous := Version
	Version = "vtest"
	defer func() {
		Version = previous
	}()

	var stdout bytes.Buffer
	if err := Run([]string{"--version"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "kkt vtest"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRunRejectsRemovedAliasesAndFlags(t *testing.T) {
	tests := [][]string{
		{"-h"},
		{"-v"},
		{"version"},
		{"classify", "implement a feature"},
		{"start", "--profile", "plan", "implement a feature"},
		{"init", "codex"},
		{"uninstall", "--dry-run"},
		{"uninstall", "--keep-binary"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			if err := Run(test, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
				t.Fatal("expected removed alias or flag to be rejected")
			}
		})
	}
}

func TestRunStartRequiresExplicitProfile(t *testing.T) {
	var stdout bytes.Buffer
	err := Run([]string{"start", "implement", "a", "feature"}, &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected start without explicit profile to fail")
	}
	if !strings.Contains(err.Error(), "unsupported profile") {
		t.Fatalf("error = %q, want unsupported profile", err.Error())
	}
}

func TestRunRejectsDailyProfile(t *testing.T) {
	err := Run([]string{"start", "daily", "implement", "a", "feature"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected daily profile to be rejected")
	}
	if !strings.Contains(err.Error(), "unsupported profile") {
		t.Fatalf("error = %q, want unsupported profile", err.Error())
	}
}

func TestPlanArtifactCommandsStayInStateFile(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	commands := [][]string{
		{"start", "plan", "make", "the", "CLI", "durable"},
		{"model", "Use compact state only"},
		{"evidence", "validated by inspection"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "model.md")); !os.IsNotExist(err) {
		t.Fatalf("plan workspace should not create model.md: %v", err)
	}
	state, err := os.ReadFile(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(state); !strings.Contains(text, "artifact: \"model\"") || !strings.Contains(text, "artifact: \"evidence\"") {
		t.Fatalf("plan artifacts were not recorded in kkt.yaml:\n%s", text)
	}
	var stdout bytes.Buffer
	if err := Run([]string{"show", "model"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("show model failed for plan workspace: %v", err)
	}
	if !strings.Contains(stdout.String(), "decision_log:") {
		t.Fatalf("show model did not return plan state:\n%s", stdout.String())
	}
}

func TestRunStartFromNestedDirectoryUsesProjectRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "packages", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	if err := Run([]string{"start", "loop", "anchor", "state"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	projectRoot, err := projectRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(workspace) != filepath.Join(projectRoot, ".kkt", "loop") {
		t.Fatalf("workspace = %q, want parent under project root .kkt/loop", workspace)
	}
	if _, err := os.Stat(filepath.Join(nested, ".kkt")); !os.IsNotExist(err) {
		t.Fatalf("nested .kkt should not exist: %v", err)
	}
}

func TestRunLoopCommandLifecycle(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	commands := [][]string{
		{"start", "loop", "upgrade", "kkt", "loop"},
		{"approve", "Approved test loop"},
		{"task", "add", "Inspect code"},
		{"task", "start", "inspect-code"},
		{"progress", "Started inspection"},
		{"criteria", "add", "Evidence recorded"},
		{"evidence", "--for", "evidence-recorded", "--command", "go test ./...", "go test ./... passed"},
		{"criteria", "satisfy", "evidence-recorded"},
		{"task", "done", "inspect-code"},
		{"validate"},
		{"done", "Loop complete"},
	}
	for _, command := range commands {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
				t.Fatalf("Run(%v) error = %v", command, err)
			}
		})
	}

	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	state, err := os.ReadFile(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(state); !strings.Contains(text, "status: \"complete\"") || !strings.Contains(text, "current_task: \"\"") {
		t.Fatalf("unexpected final state:\n%s", text)
	}
	events, err := os.ReadFile(filepath.Join(workspace, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(events); !strings.Contains(text, "task_added") || !strings.Contains(text, "done") {
		t.Fatalf("events log missing lifecycle entries:\n%s", text)
	}
}

func TestRunNextForFreshLoopUsesActiveLayer(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if err := Run([]string{"start", "loop", "bootstrap", "guidance"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	var textNext bytes.Buffer
	if err := Run([]string{"next"}, &textNext, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := textNext.String(); !strings.Contains(text, "record discovery") || strings.Contains(text, "validate, then kkt done") {
		t.Fatalf("fresh loop next should use active layer guidance:\n%s", text)
	}

	var jsonNext bytes.Buffer
	if err := Run([]string{"next", "--json"}, &jsonNext, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := jsonNext.String(); !strings.Contains(text, `"action": "continue_layer"`) || !strings.Contains(text, `"reason": "active layer is discovery"`) {
		t.Fatalf("fresh loop next --json should continue the active layer:\n%s", text)
	}
}

func TestRunNextRequiresApprovalBeforeLoopTaskExecution(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	commands := [][]string{
		{"start", "loop", "approval", "gate"},
		{"task", "add", "Inspect code"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	var next bytes.Buffer
	if err := Run([]string{"next", "--json"}, &next, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	text := next.String()
	if !strings.Contains(text, `"action": "request_approval"`) || !strings.Contains(text, `"blocked": true`) {
		t.Fatalf("unapproved loop with work should request approval:\n%s", text)
	}
	if strings.Contains(text, `"action": "start_task"`) || strings.Contains(text, `"task_id": "inspect-code"`) {
		t.Fatalf("unapproved loop should not suggest task execution:\n%s", text)
	}
}

func TestRunNextJSONAndReplayCheck(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	commands := [][]string{
		{"start", "loop", "add", "replay", "check"},
		{"approve", "Approved replay check"},
		{"task", "add", "Inspect code"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	var next bytes.Buffer
	if err := Run([]string{"next", "--json"}, &next, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := next.String(); !strings.Contains(text, `"action": "start_task"`) || !strings.Contains(text, `"task_id": "inspect-code"`) {
		t.Fatalf("next --json output did not include structured task action:\n%s", text)
	}

	if err := Run([]string{"replay", "--check"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("replay --check should pass for consistent event/state history: %v", err)
	}
}

func TestRunResumeIncludesContinuationPacket(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	commands := [][]string{
		{"start", "loop", "resume", "context"},
		{"approve", "Approved resume check"},
		{"task", "add", "Inspect code"},
		{"task", "start", "inspect-code"},
		{"criteria", "add", "Evidence recorded"},
		{"evidence", "--for", "evidence-recorded", "Inspection evidence recorded"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	var resume bytes.Buffer
	if err := Run([]string{"resume"}, &resume, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	text := resume.String()
	for _, want := range []string{"approval: approved", "current_task: inspect-code", "unsatisfied_criteria:", "latest_evidence:", "recent_events:", "validation: invalid"} {
		if !strings.Contains(text, want) {
			t.Fatalf("resume output missing %q:\n%s", want, text)
		}
	}
}

func TestRunDoneRequiresApprovalAndMappedEvidence(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	commands := [][]string{
		{"start", "loop", "prove", "terminal", "validation"},
		{"task", "add", "Inspect code"},
		{"task", "start", "inspect-code"},
		{"criteria", "add", "Evidence recorded"},
		{"evidence", "Unmapped evidence"},
		{"criteria", "satisfy", "evidence-recorded"},
		{"task", "done", "inspect-code"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	var stdout bytes.Buffer
	err = Run([]string{"done"}, &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("done should fail without approval and mapped evidence")
	}
	text := stdout.String()
	if !strings.Contains(text, "approval is not approved") || !strings.Contains(text, "criterion evidence-recorded is satisfied without mapped evidence") {
		t.Fatalf("done output missing terminal invariant failures:\n%s", text)
	}
}

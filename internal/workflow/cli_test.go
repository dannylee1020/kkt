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
		{"task", "add", "Inspect code"},
		{"task", "start", "inspect-code"},
		{"progress", "Started inspection"},
		{"evidence", "go test ./... passed"},
		{"criteria", "add", "Evidence recorded"},
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

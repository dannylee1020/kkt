package workflow

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
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
		{"intent", "--method"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			if err := Run(test, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
				t.Fatal("expected removed alias or flag to be rejected")
			}
		})
	}
}

func TestRunArtifactRecordsLayerMethodAndAdvances(t *testing.T) {
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
		{"start", "model", "choose", "API", "shape"},
		{"intent", "--method", "why_how", "Clarified owner tradeoffs"},
		{"discovery", "--method", "coupling_map", "Mapped affected API callers"},
		{"model", "--method", "ordinal_mcda", "Compared feasible API shapes"},
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
	state, err := os.ReadFile(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(state)
	for _, want := range []string{
		`active_layer: "validation"`,
		`method: "why_how"`,
		`method: "coupling_map"`,
		`method: "ordinal_mcda"`,
		"method_invocations:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("state missing %q:\n%s", want, text)
		}
	}

	var next bytes.Buffer
	if err := Run([]string{"next"}, &next, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := next.String(); !strings.Contains(text, "run kkt validate") || strings.Contains(text, "kkt evidence") {
		t.Fatalf("model-only next should validate without evidence guidance:\n%s", text)
	}
}

func TestRunArtifactRejectsInvalidLayerMethod(t *testing.T) {
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
	if err := Run([]string{"start", "model", "choose", "API", "shape"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	err = Run([]string{"model", "--method", "goal_anti_goal", "wrong layer method"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected invalid model method to fail")
	}
	if !strings.Contains(err.Error(), "unsupported model method") {
		t.Fatalf("error = %q, want unsupported model method", err.Error())
	}
}

func TestRunGuardrailsShowAndValidate(t *testing.T) {
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
	if err := Run([]string{"start", "model", "choose", "API", "shape"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	var show bytes.Buffer
	if err := Run([]string{"guardrails", "show"}, &show, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := show.String(); !strings.Contains(text, `"schema_version": 1`) || !strings.Contains(text, `"drift_policy"`) {
		t.Fatalf("guardrails show missing contract fields:\n%s", text)
	}

	var validate bytes.Buffer
	if err := Run([]string{"guardrails", "validate"}, &validate, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := validate.String(); !strings.Contains(text, "valid:") {
		t.Fatalf("guardrails validate did not pass:\n%s", text)
	}
}

func TestRunFromModelCreatesApprovalGatedRun(t *testing.T) {
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
		{"start", "model", "choose", "API", "shape"},
		{"intent", "--method", "why_how", "Clarified owner tradeoffs"},
		{"discovery", "--method", "coupling_map", "Mapped affected API callers"},
		{"model", "--method", "ordinal_mcda", "Compared feasible API shapes"},
		{"guardrails", "set", testGuardrailsJSON("model", []string{"internal/workflow/**"}, nil)},
		{"done", "Model complete"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	var created bytes.Buffer
	if err := Run([]string{"run", "from-model"}, &created, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := created.String(); !strings.Contains(text, "profile: run") {
		t.Fatalf("run from-model output missing run profile:\n%s", text)
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(workspace)) != "run" {
		t.Fatalf("current workspace = %q, want run workspace", workspace)
	}

	var judge bytes.Buffer
	if err := Run([]string{"judge", "--checkpoint", "model-ready", "--json"}, &judge, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := judge.String(); !strings.Contains(text, `"verdict": "allow"`) || !strings.Contains(text, `"workspace_type": "run"`) {
		t.Fatalf("model-ready judge should allow complete run contract:\n%s", text)
	}

	var next bytes.Buffer
	if err := Run([]string{"next", "--json"}, &next, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := next.String(); !strings.Contains(text, `"action": "request_approval"`) || !strings.Contains(text, `"blocked": true`) {
		t.Fatalf("run next should request approval before mutation:\n%s", text)
	}

	var blocked bytes.Buffer
	err = Run([]string{"judge", "--checkpoint", "pre-mutation", "--json"}, &blocked, &bytes.Buffer{})
	if err == nil {
		t.Fatal("pre-mutation judge should block without approval")
	}
	if text := blocked.String(); !strings.Contains(text, `"verdict": "block"`) || !strings.Contains(text, `"drift_type": "approval"`) {
		t.Fatalf("pre-mutation judge output missing approval block:\n%s", text)
	}
}

func TestRunJudgeBlocksChangedPathOutsideAllowedBounds(t *testing.T) {
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
	initGit(t, root)

	commands := [][]string{
		{"start", "run", "path", "scope"},
		{"intent", "--method", "goal_anti_goal", "Captured path scope"},
		{"discovery", "--method", "traceability_matrix", "Expected workflow-only change"},
		{"model", "--method", "lexicographic", "Selected workflow-only plan"},
		{"guardrails", "set", testGuardrailsJSON("run", []string{"internal/workflow/**"}, nil)},
		{"approve", "Approved workflow-only scope"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("out of scope\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var blocked bytes.Buffer
	err = Run([]string{"judge", "--checkpoint", "pre-mutation", "--json"}, &blocked, &bytes.Buffer{})
	if err == nil {
		t.Fatal("pre-mutation judge should block changed files outside allowed paths")
	}
	if text := blocked.String(); !strings.Contains(text, `"drift_type": "path_scope"`) || !strings.Contains(text, "changed path outside allowed bounds: README.md") {
		t.Fatalf("pre-mutation judge output missing path-scope block:\n%s", text)
	}
}

func TestRunJudgeAllowsChangedPathInsideAllowedBounds(t *testing.T) {
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
	initGit(t, root)

	commands := [][]string{
		{"start", "run", "path", "scope"},
		{"intent", "--method", "goal_anti_goal", "Captured path scope"},
		{"discovery", "--method", "traceability_matrix", "Expected workflow-only change"},
		{"model", "--method", "lexicographic", "Selected workflow-only plan"},
		{"guardrails", "set", testGuardrailsJSON("run", []string{"internal/workflow/**"}, nil)},
		{"approve", "Approved workflow-only scope"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "workflow"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "workflow", "change.go"), []byte("package workflow\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var allowed bytes.Buffer
	if err := Run([]string{"judge", "--checkpoint", "pre-mutation", "--json"}, &allowed, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := allowed.String(); !strings.Contains(text, `"verdict": "allow"`) {
		t.Fatalf("pre-mutation judge should allow in-scope changes:\n%s", text)
	}
}

func TestRunJudgeBlockedPathOverridesAllowedBounds(t *testing.T) {
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
	initGit(t, root)

	commands := [][]string{
		{"start", "run", "path", "scope"},
		{"intent", "--method", "goal_anti_goal", "Captured path scope"},
		{"discovery", "--method", "traceability_matrix", "Expected broad change"},
		{"model", "--method", "lexicographic", "Selected broad plan with README blocked"},
		{"guardrails", "set", testGuardrailsJSON("run", []string{"**"}, []string{"README.md"})},
		{"approve", "Approved broad scope"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("blocked\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var blocked bytes.Buffer
	err = Run([]string{"judge", "--checkpoint", "pre-mutation", "--json"}, &blocked, &bytes.Buffer{})
	if err == nil {
		t.Fatal("pre-mutation judge should block explicitly blocked paths")
	}
	if text := blocked.String(); !strings.Contains(text, `"drift_type": "path_scope"`) || !strings.Contains(text, "changed blocked path: README.md") {
		t.Fatalf("pre-mutation judge output missing blocked-path evidence:\n%s", text)
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
		{"model", "objective_function: keep plan state compact; files_to_modify: workflow state only; constraint_functions: preserve lean kkt tier; decision_variables: typed inline contract; validation_proof: go test ./..."},
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
	var stdout bytes.Buffer
	if err := Run([]string{"show", "model"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("show model failed for plan workspace: %v", err)
	}
	if !strings.Contains(stdout.String(), "decision_log:") {
		t.Fatalf("show model did not return plan state:\n%s", stdout.String())
	}
}

func TestPlanModelRequiresPlanningContractFields(t *testing.T) {
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

	if err := Run([]string{"start", "plan", "make", "planning", "explicit"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	err = Run([]string{"model", "selected plan only"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected incomplete plan model to fail")
	}
	if !strings.Contains(err.Error(), "objective_function") || !strings.Contains(err.Error(), "validation_proof") {
		t.Fatalf("error did not list missing planning fields: %v", err)
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
		{"guardrails", "set", testGuardrailsJSON("loop", []string{"**"}, nil)},
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
	if text := textNext.String(); !strings.Contains(text, "record adaptive intent") || strings.Contains(text, "validate, then kkt done") {
		t.Fatalf("fresh loop next should use active layer guidance:\n%s", text)
	}

	var jsonNext bytes.Buffer
	if err := Run([]string{"next", "--json"}, &jsonNext, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := jsonNext.String(); !strings.Contains(text, `"action": "continue_layer"`) || !strings.Contains(text, `"reason": "active layer is intent"`) {
		t.Fatalf("fresh loop next --json should continue the active layer:\n%s", text)
	}
}

func TestRunNextRequiresApprovalAfterLoopModel(t *testing.T) {
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
		{"start", "loop", "approval", "after", "model"},
		{"intent", "--method", "goal_anti_goal", "Captured goal and anti-goal"},
		{"discovery", "--method", "traceability_matrix", "Mapped implementation surfaces"},
		{"model", "--method", "lexicographic", "Selected feasible plan"},
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
		t.Fatalf("loop after model should request approval:\n%s", text)
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

func testGuardrailsJSON(executionMode string, allowedPaths, blockedPaths []string) string {
	payload := map[string]any{
		"schema_version": 1,
		"source": map[string]any{
			"workspace_type": executionMode,
			"workspace":      "",
			"request":        "test request",
		},
		"constraints": []map[string]any{
			{
				"id":            "test-scope",
				"kind":          "scope",
				"severity":      "block",
				"statement":     "Stay inside the modeled test scope.",
				"allowed_paths": allowedPaths,
				"blocked_paths": blockedPaths,
			},
		},
		"change_bounds": map[string]any{
			"allowed_paths": allowedPaths,
			"blocked_paths": blockedPaths,
		},
		"workflow": map[string]any{
			"execution_mode":                    executionMode,
			"requires_approval_before_mutation": executionMode == "run" || executionMode == "loop",
			"requires_validation_before_done":   true,
		},
		"validation": map[string]any{
			"acceptance_criteria": []string{"test scope is enforced"},
			"required_commands":   []string{"go test ./..."},
			"evidence_required":   []string{"scope audit confirms only allowed paths changed"},
		},
		"drift_policy": map[string]any{
			"block_on": []string{"missing_approval", "empty_allowed_paths", "changed_path_outside_allowed", "changed_blocked_path", "validation_failed"},
			"warn_on":  []string{},
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func initGit(t *testing.T, root string) {
	t.Helper()
	command := exec.Command("git", "-C", root, "init")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}
}

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

func testOptimizationModel(summary string) string {
	return `## Objective Function
Optimize the requested change while preserving the current public behavior.

## Decision Variables and Affected Surfaces
- Decision variables: implementation shape and validation scope.
- Affected surfaces: internal workflow code and its tests.

## Constraint Functions
- Hard: preserve existing contracts and keep changes inside the modeled surfaces.
- Soft: prefer the smallest maintainable implementation.

## Candidate Feasibility
- Feasible: ` + summary + `.
- Rejected: broader changes that expand the requested scope.

## Selected Optimum
` + summary + `.

## Binding Constraints
The existing public contract and required validation commands constrain the change.

## Validation Plan and Certificate
Run the repository tests and inspect the final diff for scope and contract compliance.
`
}

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
		{"model", "--method", "ordinal_mcda", testOptimizationModel("Compared the feasible API shapes")},
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
	if text := next.String(); !strings.Contains(text, "kkt done") || strings.Contains(text, "kkt evidence") || strings.Contains(text, "kkt validate") {
		t.Fatalf("model-only next should finish without execution-evidence ceremony:\n%s", text)
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

	if err := Run([]string{"guardrails", "set", testGuardrailsJSON("model", []string{"**"}, nil)}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var validate bytes.Buffer
	if err := Run([]string{"guardrails", "validate"}, &validate, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := validate.String(); !strings.Contains(text, "valid:") {
		t.Fatalf("guardrails validate did not pass:\n%s", text)
	}
}

func TestGuardrailsConfigurePatchesValidatedContract(t *testing.T) {
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
	if err := Run([]string{"start", "run", "configure", "guardrails"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := Run([]string{"guardrails", "configure",
		"--allowed", "internal/workflow/**,README.md",
		"--blocked", ".env*,dist/**",
		"--command", "go test ./...",
		"--acceptance", "workflow tests pass",
		"--evidence", "test output recorded",
		"--stop", "scope expands",
		"--block-on", "changed_blocked_path,validation_failed",
		"--mode", "enforce",
	}, &output, &bytes.Buffer{}); err != nil {
		t.Fatalf("guardrails configure failed: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "configured: guardrails") {
		t.Fatalf("configure output = %q", output.String())
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := readGuardrails(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got := contract.ChangeBounds.AllowedPaths; len(got) != 2 || got[0] != "internal/workflow/**" || got[1] != "README.md" {
		t.Fatalf("allowed paths = %#v", got)
	}
	if len(contract.Validation.RequiredCommands) != 1 || contract.Validation.RequiredCommands[0] != "go test ./..." {
		t.Fatalf("required commands = %#v", contract.Validation.RequiredCommands)
	}
	if contract.Mode != "enforce" || !contract.Workflow.RequiresApprovalBeforeMutation || !contract.Workflow.RequiresValidationBeforeDone {
		t.Fatalf("configuration changed protected workflow gates: %#v mode=%q", contract.Workflow, contract.Mode)
	}
	if issues := validateGuardrails(workspace); len(issues) != 0 {
		t.Fatalf("configured guardrails should validate: %v", issues)
	}
}

func TestStatusJSONIncludesWorkflowDiagnostics(t *testing.T) {
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
	if err := Run([]string{"start", "loop", "status", "diagnostics"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, command := range [][]string{
		{"task", "add", "Inspect status projection"},
		{"criteria", "add", "Status exposes actionable diagnostics"},
	} {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	var status bytes.Buffer
	if err := Run([]string{"status", "--json"}, &status, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	text := status.String()
	var payload map[string]any
	if err := json.Unmarshal(status.Bytes(), &payload); err != nil {
		t.Fatalf("status JSON is invalid: %v\n%s", err, text)
	}
	for _, want := range []string{
		`"workspace_type": "loop"`,
		`"contract_version": "2"`,
		`"layers"`,
		`"guardrails"`,
		`"replay"`,
		`"task_counts"`,
		`"criterion_counts"`,
		`"next"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("status JSON missing %q:\n%s", want, text)
		}
	}
}

func TestRunGuardrailsSetRejectsInvalidContract(t *testing.T) {
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
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(workspace, "guardrails.json"))
	if err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(testGuardrailsJSON("model", []string{"**"}, nil), `"workspace":".kkt/test-workspace"`, `"workspace":""`, 1)
	var output bytes.Buffer
	err = Run([]string{"guardrails", "set", invalid}, &output, &bytes.Buffer{})
	if err == nil {
		t.Fatal("guardrails set should reject an invalid contract")
	}
	if text := output.String(); !strings.Contains(text, "source.workspace is required") {
		t.Fatalf("guardrails set output missing source workspace issue:\n%s", text)
	}
	after, err := os.ReadFile(filepath.Join(workspace, "guardrails.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("invalid guardrails set should not replace the existing contract")
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
		{"model", "--method", "ordinal_mcda", testOptimizationModel("Compared the feasible API shapes")},
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
	stateBytes, err := os.ReadFile(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	stateText := string(stateBytes)
	for _, want := range []string{
		`intent:`,
		`status: "complete"`,
		`method: "imported"`,
		`method_invocations:`,
	} {
		if !strings.Contains(stateText, want) {
			t.Fatalf("run state missing imported model marker %q:\n%s", want, stateText)
		}
	}

	var judge bytes.Buffer
	err = Run([]string{"judge", "--checkpoint", "model-ready", "--json"}, &judge, &bytes.Buffer{})
	if err == nil {
		t.Fatal("model-ready judge should block before the execution plan is recorded")
	}
	if text := judge.String(); !strings.Contains(text, `"verdict": "block"`) || !strings.Contains(text, "execution layer is pending") {
		t.Fatalf("model-ready judge should report the missing execution contract:\n%s", text)
	}
	if err := Run([]string{"plan", "Execute the imported selected model."}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	judge.Reset()
	if err := Run([]string{"judge", "--checkpoint", "model-ready", "--json"}, &judge, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := judge.String(); !strings.Contains(text, `"verdict": "allow"`) || !strings.Contains(text, `"workspace_type": "run"`) {
		t.Fatalf("model-ready judge should allow a complete run contract:\n%s", text)
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

func TestExecutionFromModelRejectsIncompleteAndNonModelSources(t *testing.T) {
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
	if err := Run([]string{"start", "model", "incomplete", "decision"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"loop", "from-model"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "complete model workspace") {
		t.Fatalf("loop import should reject incomplete model, got %v", err)
	}
	if err := Run([]string{"start", "run", "not", "a", "model"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"run", "from-model"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "requires a model workspace") {
		t.Fatalf("run import should reject non-model workspace, got %v", err)
	}
}

func TestLoopFromModelCreatesApprovalGatedLoop(t *testing.T) {
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
		{"start", "model", "migrate", "workflow", "state"},
		{"intent", "--method", "why_how", "Clarified migration tradeoffs"},
		{"discovery", "--method", "coupling_map", "Mapped workflow callers"},
		{"model", "--method", "ordinal_mcda", testOptimizationModel("Selected the staged migration")},
		{"guardrails", "set", testGuardrailsJSON("model", []string{"internal/workflow/**"}, nil)},
		{"done", "Model complete"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}
	source, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	sourceModel, err := os.ReadFile(filepath.Join(source, "model.md"))
	if err != nil {
		t.Fatal(err)
	}
	sourceState, err := os.ReadFile(filepath.Join(source, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	var created bytes.Buffer
	if err := Run([]string{"loop", "from-model", source}, &created, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := created.String(); !strings.Contains(text, "profile: loop") || !strings.Contains(text, "tasks, and criteria") {
		t.Fatalf("loop from-model output missing loop guidance:\n%s", text)
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.Base(filepath.Dir(workspace)), "loop"; got != want {
		t.Fatalf("current workspace = %q, want loop workspace", workspace)
	}
	for _, name := range []string{"events.jsonl", "plan.md", "progress.md", "evidence.md"} {
		if _, err := os.Stat(filepath.Join(workspace, name)); err != nil {
			t.Fatalf("imported loop missing %s: %v", name, err)
		}
	}
	state, err := os.ReadFile(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`workspace_type: loop`, `method: "imported"`, `active_layer: "execution"`} {
		if !strings.Contains(string(state), want) {
			t.Fatalf("loop state missing %q:\n%s", want, state)
		}
	}
	guardrails, err := os.ReadFile(filepath.Join(workspace, "guardrails.json"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(guardrails); !strings.Contains(text, `"execution_mode": "loop"`) || !strings.Contains(text, filepath.Base(source)) {
		t.Fatalf("imported loop guardrails lost model provenance:\n%s", text)
	}
	if currentModel, err := os.ReadFile(filepath.Join(source, "model.md")); err != nil || string(currentModel) != string(sourceModel) {
		t.Fatalf("source model changed during loop import: %v", err)
	}
	if currentState, err := os.ReadFile(filepath.Join(source, "kkt.yaml")); err != nil || string(currentState) != string(sourceState) {
		t.Fatalf("source state changed during loop import: %v", err)
	}

	var next bytes.Buffer
	if err := Run([]string{"next", "--json"}, &next, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := next.String(); !strings.Contains(text, `"action": "materialize_execution_contract"`) {
		t.Fatalf("imported loop should require execution-contract materialization:\n%s", text)
	}
	for _, command := range [][]string{
		{"plan", "Execute the imported migration."},
		{"task", "add", "Migrate workflow state"},
		{"criteria", "add", "Migration validation passes"},
	} {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}
	var judge bytes.Buffer
	if err := Run([]string{"judge", "--checkpoint", "model-ready", "--json"}, &judge, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := judge.String(); !strings.Contains(text, `"verdict": "allow"`) {
		t.Fatalf("materialized imported loop should be model-ready:\n%s", text)
	}
	if err := Run([]string{"approve", "Approved imported migration"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
}

func TestExecutionContractChangeInvalidatesApproval(t *testing.T) {
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
	startValidationRunWorkspace(t, nil)

	if err := Run([]string{"guardrails", "set", testGuardrailsJSON("run", []string{"**"}, nil)}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	state, err := ReadState(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if state.ApprovalStatus != "pending" || state.Status != "modeling" {
		t.Fatalf("contract change should invalidate approval, got %#v", state)
	}
	var judge bytes.Buffer
	err = Run([]string{"judge", "--checkpoint", "pre-mutation", "--json"}, &judge, &bytes.Buffer{})
	if err == nil {
		t.Fatal("pre-mutation should block after execution-contract changes invalidate approval")
	}
	if text := judge.String(); !strings.Contains(text, `"drift_type": "approval"`) {
		t.Fatalf("expected approval block after contract change:\n%s", text)
	}
}

func TestModelChangeRequiresReplannedExecutionBeforeApproval(t *testing.T) {
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
	startValidationRunWorkspace(t, nil)

	if err := Run([]string{"model", "--method", "lexicographic", testOptimizationModel("Selected the revised execution model")}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	state, err := ReadState(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if state.ApprovalStatus != "pending" {
		t.Fatalf("model change should invalidate approval, got %#v", state)
	}
	statuses, err := layerStatuses(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if statuses["execution"] != "pending" {
		t.Fatalf("model change should require a new execution plan, got %#v", statuses)
	}
	if err := Run([]string{"approve", "Attempt approval with stale plan"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("approval should reject a model revision until the execution plan is refreshed")
	}
	if err := Run([]string{"plan", "Execute the revised selected model."}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"approve", "Approved revised model"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
}

func TestRunJudgeAllowsUnrelatedChangedPathOutsideAllowedBounds(t *testing.T) {
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
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the workflow-only plan")},
		{"guardrails", "set", testGuardrailsJSON("run", []string{"internal/workflow/**"}, nil)},
		{"plan", "Execute the selected workflow-only plan."},
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

	var allowed bytes.Buffer
	if err := Run([]string{"judge", "--checkpoint", "pre-mutation", "--json"}, &allowed, &bytes.Buffer{}); err != nil {
		t.Fatalf("unrelated path outside allowed bounds should not block:\n%s", allowed.String())
	}
	if text := allowed.String(); !strings.Contains(text, `"verdict": "allow"`) || strings.Contains(text, "changed path outside allowed bounds: README.md") {
		t.Fatalf("pre-mutation judge should ignore unrelated out-of-scope path:\n%s", text)
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
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the workflow-only plan")},
		{"guardrails", "set", testGuardrailsJSON("run", []string{"internal/workflow/**"}, nil)},
		{"plan", "Execute the selected workflow-only plan."},
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

func TestApprovalBaselineAndUnrelatedDirtyPathDoNotBlock(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("preexisting\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	commands := [][]string{
		{"start", "run", "path", "scope"},
		{"intent", "--method", "goal_anti_goal", "Captured path scope"},
		{"discovery", "--method", "traceability_matrix", "Expected workflow-only change"},
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the workflow-only plan")},
		{"guardrails", "set", testGuardrailsJSON("run", []string{"internal/workflow/**"}, nil)},
		{"plan", "Execute the selected workflow-only plan."},
		{"approve", "Approved workflow-only scope"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	var allowed bytes.Buffer
	if err := Run([]string{"judge", "--checkpoint", "pre-mutation", "--json"}, &allowed, &bytes.Buffer{}); err != nil {
		t.Fatalf("unchanged preexisting dirty path should not block:\n%s", allowed.String())
	}
	if text := allowed.String(); !strings.Contains(text, `"verdict": "allow"`) {
		t.Fatalf("pre-mutation judge should allow unchanged preexisting dirty path:\n%s", text)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed after approval\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stillAllowed bytes.Buffer
	if err := Run([]string{"judge", "--checkpoint", "pre-mutation", "--json"}, &stillAllowed, &bytes.Buffer{}); err != nil {
		t.Fatalf("changed unrelated out-of-scope path should not block:\n%s", stillAllowed.String())
	}
	if text := stillAllowed.String(); !strings.Contains(text, `"verdict": "allow"`) || strings.Contains(text, "changed path outside allowed bounds: README.md") {
		t.Fatalf("pre-mutation judge should ignore changed unrelated out-of-scope path:\n%s", text)
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
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the broad plan with README blocked")},
		{"guardrails", "set", testGuardrailsJSON("run", []string{"**"}, []string{"README.md"})},
		{"plan", "Execute the selected broad plan."},
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

func TestStartRunCreatesCompactDurableWorkspace(t *testing.T) {
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

	var stdout bytes.Buffer
	if err := Run([]string{"start", "run", "legacy", "workflow"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := stdout.String(); strings.Contains(text, "deprecated") || strings.Contains(text, "warning:") || strings.Contains(text, "run from-model") {
		t.Fatalf("start run should be a supported direct workflow without deprecation output:\n%s", text)
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		t.Fatal(err)
	}
	state, err := ReadState(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if state.WorkspaceType != "run" || state.ContractVersion != "2" {
		t.Fatalf("direct run state = %#v, want run contract version 2", state)
	}
}

func TestTaskStartEnforcesApprovalAtTransition(t *testing.T) {
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
		{"start", "loop", "gate", "task", "start"},
		{"intent", "--method", "goal_anti_goal", "Captured loop goal"},
		{"discovery", "--method", "traceability_matrix", "Mapped loop state"},
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the loop plan")},
		{"plan", "Execute the selected loop plan."},
		{"task", "add", "Inspect code"},
		{"criteria", "add", "Evidence recorded"},
		{"guardrails", "set", testGuardrailsJSON("loop", []string{"**"}, nil)},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	err = Run([]string{"task", "start", "inspect-code"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "pre-mutation checkpoint") || !strings.Contains(err.Error(), "mutation requires approval") {
		t.Fatalf("task start should enforce approval internally, got %v", err)
	}
}

func TestNextBlocksReplayDrift(t *testing.T) {
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
		{"start", "loop", "detect", "replay", "drift"},
		{"intent", "--method", "goal_anti_goal", "Captured loop goal"},
		{"discovery", "--method", "traceability_matrix", "Mapped loop state"},
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the loop plan")},
		{"plan", "Execute the selected loop plan."},
		{"task", "add", "Inspect code"},
		{"criteria", "add", "Evidence recorded"},
		{"guardrails", "set", testGuardrailsJSON("loop", []string{"**"}, nil)},
		{"approve", "Approved replay check"},
		{"task", "start", "inspect-code"},
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
	loop, err := readLoopState(workspace)
	if err != nil {
		t.Fatal(err)
	}
	loop.CurrentTask = ""
	loop.Tasks[0].Status = "pending"
	if err := writeLoopState(workspace, loop); err != nil {
		t.Fatal(err)
	}

	var next bytes.Buffer
	if err := Run([]string{"next", "--json"}, &next, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if text := next.String(); !strings.Contains(text, `"action": "resolve_replay_drift"`) || !strings.Contains(text, `"blocked": true`) {
		t.Fatalf("next should block replay drift:\n%s", text)
	}
}

func TestDoneRunsFinalizePathChecks(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	startValidationRunWorkspaceWithBounds(t, nil, []string{"**"}, []string{"README.md"})
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("blocked after approval\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err = Run([]string{"done", "should block"}, &stdout, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "changed paths violate guardrail bounds") {
		t.Fatalf("done should enforce finalize path checks, got %v", err)
	}
	if text := stdout.String(); !strings.Contains(text, "changed blocked path: README.md") {
		t.Fatalf("done should report blocked-path evidence:\n%s", text)
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
		{"model", testOptimizationModel("Keep plan state compact with a typed inline contract")},
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

func TestPlanDonePerformsFinalizeWithoutManualJudge(t *testing.T) {
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
		{"start", "plan", "make", "compact", "planning", "complete"},
		{"model", "## Objective Function\nComplete compact planning.\n## Known Constraints\n- Explicit: none.\n## Decision Variables\nNone — fixed workflow.\n## Affected Surfaces\nWorkflow state.\n## Constraint Functions\n- Hard: preserve the simple plan tier.\n- Soft: minimize durable state and ceremony.\n## Candidate Feasibility\n- Feasible: compact contract.\n## Selected Plan\nRecord the compact contract.\n## Binding Constraints\nMinimal durable state.\n## Validation Plan and Proof\ngo test ./...\n## Execution Implications\nNone.\n## Guardrail Variables\nNone.\n## Analysis Extensions\nNone.\n## Residual Risk\nValidation command availability."},
		{"approve", "Approved compact plan"},
		{"evidence", "Compact-plan evidence recorded."},
		{"done", "Compact plan complete"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
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
		{"intent", "--method", "goal_anti_goal", "Captured loop goal"},
		{"discovery", "--method", "traceability_matrix", "Mapped loop state"},
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the loop plan")},
		{"plan", "Execute the selected loop plan."},
		{"task", "add", "Inspect code"},
		{"criteria", "add", "Evidence recorded"},
		{"guardrails", "set", testGuardrailsJSON("loop", []string{"**"}, nil)},
		{"approve", "Approved test loop"},
		{"task", "start", "inspect-code"},
		{"progress", "Started inspection"},
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

func TestRunNextRequiresExecutionContractAfterLoopModel(t *testing.T) {
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
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the feasible plan")},
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
	if !strings.Contains(text, `"action": "materialize_execution_contract"`) || strings.Contains(text, `"blocked": true`) {
		t.Fatalf("loop after model should require execution-contract materialization:\n%s", text)
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
	if !strings.Contains(text, `"action": "continue_layer"`) || !strings.Contains(text, `"reason": "active layer is intent"`) {
		t.Fatalf("loop with early task state should still complete modeling first:\n%s", text)
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
		{"intent", "--method", "goal_anti_goal", "Captured loop goal"},
		{"discovery", "--method", "traceability_matrix", "Mapped loop state"},
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the loop plan")},
		{"plan", "Execute the selected loop plan."},
		{"task", "add", "Inspect code"},
		{"criteria", "add", "Evidence recorded"},
		{"guardrails", "set", testGuardrailsJSON("loop", []string{"**"}, nil)},
		{"approve", "Approved replay check"},
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
		{"intent", "--method", "goal_anti_goal", "Captured loop goal"},
		{"discovery", "--method", "traceability_matrix", "Mapped loop state"},
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the loop plan")},
		{"plan", "Execute the selected loop plan."},
		{"task", "add", "Inspect code"},
		{"criteria", "add", "Evidence recorded"},
		{"guardrails", "set", testGuardrailsJSON("loop", []string{"**"}, nil)},
		{"approve", "Approved resume check"},
		{"task", "start", "inspect-code"},
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
	for _, want := range []string{"approval: approved", "current_task: inspect-code", "unsatisfied_criteria:", "latest_evidence:", "recent_events:", "replay: ok", "validation: invalid"} {
		if !strings.Contains(text, want) {
			t.Fatalf("resume output missing %q:\n%s", want, text)
		}
	}
}

func TestCriteriaSatisfyRequiresMappedEvidence(t *testing.T) {
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
		{"intent", "--method", "goal_anti_goal", "Captured loop goal"},
		{"discovery", "--method", "traceability_matrix", "Mapped loop state"},
		{"model", "--method", "lexicographic", testOptimizationModel("Selected the loop plan")},
		{"plan", "Execute the selected loop plan."},
		{"task", "add", "Inspect code"},
		{"criteria", "add", "Evidence recorded"},
		{"guardrails", "set", testGuardrailsJSON("loop", []string{"**"}, nil)},
		{"approve", "Approved terminal validation"},
		{"evidence", "Unmapped evidence"},
	}
	for _, command := range commands {
		if err := Run(command, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run(%v) error = %v", command, err)
		}
	}

	var stdout bytes.Buffer
	err = Run([]string{"criteria", "satisfy", "evidence-recorded"}, &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("criteria satisfy should require mapped evidence")
	}
	if text := err.Error(); !strings.Contains(text, "requires mapped evidence") || !strings.Contains(text, "kkt evidence --for evidence-recorded") {
		t.Fatalf("criteria satisfy error should explain required evidence: %v", err)
	}
}

func TestValidateReportsMissingRequiredCommandProof(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	startValidationRunWorkspace(t, []string{"printf ok"})

	var stdout bytes.Buffer
	err = Run([]string{"validate"}, &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("validate should fail when a required command has no proof")
	}
	if text := stdout.String(); !strings.Contains(text, "required command not run: printf ok") || !strings.Contains(text, "kkt validate --run") {
		t.Fatalf("validate output missing command proof issue:\n%s", text)
	}
}

func TestValidateModelDoesNotRequireCommandProof(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	startValidationModelWorkspace(t, []string{"printf ok"})

	var stdout bytes.Buffer
	if err := Run([]string{"validate"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("model validate should not require command proof: %v\n%s", err, stdout.String())
	}
	if text := stdout.String(); !strings.Contains(text, "valid:") {
		t.Fatalf("model validate output missing valid status:\n%s", text)
	}
}

func TestValidateRunRecordsRequiredCommandProof(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	workspace := startValidationRunWorkspace(t, []string{"printf ok"})

	var stdout bytes.Buffer
	if err := Run([]string{"validate", "--run", "--timeout", "5s"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("validate --run error = %v\n%s", err, stdout.String())
	}
	if text := stdout.String(); !strings.Contains(text, "passed: printf ok") || !strings.Contains(text, "valid:") {
		t.Fatalf("validate --run output missing pass and valid status:\n%s", text)
	}
	events, err := readEvents(workspace, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sawCommandProof bool
	for _, event := range events {
		if event.Type == "validation_command_passed" && event.Data["command"] == "printf ok" {
			sawCommandProof = true
		}
	}
	if !sawCommandProof {
		t.Fatalf("events missing validation_command_passed proof: %#v", events)
	}
}

func TestValidateRunFailsWhenRequiredCommandFails(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	startValidationRunWorkspace(t, []string{"false"})

	var stdout bytes.Buffer
	err = Run([]string{"validate", "--run", "--timeout", "5s"}, &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("validate --run should fail when a required command fails")
	}
	if text := stdout.String(); !strings.Contains(text, "failed: false") || !strings.Contains(text, "log:") {
		t.Fatalf("validate --run output missing failure details:\n%s", text)
	}
}

func TestValidateReportsStaleRequiredCommandProof(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	startValidationRunWorkspace(t, []string{"true"})

	if err := Run([]string{"validate", "--run", "--timeout", "5s"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "changed.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err = Run([]string{"validate"}, &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("validate should fail when command proof is stale")
	}
	if text := stdout.String(); !strings.Contains(text, "required command proof is stale: true") {
		t.Fatalf("validate output missing stale proof issue:\n%s", text)
	}
}

func TestValidateIgnoresUnrelatedChangedPathForRequiredCommandProof(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	startValidationRunWorkspaceWithBounds(t, []string{"true"}, []string{"internal/workflow/**"}, nil)

	if err := Run([]string{"validate", "--run", "--timeout", "5s"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("unrelated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := Run([]string{"validate"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("validate should ignore unrelated changed path: %v\n%s", err, stdout.String())
	}
	if text := stdout.String(); !strings.Contains(text, "valid:") || strings.Contains(text, "required command proof is stale") {
		t.Fatalf("validate should keep command proof fresh for unrelated change:\n%s", text)
	}
}

func TestStatusReportsStaleCompleteWorkspace(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	startValidationRunWorkspace(t, []string{"true"})
	if err := Run([]string{"validate", "--run", "--timeout", "5s"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"done", "Run complete"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "changed.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var status bytes.Buffer
	if err := Run([]string{"status", "--json"}, &status, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	text := status.String()
	if !strings.Contains(text, `"stale_complete": true`) || !strings.Contains(text, `"action": "repair_invalid_completion"`) {
		t.Fatalf("status --json should report stale complete repair action:\n%s", text)
	}
}

func TestJudgeFinalizeBlocksMissingRequiredCommandProof(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
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
	startValidationRunWorkspace(t, []string{"printf ok"})

	var stdout bytes.Buffer
	err = Run([]string{"judge", "--checkpoint", "finalize", "--json"}, &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("finalize judge should block when required command proof is missing")
	}
	if text := stdout.String(); !strings.Contains(text, `"verdict": "block"`) || !strings.Contains(text, "required command not run: printf ok") {
		t.Fatalf("judge output missing validation proof issue:\n%s", text)
	}
}

func startValidationModelWorkspace(t *testing.T, requiredCommands []string) string {
	t.Helper()
	commands := [][]string{
		{"start", "model", "choose", "API", "shape"},
		{"intent", "--method", "why_how", "Clarified owner tradeoffs"},
		{"discovery", "--method", "coupling_map", "Mapped affected API callers"},
		{"model", "--method", "ordinal_mcda", testOptimizationModel("Compared the feasible API shapes")},
		{"guardrails", "set", testGuardrailsJSONWithCommands("model", []string{"**"}, nil, requiredCommands)},
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
	return workspace
}

func startValidationRunWorkspace(t *testing.T, requiredCommands []string) string {
	t.Helper()
	return startValidationRunWorkspaceWithBounds(t, requiredCommands, []string{"**"}, nil)
}

func startValidationRunWorkspaceWithBounds(t *testing.T, requiredCommands, allowedPaths, blockedPaths []string) string {
	t.Helper()
	commands := [][]string{
		{"start", "run", "execute", "selected", "model"},
		{"intent", "--method", "why_how", "Clarified owner tradeoffs"},
		{"discovery", "--method", "coupling_map", "Mapped affected API callers"},
		{"model", "--method", "ordinal_mcda", testOptimizationModel("Compared the feasible API shapes")},
		{"guardrails", "set", testGuardrailsJSONWithCommands("run", allowedPaths, blockedPaths, requiredCommands)},
		{"plan", "Run selected validation model."},
		{"approve", "Approved validation run"},
		{"evidence", "Narrative evidence recorded."},
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
	return workspace
}

func testGuardrailsJSON(executionMode string, allowedPaths, blockedPaths []string) string {
	return testGuardrailsJSONWithCommands(executionMode, allowedPaths, blockedPaths, nil)
}

func testGuardrailsJSONWithCommands(executionMode string, allowedPaths, blockedPaths, requiredCommands []string) string {
	payload := map[string]any{
		"schema_version": 1,
		"source": map[string]any{
			"workspace_type": executionMode,
			"workspace":      ".kkt/test-workspace",
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
			"required_commands":   requiredCommands,
			"evidence_required":   []string{"scope audit confirms only allowed paths changed"},
		},
		"drift_policy": map[string]any{
			"block_on": []string{"missing_approval", "empty_allowed_paths", "changed_blocked_path", "validation_failed"},
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

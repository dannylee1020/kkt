package workflow

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Workspace struct {
	Path    string
	Profile string
}

type State struct {
	SchemaVersion  string
	WorkspaceType  string
	Profile        string
	Status         string
	ActiveLayer    string
	ApprovalStatus string
}

type ValidationResult struct {
	OK     bool
	Issues []string
}

func StartWorkflow(root, request, profile string) (Workspace, error) {
	if profile != "plan" && profile != "loop" && profile != "model" && profile != "run" {
		return Workspace{}, fmt.Errorf("unsupported profile %q", profile)
	}
	projectRootDir, err := projectRoot(root)
	if err != nil {
		return Workspace{}, err
	}
	now := time.Now().UTC()
	slug := fmt.Sprintf("%s-%s", now.Format("20060102-150405"), slugify(request))
	base := filepath.Join(projectRootDir, ".kkt")
	workspace := workspacePath(base, profile, slug)
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return Workspace{}, err
	}

	files := workspaceFiles(request, profile, now)
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(content), 0o644); err != nil {
			return Workspace{}, err
		}
	}
	if err := os.WriteFile(filepath.Join(base, "current"), []byte(currentPointer(profile, slug)+"\n"), 0o644); err != nil {
		return Workspace{}, err
	}
	return Workspace{Path: workspace, Profile: profile}, nil
}

func ResolveWorkspace(root, candidate string) (string, error) {
	if candidate != "" {
		info, err := os.Stat(candidate)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", fmt.Errorf("workspace path is not a directory: %s", candidate)
		}
		return candidate, nil
	}

	projectRootDir, err := projectRoot(root)
	if err != nil {
		return "", err
	}
	base := filepath.Join(projectRootDir, ".kkt")
	current, err := os.ReadFile(filepath.Join(base, "current"))
	if err == nil {
		path := filepath.Clean(filepath.Join(base, strings.TrimSpace(string(current))))
		if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
			return path, nil
		}
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("no .kkt workspace found; run kkt start plan|model|run|loop first")
		}
		return "", err
	}
	type workspaceCandidate struct {
		sortKey string
		path    string
	}
	var dirs []workspaceCandidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "model" || entry.Name() == "run" || entry.Name() == "loop" {
			nested, readErr := os.ReadDir(filepath.Join(base, entry.Name()))
			if readErr != nil {
				return "", readErr
			}
			for _, nestedEntry := range nested {
				if nestedEntry.IsDir() {
					dirs = append(dirs, workspaceCandidate{
						sortKey: nestedEntry.Name(),
						path:    filepath.Join(base, entry.Name(), nestedEntry.Name()),
					})
				}
			}
			continue
		}
		if entry.Name() != "." {
			dirs = append(dirs, workspaceCandidate{
				sortKey: entry.Name(),
				path:    filepath.Join(base, entry.Name()),
			})
		}
	}
	if len(dirs) == 0 {
		return "", errors.New("no .kkt workspace found; run kkt start plan|model|run|loop first")
	}
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].sortKey < dirs[j].sortKey
	})
	return dirs[len(dirs)-1].path, nil
}

func projectRoot(start string) (string, error) {
	root, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		root = filepath.Dir(root)
	}
	fallback := root
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			return root, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(root)
		if parent == root {
			return fallback, nil
		}
		root = parent
	}
}

func ReadState(workspace string) (State, error) {
	file, err := os.Open(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		return State{}, err
	}
	defer file.Close()

	state := State{}
	inApproval := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if line == "approval:" {
			inApproval = true
			continue
		}
		if rawLine != "" && !strings.HasPrefix(rawLine, " ") && !strings.HasPrefix(rawLine, "-") && line != "approval:" {
			inApproval = false
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch key {
		case "schema_version":
			state.SchemaVersion = value
		case "workspace_type":
			state.WorkspaceType = value
		case "profile":
			state.Profile = value
		case "status":
			if inApproval {
				state.ApprovalStatus = value
			} else if state.Status == "" {
				state.Status = value
			}
		case "active_layer":
			state.ActiveLayer = value
		}
	}
	if err := scanner.Err(); err != nil {
		return State{}, err
	}
	if state.ApprovalStatus == "" {
		state.ApprovalStatus = "pending"
	}
	return state, nil
}

func NextInstruction(state State) string {
	switch state.ActiveLayer {
	case "intent":
		return "next: record adaptive intent with kkt intent --method <method>, then inspect the repo and record discovery with kkt discovery --method <method>"
	case "discovery":
		if state.WorkspaceType == "plan" {
			return "next: inspect the repo, then record objective_function, files_to_modify, constraint_functions, decision_variables, and validation_proof with kkt model before edits"
		}
		return "next: record discovery with repo facts, constraints, validation paths, and unknowns using kkt discovery --method <method>"
	case "modeling":
		if state.WorkspaceType == "plan" {
			return "next: record objective_function, files_to_modify, constraint_functions, decision_variables, and validation_proof with kkt model; then get explicit approval before edits"
		}
		return "next: record the selected model with kkt model --method <method>, show it, and get explicit approval before edits"
	case "execution":
		if (state.WorkspaceType == "loop" || state.WorkspaceType == "run") && state.ApprovalStatus != "approved" {
			return "next: show the selected model and record approval with kkt approve before execution"
		}
		return "next: execute only the approved plan and record progress with kkt progress"
	case "validation":
		if state.WorkspaceType == "model" {
			return "next: run kkt validate, then finish the decision brief with kkt done"
		}
		return "next: run validation, record evidence with kkt evidence, then finish with kkt done"
	default:
		return "next: inspect kkt.yaml and continue from the active layer"
	}
}

func ValidateWorkspace(workspace string) (ValidationResult, error) {
	state, err := ReadState(workspace)
	if err != nil {
		return ValidationResult{}, err
	}
	required := requiredFiles(state.WorkspaceType)
	result := ValidationResult{OK: true}
	for _, name := range required {
		path := filepath.Join(workspace, name)
		info, err := os.Stat(path)
		if err != nil {
			result.OK = false
			result.Issues = append(result.Issues, fmt.Sprintf("missing %s", name))
			continue
		}
		if info.Size() == 0 && name != "events.jsonl" {
			result.OK = false
			result.Issues = append(result.Issues, fmt.Sprintf("empty %s", name))
		}
	}
	if state.WorkspaceType == "plan" {
		for _, issue := range planContractIssues(workspace) {
			result.OK = false
			result.Issues = append(result.Issues, issue)
		}
	}
	if state.WorkspaceType == "run" || state.WorkspaceType == "loop" || (state.WorkspaceType == "model" && state.ActiveLayer == "validation") {
		if contract, guardrailErr := readGuardrails(workspace); guardrailErr == nil {
			for _, issue := range guardrailExecutionReadinessIssues(contract) {
				result.OK = false
				result.Issues = append(result.Issues, issue)
			}
		}
	}
	evidence, err := os.ReadFile(filepath.Join(workspace, "evidence.md"))
	if state.WorkspaceType == "run" && err == nil && strings.Contains(string(evidence), "Status: pending") {
		result.OK = false
		result.Issues = append(result.Issues, "evidence.md is still pending")
	}
	if state.WorkspaceType == "loop" && err == nil && strings.Contains(string(evidence), "Status: pending") {
		result.OK = false
		result.Issues = append(result.Issues, "evidence.md is still pending")
	}
	if state.WorkspaceType == "run" && state.ApprovalStatus != "approved" {
		result.OK = false
		result.Issues = append(result.Issues, "approval is not approved")
	}
	if state.WorkspaceType == "loop" {
		if state.ApprovalStatus != "approved" {
			result.OK = false
			result.Issues = append(result.Issues, "approval is not approved")
		}
		loop, loopErr := readLoopState(workspace)
		if loopErr != nil {
			result.OK = false
			result.Issues = append(result.Issues, loopErr.Error())
		} else {
			for _, task := range loop.Tasks {
				if task.Status != "done" && task.Status != "skipped" {
					result.OK = false
					result.Issues = append(result.Issues, fmt.Sprintf("task %s is %s", task.ID, task.Status))
				}
			}
			for _, criterion := range loop.AcceptanceCriteria {
				if criterion.Status != "satisfied" {
					result.OK = false
					result.Issues = append(result.Issues, fmt.Sprintf("criterion %s is %s", criterion.ID, criterion.Status))
				}
			}
			for _, stop := range loop.StopConditions {
				if stop.Status == "active" {
					result.OK = false
					result.Issues = append(result.Issues, fmt.Sprintf("stop condition active: %s", stop.Text))
				}
			}
			if len(loop.Evidence) == 0 {
				result.OK = false
				result.Issues = append(result.Issues, "no loop evidence recorded")
			}
			for _, issue := range evidenceMappingIssues(loop) {
				result.OK = false
				result.Issues = append(result.Issues, issue)
			}
		}
	}
	return result, nil
}

func planContractIssues(workspace string) []string {
	content, err := os.ReadFile(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		return []string{err.Error()}
	}
	text := string(content)
	var issues []string
	for _, field := range requiredPlanContractFields() {
		if !strings.Contains(text, field+":") {
			issues = append(issues, "missing plan contract field "+field)
		}
	}
	return issues
}

func requiredPlanContractFields() []string {
	return []string{
		"planning_contract",
		"objective_function",
		"files_to_modify",
		"constraint_functions",
		"decision_variables",
		"validation_proof",
	}
}

func stateYAML(request, profile string, now time.Time) string {
	escapedRequest := strings.ReplaceAll(request, `"`, `\"`)
	if profile == "plan" {
		return fmt.Sprintf(`schema_version: 1
workflow_type: kkt
workspace_type: plan
profile: plan
status: modeling
active_layer: modeling
created_at: %s
request: "%s"
layers:
  intent:
    status: complete
    method: goal_anti_goal
    summary: "Initial user request captured by kkt start plan."
  discovery:
    status: pending
    method: traceability_matrix
    summary: ""
  modeling:
    status: pending
    method: lexicographic
    summary: ""
decision_log: []
planning_contract:
  objective_function:
    status: pending
    summary: ""
  files_to_modify:
    status: pending
    items: []
  constraint_functions:
    status: pending
    hard: []
    soft: []
  decision_variables:
    status: pending
    items: []
  validation_proof:
    status: pending
    commands: []
    evidence: []
artifact_refs:
  state: kkt.yaml
approval:
  required: true
  status: pending
  approved_scope: ""
stop_conditions:
  - "No feasible plan satisfies hard constraints."
  - "User does not approve the selected model."
  - "Destructive action, credentials, paid service, or external access is required."
`, now.Format(time.RFC3339), escapedRequest)
	}
	if profile == "model" {
		return fmt.Sprintf(`schema_version: 1
workflow_type: kkt
workspace_type: model
profile: model
status: modeling
active_layer: intent
created_at: %s
request: "%s"
layers:
  intent:
    status: pending
    method: pending
    summary: ""
    artifact: intent.md
  discovery:
    status: pending
    method: pending
    summary: ""
    artifact: discovery.md
  modeling:
    status: pending
    method: pending
    summary: ""
    artifact: model.md
method_invocations: []
decision_log: []
artifact_refs:
  intent: intent.md
  discovery: discovery.md
  model: model.md
  guardrails: guardrails.json
approval:
  required: false
  status: not_required
  approved_scope: ""
stop_conditions:
  - "No feasible model satisfies hard constraints."
  - "A product, risk, scope, or implementation tradeoff cannot be resolved from repository evidence."
`, now.Format(time.RFC3339), escapedRequest)
	}
	if profile == "run" {
		return fmt.Sprintf(`schema_version: 1
workflow_type: kkt
workspace_type: run
profile: run
status: modeling
active_layer: intent
created_at: %s
request: "%s"
layers:
  intent:
    status: pending
    method: pending
    summary: ""
    artifact: intent.md
  discovery:
    status: pending
    method: pending
    summary: ""
    artifact: discovery.md
  modeling:
    status: pending
    method: pending
    summary: ""
    artifact: model.md
  execution:
    status: pending
    method: pending
    summary: ""
    artifact: plan.md
  validation:
    status: pending
    method: pending
    summary: ""
    artifact: evidence.md
method_invocations: []
decision_log: []
artifact_refs:
  intent: intent.md
  discovery: discovery.md
  model: model.md
  guardrails: guardrails.json
  plan: plan.md
  progress: progress.md
  evidence: evidence.md
  notes: notes.md
approval:
  required: true
  status: pending
  approved_scope: ""
stop_conditions:
  - "No feasible plan satisfies hard constraints."
  - "User does not approve the selected model."
  - "Model-ready judge blocks execution."
  - "Destructive action, credentials, paid service, or external access is required."
`, now.Format(time.RFC3339), escapedRequest)
	}
	return fmt.Sprintf(`schema_version: 1
workflow_type: kkt
workspace_type: loop
profile: loop
status: modeling
active_layer: intent
created_at: %s
request: "%s"
layers:
  intent:
    status: pending
    method: pending
    summary: ""
    artifact: intent.md
  discovery:
    status: pending
    method: pending
    summary: ""
    artifact: discovery.md
  modeling:
    status: pending
    method: pending
    summary: ""
    artifact: model.md
  execution:
    status: pending
    method: pending
    summary: ""
    artifact: plan.md
  validation:
    status: pending
    method: pending
    summary: ""
    artifact: evidence.md
method_invocations: []
decision_log: []
artifact_refs:
  intent: intent.md
  discovery: discovery.md
  model: model.md
  guardrails: guardrails.json
  plan: plan.md
  progress: progress.md
  evidence: evidence.md
  notes: notes.md
  events: events.jsonl
approval:
  required: true
  status: pending
  approved_scope: ""
stop_conditions:
  - "No feasible plan satisfies hard constraints."
  - "User does not approve the selected model."
  - "Destructive action, credentials, paid service, or external access is required."
loop_state:
  current_task: ""
  tasks:
  acceptance_criteria:
  evidence:
  stop_conditions:
    - id: "no-feasible-plan"
      text: "No feasible plan satisfies hard constraints."
      status: "clear"
    - id: "missing-approval"
      text: "User does not approve the selected model."
      status: "clear"
    - id: "unsafe-action"
      text: "Destructive action, credentials, paid service, or external access is required."
      status: "clear"
`, now.Format(time.RFC3339), escapedRequest)
}

func workspacePath(base, profile, slug string) string {
	switch profile {
	case "plan":
		return base
	case "model":
		return filepath.Join(base, "model", slug)
	case "run":
		return filepath.Join(base, "run", slug)
	default:
		return filepath.Join(base, "loop", slug)
	}
}

func currentPointer(profile, slug string) string {
	switch profile {
	case "plan":
		return "."
	case "model":
		return filepath.Join("model", slug)
	case "run":
		return filepath.Join("run", slug)
	default:
		return filepath.Join("loop", slug)
	}
}

func workspaceFiles(request, profile string, now time.Time) map[string]string {
	files := map[string]string{
		"kkt.yaml": stateYAML(request, profile, now),
	}
	if profile == "plan" {
		return files
	}
	files["intent.md"] = intentMarkdown(request)
	files["discovery.md"] = "# Discovery\n\nStatus: pending\n\nRecord repo facts, discovered constraints, validation paths, and remaining unknowns here.\n"
	files["model.md"] = "# Model\n\nStatus: pending\n\nRecord method selection, objective function, known constraints, files to modify, constraint functions, decision variables, candidate feasibility, selected plan, binding constraints, validation proof, execution implications, and residual risk here.\n"
	files["guardrails.json"] = defaultGuardrailsJSON(request, profile, "")
	if profile == "model" {
		return files
	}
	files["plan.md"] = "# Plan\n\nStatus: pending\n\nRecord acceptance criteria, validation plan, evidence required, stop conditions, and continuation policy here.\n"
	files["progress.md"] = "# Progress\n\nStatus: pending\n\n- [ ] Complete discovery\n- [ ] Complete model\n- [ ] Get approval before implementation\n- [ ] Execute approved plan\n- [ ] Validate with evidence\n"
	files["evidence.md"] = "# Evidence\n\nStatus: pending\n\nRecord validation commands, outputs, artifacts, and final constraint audit here.\n"
	files["notes.md"] = "# Notes\n\n"
	if profile == "loop" {
		files["events.jsonl"] = ""
	}
	return files
}

func requiredFiles(workspaceType string) []string {
	switch workspaceType {
	case "plan":
		return []string{"kkt.yaml"}
	case "model":
		return []string{"kkt.yaml", "intent.md", "discovery.md", "model.md"}
	case "run":
		return []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "guardrails.json", "plan.md", "progress.md", "evidence.md", "notes.md"}
	default:
		return []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "guardrails.json", "plan.md", "progress.md", "evidence.md", "notes.md", "events.jsonl"}
	}
}

func intentMarkdown(request string) string {
	return fmt.Sprintf(`# Intent

Status: pending

## User Goal

%s

## Desired Behavior

To be refined by the agent after discovery if the request has hidden product choices.

## User-Visible Success

The selected implementation satisfies the request while preserving discovered constraints.

## Scope Boundary

One existing coding agent uses KKT as a workflow tool. KKT does not own the session, spawn subagents, or route models.
`, request)
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(value string) string {
	value = strings.ToLower(value)
	value = nonSlug.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	parts := strings.Split(value, "-")
	if len(parts) > 8 {
		parts = parts[:8]
	}
	value = strings.Join(parts, "-")
	if value == "" {
		return "request"
	}
	return value
}

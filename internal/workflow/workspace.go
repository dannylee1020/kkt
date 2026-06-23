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
	if profile != "daily" && profile != "loop" && profile != "model" {
		return Workspace{}, fmt.Errorf("unsupported profile %q", profile)
	}
	now := time.Now().UTC()
	slug := fmt.Sprintf("%s-%s", now.Format("20060102-150405"), slugify(request))
	base := filepath.Join(root, ".kkt")
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

	base := filepath.Join(root, ".kkt")
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
			return "", errors.New("no .kkt workspace found; run kkt start first")
		}
		return "", err
	}
	type candidate struct {
		sortKey string
		path    string
	}
	var dirs []candidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "model" || entry.Name() == "loop" {
			nested, readErr := os.ReadDir(filepath.Join(base, entry.Name()))
			if readErr != nil {
				return "", readErr
			}
			for _, nestedEntry := range nested {
				if nestedEntry.IsDir() {
					dirs = append(dirs, candidate{
						sortKey: nestedEntry.Name(),
						path:    filepath.Join(base, entry.Name(), nestedEntry.Name()),
					})
				}
			}
			continue
		}
		if entry.Name() != "." {
			dirs = append(dirs, candidate{
				sortKey: entry.Name(),
				path:    filepath.Join(base, entry.Name()),
			})
		}
	}
	if len(dirs) == 0 {
		return "", errors.New("no .kkt workspace found; run kkt start first")
	}
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].sortKey < dirs[j].sortKey
	})
	return dirs[len(dirs)-1].path, nil
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
		return "next: complete intent.md, then inspect the repo and record discovery.md"
	case "discovery":
		if state.WorkspaceType == "daily" {
			return "next: keep compact state in .kkt/kkt.yaml, inspect the repo, and record the selected model before edits"
		}
		return "next: complete discovery.md with repo facts, constraints, validation paths, and unknowns"
	case "modeling":
		if state.WorkspaceType == "daily" {
			return "next: record the selected model in .kkt/kkt.yaml and get explicit approval before edits"
		}
		return "next: complete model.md, show the selected model, and get explicit approval before edits"
	case "execution":
		return "next: execute only the approved plan and update progress.md"
	case "validation":
		return "next: run validation, update evidence.md, and finish with a constraint audit"
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
		if info.Size() == 0 {
			result.OK = false
			result.Issues = append(result.Issues, fmt.Sprintf("empty %s", name))
		}
	}
	evidence, err := os.ReadFile(filepath.Join(workspace, "evidence.md"))
	if state.WorkspaceType == "loop" && err == nil && strings.Contains(string(evidence), "Status: pending") {
		result.OK = false
		result.Issues = append(result.Issues, "evidence.md is still pending")
	}
	return result, nil
}

func stateYAML(request, profile string, now time.Time) string {
	escapedRequest := strings.ReplaceAll(request, `"`, `\"`)
	if profile == "daily" {
		return fmt.Sprintf(`schema_version: 1
workflow_type: kkt
workspace_type: daily
profile: daily
status: modeling
active_layer: modeling
created_at: %s
request: "%s"
layers:
  intent:
    status: complete
    method: goal_anti_goal
    summary: "Initial user request captured by kkt start."
  discovery:
    status: pending
    method: traceability_matrix
    summary: ""
  modeling:
    status: pending
    method: lexicographic
    summary: ""
decision_log: []
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
active_layer: discovery
created_at: %s
request: "%s"
layers:
  intent:
    status: complete
    method: goal_anti_goal
    summary: "Initial user request captured by kkt start."
    artifact: intent.md
  discovery:
    status: pending
    method: traceability_matrix
    summary: ""
    artifact: discovery.md
  modeling:
    status: pending
    method: lexicographic
    summary: ""
    artifact: model.md
decision_log: []
artifact_refs:
  intent: intent.md
  discovery: discovery.md
  model: model.md
approval:
  required: false
  status: not_required
  approved_scope: ""
stop_conditions:
  - "No feasible model satisfies hard constraints."
  - "A product, risk, scope, or implementation tradeoff cannot be resolved from repository evidence."
`, now.Format(time.RFC3339), escapedRequest)
	}
	return fmt.Sprintf(`schema_version: 1
workflow_type: kkt
workspace_type: loop
profile: loop
status: modeling
active_layer: discovery
created_at: %s
request: "%s"
layers:
  intent:
    status: complete
    method: goal_anti_goal
    summary: "Initial user request captured by kkt start."
    artifact: intent.md
  discovery:
    status: pending
    method: traceability_matrix
    summary: ""
    artifact: discovery.md
  modeling:
    status: pending
    method: lexicographic
    summary: ""
    artifact: model.md
  execution:
    status: pending
    method: contract_preserving_change
    summary: ""
    artifact: plan.md
  validation:
    status: pending
    method: hard_constraint_audit
    summary: ""
    artifact: evidence.md
method_invocations:
  - layer: intent
    method: goal_anti_goal
    reason: "Capture rough request before discovery."
    inputs: "user request"
    outputs: intent.md
decision_log: []
artifact_refs:
  intent: intent.md
  discovery: discovery.md
  model: model.md
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
  - "Destructive action, credentials, paid service, or external access is required."
`, now.Format(time.RFC3339), escapedRequest)
}

func workspacePath(base, profile, slug string) string {
	switch profile {
	case "daily":
		return base
	case "model":
		return filepath.Join(base, "model", slug)
	default:
		return filepath.Join(base, "loop", slug)
	}
}

func currentPointer(profile, slug string) string {
	switch profile {
	case "daily":
		return "."
	case "model":
		return filepath.Join("model", slug)
	default:
		return filepath.Join("loop", slug)
	}
}

func workspaceFiles(request, profile string, now time.Time) map[string]string {
	files := map[string]string{
		"kkt.yaml": stateYAML(request, profile, now),
	}
	if profile == "daily" {
		return files
	}
	files["intent.md"] = intentMarkdown(request)
	files["discovery.md"] = "# Discovery\n\nStatus: pending\n\nRecord repo facts, discovered constraints, validation paths, and remaining unknowns here.\n"
	files["model.md"] = "# Model\n\nStatus: pending\n\nRecord candidate plans, feasibility checks, selected plan, binding constraints, and residual risk here.\n"
	if profile == "model" {
		return files
	}
	files["plan.md"] = "# Plan\n\nStatus: pending\n\nRecord acceptance criteria, validation plan, evidence required, stop conditions, and continuation policy here.\n"
	files["progress.md"] = "# Progress\n\nStatus: pending\n\n- [ ] Complete discovery\n- [ ] Complete model\n- [ ] Get approval before implementation\n- [ ] Execute approved plan\n- [ ] Validate with evidence\n"
	files["evidence.md"] = "# Evidence\n\nStatus: pending\n\nRecord validation commands, outputs, artifacts, and final constraint audit here.\n"
	files["notes.md"] = "# Notes\n\n"
	return files
}

func requiredFiles(workspaceType string) []string {
	switch workspaceType {
	case "daily":
		return []string{"kkt.yaml"}
	case "model":
		return []string{"kkt.yaml", "intent.md", "discovery.md", "model.md"}
	default:
		return []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "plan.md", "progress.md", "evidence.md", "notes.md"}
	}
}

func intentMarkdown(request string) string {
	return fmt.Sprintf(`# Intent

Status: complete

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

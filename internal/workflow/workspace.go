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
	workspace := filepath.Join(base, slug)
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return Workspace{}, err
	}

	files := map[string]string{
		"kkt.yaml":    stateYAML(request, profile, now),
		"intent.md":   intentMarkdown(request),
		"discovery.md": "# Discovery\n\nStatus: pending\n\nRecord repo facts, discovered constraints, validation paths, and remaining unknowns here.\n",
		"model.md":    "# Model\n\nStatus: pending\n\nRecord candidate plans, feasibility checks, selected plan, binding constraints, and residual risk here.\n",
		"plan.md":     "# Plan\n\nStatus: pending\n\nRecord acceptance criteria, validation plan, evidence required, stop conditions, and continuation policy here.\n",
		"progress.md": "# Progress\n\nStatus: pending\n\n- [ ] Complete discovery\n- [ ] Complete model\n- [ ] Get approval before implementation\n- [ ] Execute approved plan\n- [ ] Validate with evidence\n",
		"evidence.md": "# Evidence\n\nStatus: pending\n\nRecord validation commands, outputs, artifacts, and final constraint audit here.\n",
		"notes.md":    "# Notes\n\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(content), 0o644); err != nil {
			return Workspace{}, err
		}
	}
	if err := os.WriteFile(filepath.Join(base, "current"), []byte(slug+"\n"), 0o644); err != nil {
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
		path := filepath.Join(base, strings.TrimSpace(string(current)))
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
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	if len(dirs) == 0 {
		return "", errors.New("no .kkt workspace found; run kkt start first")
	}
	sort.Strings(dirs)
	return filepath.Join(base, dirs[len(dirs)-1]), nil
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
		return "next: complete discovery.md with repo facts, constraints, validation paths, and unknowns"
	case "modeling":
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
	required := []string{"kkt.yaml", "intent.md", "discovery.md", "model.md", "plan.md", "progress.md", "evidence.md", "notes.md"}
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
	if err == nil && strings.Contains(string(evidence), "Status: pending") {
		result.OK = false
		result.Issues = append(result.Issues, "evidence.md is still pending")
	}
	return result, nil
}

func stateYAML(request, profile string, now time.Time) string {
	escapedRequest := strings.ReplaceAll(request, `"`, `\"`)
	return fmt.Sprintf(`schema_version: 1
workflow_type: kkt
workspace_type: loop
profile: %s
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
`, profile, now.Format(time.RFC3339), escapedRequest)
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

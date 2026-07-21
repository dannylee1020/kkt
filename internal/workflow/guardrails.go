package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type GuardrailContract struct {
	SchemaVersion          int                   `json:"schema_version"`
	Mode                   string                `json:"mode,omitempty"`
	Source                 GuardrailSource       `json:"source"`
	Constraints            []GuardrailConstraint `json:"constraints,omitempty"`
	Scope                  GuardrailScope        `json:"scope,omitempty"`
	ChangeBounds           GuardrailBounds       `json:"change_bounds"`
	Workflow               GuardrailWorkflow     `json:"workflow"`
	Validation             GuardrailChecks       `json:"validation"`
	StopConditions         []string              `json:"stop_conditions,omitempty"`
	DriftPolicy            GuardrailDriftPolicy  `json:"drift_policy"`
	GeneratedAt            string                `json:"generated_at,omitempty"`
	ModelReadyCheckpoint   []string              `json:"model_ready_checkpoint,omitempty"`
	ContinuationCheckpoint []string              `json:"continuation_checkpoint,omitempty"`
	FinalizeCheckpoint     []string              `json:"finalize_checkpoint,omitempty"`
	ImplementationNotes    []string              `json:"implementation_notes,omitempty"`
}

type GuardrailSource struct {
	WorkspaceType string `json:"workspace_type"`
	Workspace     string `json:"workspace"`
	Request       string `json:"request"`
}

type GuardrailScope struct {
	Allowed []string `json:"allowed,omitempty"`
	Blocked []string `json:"blocked,omitempty"`
}

type GuardrailConstraint struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Severity     string   `json:"severity"`
	Statement    string   `json:"statement"`
	AllowedPaths []string `json:"allowed_paths,omitempty"`
	BlockedPaths []string `json:"blocked_paths,omitempty"`
}

type GuardrailBounds struct {
	AllowedPaths                          []string `json:"allowed_paths,omitempty"`
	BlockedPaths                          []string `json:"blocked_paths,omitempty"`
	AllowedPathsOrSurfaces                []string `json:"allowed_paths_or_surfaces,omitempty"`
	BlockedPathsOrSurfaces                []string `json:"blocked_paths_or_surfaces,omitempty"`
	RequireExplicitApprovalOutsideAllowed bool     `json:"require_explicit_approval_outside_allowed,omitempty"`
}

type GuardrailWorkflow struct {
	ExecutionMode                  string `json:"execution_mode"`
	RequiresApprovalBeforeMutation bool   `json:"requires_approval_before_mutation"`
	RequiresValidationBeforeDone   bool   `json:"requires_validation_before_done"`
}

type GuardrailChecks struct {
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	RequiredCommands   []string `json:"required_commands"`
	EvidenceRequired   []string `json:"evidence_required"`
}

type GuardrailDriftPolicy struct {
	BlockOn []string `json:"block_on,omitempty"`
	WarnOn  []string `json:"warn_on,omitempty"`
	Legacy  string   `json:"-"`
}

type guardrailConfigureOptions struct {
	AllowedPaths       []string
	AllowedSet         bool
	BlockedPaths       []string
	BlockedSet         bool
	AcceptanceCriteria []string
	AcceptanceSet      bool
	RequiredCommands   []string
	CommandsSet        bool
	EvidenceRequired   []string
	EvidenceSet        bool
	StopConditions     []string
	StopsSet           bool
	BlockOn            []string
	BlockOnSet         bool
	WarnOn             []string
	WarnOnSet          bool
	Mode               string
	ModeSet            bool
}

func (policy *GuardrailDriftPolicy) UnmarshalJSON(data []byte) error {
	var legacy string
	if err := json.Unmarshal(data, &legacy); err == nil {
		policy.Legacy = legacy
		return nil
	}
	type driftPolicy GuardrailDriftPolicy
	var next driftPolicy
	if err := json.Unmarshal(data, &next); err != nil {
		return err
	}
	*policy = GuardrailDriftPolicy(next)
	return nil
}

type JudgeResult struct {
	SchemaVersion int      `json:"schema_version"`
	Verdict       string   `json:"verdict"`
	Checkpoint    string   `json:"checkpoint"`
	Mode          string   `json:"mode"`
	Workspace     string   `json:"workspace,omitempty"`
	WorkspaceType string   `json:"workspace_type,omitempty"`
	ActiveLayer   string   `json:"active_layer,omitempty"`
	DriftType     string   `json:"drift_type,omitempty"`
	Reason        string   `json:"reason"`
	Repair        []string `json:"repair,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
}

type ApprovalBaseline struct {
	SchemaVersion int               `json:"schema_version"`
	RecordedAt    string            `json:"recorded_at"`
	Paths         map[string]string `json:"paths"`
}

func runGuardrails(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("guardrails requires an action: show, set, or validate")
	}
	action := args[0]
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	path := filepath.Join(workspace, "guardrails.json")
	switch action {
	case "show":
		if len(args) > 1 {
			return errors.New("guardrails show accepts no content")
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = stdout.Write(payload)
		return err
	case "set":
		content, err := commandContent(args[1:])
		if err != nil {
			return err
		}
		content = strings.TrimSpace(content)
		if content == "" {
			return errors.New("guardrails set requires JSON content")
		}
		var contract GuardrailContract
		if err := json.Unmarshal([]byte(content), &contract); err != nil {
			return err
		}
		if issues := validateCompleteGuardrailContract(contract); len(issues) > 0 {
			for _, issue := range issues {
				fmt.Fprintf(stdout, "- %s\n", issue)
			}
			return errors.New("guardrails contract is invalid")
		}
		if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
			return err
		}
		if err := invalidateExecutionApproval(workspace, "guardrails changed"); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "recorded: guardrails")
		return nil
	case "configure":
		return runGuardrailConfigure(path, args[1:], workspace, stdout)
	case "validate":
		if len(args) > 1 {
			return errors.New("guardrails validate accepts no content")
		}
		issues := validateGuardrails(workspace)
		if len(issues) == 0 {
			fmt.Fprintf(stdout, "valid: %s\n", path)
			return nil
		}
		fmt.Fprintf(stdout, "invalid: %s\n", path)
		for _, issue := range issues {
			fmt.Fprintf(stdout, "- %s\n", issue)
		}
		return errors.New("guardrails validation failed")
	default:
		return fmt.Errorf("unsupported guardrails action %q", action)
	}
}

func runGuardrailConfigure(path string, args []string, workspace string, stdout io.Writer) error {
	options, err := parseGuardrailConfigureArgs(args)
	if err != nil {
		return err
	}
	contract, err := readGuardrails(workspace)
	if err != nil {
		return err
	}
	if options.AllowedSet {
		contract.ChangeBounds.AllowedPaths = options.AllowedPaths
		contract.ChangeBounds.AllowedPathsOrSurfaces = nil
	}
	if options.BlockedSet {
		contract.ChangeBounds.BlockedPaths = options.BlockedPaths
		contract.ChangeBounds.BlockedPathsOrSurfaces = nil
	}
	if options.AcceptanceSet {
		contract.Validation.AcceptanceCriteria = options.AcceptanceCriteria
	}
	if options.CommandsSet {
		contract.Validation.RequiredCommands = options.RequiredCommands
	}
	if options.EvidenceSet {
		contract.Validation.EvidenceRequired = options.EvidenceRequired
	}
	if options.StopsSet {
		contract.StopConditions = options.StopConditions
	}
	if options.BlockOnSet {
		contract.DriftPolicy.BlockOn = options.BlockOn
		contract.DriftPolicy.Legacy = ""
	}
	if options.WarnOnSet {
		contract.DriftPolicy.WarnOn = options.WarnOn
	}
	if options.ModeSet {
		contract.Mode = options.Mode
	}
	if issues := validateCompleteGuardrailContract(contract); len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(stdout, "- %s\n", issue)
		}
		return errors.New("guardrails contract is invalid")
	}
	payload, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		return err
	}
	if err := invalidateExecutionApproval(workspace, "guardrails changed"); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "configured: guardrails")
	return nil
}

func parseGuardrailConfigureArgs(args []string) (guardrailConfigureOptions, error) {
	options := guardrailConfigureOptions{}
	for i := 0; i < len(args); i++ {
		flag := args[i]
		next := func() (string, error) {
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return "", fmt.Errorf("guardrails configure %s requires a value", flag)
			}
			return strings.TrimSpace(args[i]), nil
		}
		switch flag {
		case "--allowed":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			options.AllowedPaths = append(options.AllowedPaths, splitCSV(value)...)
			options.AllowedSet = true
		case "--blocked":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			options.BlockedPaths = append(options.BlockedPaths, splitCSV(value)...)
			options.BlockedSet = true
		case "--acceptance":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			options.AcceptanceCriteria = append(options.AcceptanceCriteria, value)
			options.AcceptanceSet = true
		case "--command":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			options.RequiredCommands = append(options.RequiredCommands, value)
			options.CommandsSet = true
		case "--evidence":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			options.EvidenceRequired = append(options.EvidenceRequired, value)
			options.EvidenceSet = true
		case "--stop":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			options.StopConditions = append(options.StopConditions, value)
			options.StopsSet = true
		case "--block-on":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			options.BlockOn = append(options.BlockOn, splitCSV(value)...)
			options.BlockOnSet = true
		case "--warn-on":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			options.WarnOn = append(options.WarnOn, splitCSV(value)...)
			options.WarnOnSet = true
		case "--mode":
			value, err := next()
			if err != nil {
				return guardrailConfigureOptions{}, err
			}
			if value != "observe" && value != "enforce" {
				return guardrailConfigureOptions{}, fmt.Errorf("guardrails configure --mode must be observe or enforce")
			}
			options.Mode = value
			options.ModeSet = true
		default:
			return guardrailConfigureOptions{}, fmt.Errorf("guardrails configure does not accept flag: %s", flag)
		}
	}
	if !options.AllowedSet && !options.BlockedSet && !options.AcceptanceSet && !options.CommandsSet && !options.EvidenceSet && !options.StopsSet && !options.BlockOnSet && !options.WarnOnSet && !options.ModeSet {
		return guardrailConfigureOptions{}, errors.New("guardrails configure requires at least one configuration flag")
	}
	options.AllowedPaths = uniqueStrings(options.AllowedPaths)
	options.BlockedPaths = uniqueStrings(options.BlockedPaths)
	options.AcceptanceCriteria = uniqueStrings(options.AcceptanceCriteria)
	options.RequiredCommands = uniqueStrings(options.RequiredCommands)
	options.EvidenceRequired = uniqueStrings(options.EvidenceRequired)
	options.StopConditions = uniqueStrings(options.StopConditions)
	options.BlockOn = uniqueStrings(options.BlockOn)
	options.WarnOn = uniqueStrings(options.WarnOn)
	return options, nil
}

func runRun(args []string, stdout io.Writer) error {
	return runExecutionFromModel(args, "run", stdout)
}

func runLoop(args []string, stdout io.Writer) error {
	return runExecutionFromModel(args, "loop", stdout)
}

func runExecutionFromModel(args []string, profile string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s requires an action: from-model", profile)
	}
	if args[0] != "from-model" {
		return fmt.Errorf("unsupported %s action %q", profile, args[0])
	}
	if len(args) > 2 {
		return fmt.Errorf("%s from-model accepts at most one model workspace", profile)
	}
	workspace, err := createExecutionFromModel(".", firstArg(args[1:]), profile)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "created: %s\n", workspace.Path)
	fmt.Fprintf(stdout, "profile: %s\n", profile)
	if profile == "loop" {
		fmt.Fprintln(stdout, "next: materialize the execution plan, tasks, and criteria, then approve before mutation")
	} else {
		fmt.Fprintln(stdout, "next: materialize the execution plan, then approve before mutation")
	}
	return nil
}

func runJudge(args []string, stdout io.Writer) error {
	checkpoint, jsonOutput, path, err := parseJudgeArgs(args)
	if err != nil {
		return err
	}
	result, err := judgeWorkspace(".", path, checkpoint)
	if err != nil {
		return err
	}
	if jsonOutput {
		if err := writeJSON(stdout, result); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(stdout, "%s: %s\n", result.Verdict, result.Reason)
		for _, repair := range result.Repair {
			fmt.Fprintf(stdout, "- %s\n", repair)
		}
	}
	if result.Verdict == "block" {
		return errors.New(result.Reason)
	}
	return nil
}

func parseJudgeArgs(args []string) (string, bool, string, error) {
	checkpoint := ""
	jsonOutput := false
	path := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--checkpoint":
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return "", false, "", errors.New("judge --checkpoint requires a checkpoint")
			}
			checkpoint = strings.TrimSpace(args[i])
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", false, "", fmt.Errorf("judge does not accept flag: %s", args[i])
			}
			if path != "" {
				return "", false, "", errors.New("judge accepts at most one workspace path")
			}
			path = args[i]
		}
	}
	if checkpoint == "" {
		return "", false, "", errors.New("judge requires --checkpoint")
	}
	return checkpoint, jsonOutput, path, nil
}

func judgeWorkspace(root, path, checkpoint string) (JudgeResult, error) {
	workspace, err := ResolveWorkspace(root, path)
	if err != nil {
		if strings.Contains(err.Error(), "no .kkt workspace found") {
			return JudgeResult{
				SchemaVersion: 1,
				Verdict:       "allow",
				Checkpoint:    checkpoint,
				Mode:          "observe-inactive",
				Reason:        "no active .kkt workspace found",
			}, nil
		}
		return JudgeResult{}, err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return JudgeResult{}, err
	}
	result := JudgeResult{
		SchemaVersion: 1,
		Verdict:       "allow",
		Checkpoint:    checkpoint,
		Mode:          "enforce-active",
		Workspace:     workspace,
		WorkspaceType: state.WorkspaceType,
		ActiveLayer:   state.ActiveLayer,
		Reason:        "checkpoint allowed",
	}
	contract, contractErr := readGuardrails(workspace)
	guardrailIssues := validateGuardrails(workspace)
	hasGuardrails := len(guardrailIssues) == 0
	if state.WorkspaceType != "plan" && !hasGuardrails {
		result.Verdict = "warn"
		result.DriftType = "guardrail_contract"
		result.Reason = "guardrail contract is incomplete"
		result.Repair = append(result.Repair, "record a complete guardrails.json before relying on semantic drift enforcement")
		result.Evidence = append(result.Evidence, guardrailIssues...)
	}
	switch checkpoint {
	case "model-ready":
		if state.WorkspaceType == "run" || state.WorkspaceType == "loop" {
			if !hasGuardrails {
				result.Verdict = "block"
				result.Reason = "model-ready checkpoint requires valid guardrails"
			}
			if contractErr == nil {
				if issues := guardrailExecutionReadinessIssues(contract); len(issues) > 0 {
					result.Verdict = "block"
					result.DriftType = "guardrail_contract"
					result.Reason = "model-ready checkpoint requires constraint path bounds"
					result.Evidence = append(result.Evidence, issues...)
					result.Repair = append(result.Repair, "populate guardrails.json constraints and change_bounds.allowed_paths from the selected model")
				}
			}
			if issues := executionContractReadinessIssues(workspace, state); len(issues) > 0 {
				result.Verdict = "block"
				result.DriftType = "model_ready"
				result.Reason = "model-ready checkpoint requires a complete execution contract"
				result.Evidence = append(result.Evidence, issues...)
				result.Repair = append(result.Repair, "record plan.md before approval; loop workspaces must also record at least one task and acceptance criterion")
			}
		}
	case "pre-mutation":
		if (state.WorkspaceType == "run" || state.WorkspaceType == "loop") && state.ApprovalStatus != "approved" {
			result.Verdict = "block"
			result.DriftType = "approval"
			result.Reason = "mutation requires approval"
			result.Repair = append(result.Repair, "show the selected model and record approval with kkt approve")
		}
		if (state.WorkspaceType == "run" || state.WorkspaceType == "loop") && contractErr == nil {
			if issues := changedPathIssues(root, workspace, contract); len(issues) > 0 {
				result.Verdict = "block"
				result.DriftType = "path_scope"
				result.Reason = "changed paths violate guardrail bounds"
				result.Evidence = append(result.Evidence, issues...)
				result.Repair = append(result.Repair, "revert out-of-scope files or re-model and update guardrails.json before continuing")
			}
		}
	case "continuation":
		if state.WorkspaceType == "loop" {
			if state.Status == "blocked" {
				result.Verdict = "block"
				result.DriftType = "stop_condition"
				result.Reason = "blocked workflow cannot continue"
				result.Repair = append(result.Repair, "resolve the workflow blocker before continuing")
			}
			replay, replayErr := CheckReplay(workspace)
			if replayErr != nil {
				return JudgeResult{}, replayErr
			}
			if !replay.OK {
				result.Verdict = "block"
				result.DriftType = "replay"
				result.Reason = "loop replay drift detected"
				result.Evidence = append(result.Evidence, replay.Issues...)
				result.Repair = append(result.Repair, "inspect kkt.yaml and events.jsonl before continuing")
			}
			loop, loopErr := readLoopState(workspace)
			if loopErr == nil {
				for _, stop := range loop.StopConditions {
					if stop.Status == "active" {
						result.Verdict = "block"
						result.DriftType = "stop_condition"
						result.Reason = "active stop condition blocks continuation"
						result.Evidence = append(result.Evidence, stop.Text)
						result.Repair = append(result.Repair, "resolve the active stop condition before continuing")
						break
					}
				}
			}
		}
	case "finalize":
		validation, validationErr := ValidateWorkspace(workspace)
		if validationErr != nil {
			return JudgeResult{}, validationErr
		}
		if !validation.OK {
			result.Verdict = "block"
			result.DriftType = "validation"
			result.Reason = "workspace validation failed"
			result.Evidence = append(result.Evidence, validation.Issues...)
			result.Repair = append(result.Repair, "record required evidence and satisfy validation before completion")
		}
		if (state.WorkspaceType == "run" || state.WorkspaceType == "loop") && contractErr == nil {
			if issues := changedPathIssues(root, workspace, contract); len(issues) > 0 {
				result.Verdict = "block"
				result.DriftType = "path_scope"
				result.Reason = "changed paths violate guardrail bounds"
				result.Evidence = append(result.Evidence, issues...)
				result.Repair = append(result.Repair, "revert out-of-scope files or re-model and update guardrails.json before completion")
			}
		}
	case "pre-tool", "post-tool", "pre-compact", "post-compact":
		if (state.WorkspaceType == "run" || state.WorkspaceType == "loop") && contractErr == nil {
			if issues := changedPathIssues(root, workspace, contract); len(issues) > 0 {
				result.Verdict = "block"
				result.DriftType = "path_scope"
				result.Reason = "changed paths violate guardrail bounds"
				result.Evidence = append(result.Evidence, issues...)
				result.Repair = append(result.Repair, "revert out-of-scope files or re-model and update guardrails.json before continuing")
			}
		}
	default:
		result.Verdict = "warn"
		result.DriftType = "unknown_checkpoint"
		result.Reason = "unknown checkpoint; only workspace-level guardrails were checked"
		result.Repair = append(result.Repair, "use model-ready, pre-mutation, continuation, finalize, pre-tool, post-tool, pre-compact, or post-compact")
	}
	return result, nil
}

func executionContractReadinessIssues(workspace string, state State) []string {
	statuses, err := layerStatuses(workspace)
	if err != nil {
		return []string{err.Error()}
	}
	var issues []string
	issues = append(issues, workspaceModelContractIssues(workspace)...)
	for _, layer := range []string{"intent", "discovery", "modeling", "execution"} {
		if statuses[layer] != "complete" {
			status := statuses[layer]
			if status == "" {
				status = "missing"
			}
			issues = append(issues, fmt.Sprintf("%s layer is %s", layer, status))
		}
	}
	if state.WorkspaceType != "loop" {
		return issues
	}
	loop, err := readLoopState(workspace)
	if err != nil {
		return append(issues, err.Error())
	}
	if len(loop.Tasks) == 0 {
		issues = append(issues, "loop execution contract has no tasks")
	}
	if len(loop.AcceptanceCriteria) == 0 {
		issues = append(issues, "loop execution contract has no acceptance criteria")
	}
	return issues
}

func defaultGuardrailsJSON(request, profile, sourceWorkspace string) string {
	contract := GuardrailContract{
		SchemaVersion: 1,
		Source: GuardrailSource{
			WorkspaceType: profile,
			Workspace:     sourceWorkspace,
			Request:       request,
		},
		Constraints: []GuardrailConstraint{
			{
				ID:        "selected-model-scope",
				Kind:      "scope",
				Severity:  "block",
				Statement: "Implement only the selected KKT model and execution contract.",
			},
		},
		ChangeBounds: GuardrailBounds{
			AllowedPaths:                          []string{},
			BlockedPaths:                          []string{".git/**", ".env*", "dist/**"},
			RequireExplicitApprovalOutsideAllowed: true,
		},
		Workflow: GuardrailWorkflow{
			ExecutionMode:                  profile,
			RequiresApprovalBeforeMutation: profile == "run" || profile == "loop",
			RequiresValidationBeforeDone:   true,
		},
		Validation: GuardrailChecks{
			AcceptanceCriteria: []string{},
			RequiredCommands:   []string{},
			EvidenceRequired:   []string{"validation evidence recorded before done"},
		},
		DriftPolicy: GuardrailDriftPolicy{
			BlockOn: []string{
				"missing_approval",
				"empty_allowed_paths",
				"changed_blocked_path",
				"validation_failed",
			},
			WarnOn: []string{},
		},
	}
	payload, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return "{}\n"
	}
	return string(payload) + "\n"
}

func readGuardrails(workspace string) (GuardrailContract, error) {
	payload, err := os.ReadFile(filepath.Join(workspace, "guardrails.json"))
	if err != nil {
		return GuardrailContract{}, err
	}
	var contract GuardrailContract
	if err := json.Unmarshal(payload, &contract); err != nil {
		return GuardrailContract{}, err
	}
	return contract, nil
}

func validateGuardrails(workspace string) []string {
	contract, err := readGuardrails(workspace)
	if err != nil {
		return []string{"guardrails.json could not be read: " + err.Error()}
	}
	return validateCompleteGuardrailContract(contract)
}

func validateCompleteGuardrailContract(contract GuardrailContract) []string {
	var issues []string
	if contract.SchemaVersion != 1 {
		issues = append(issues, "guardrails.json schema_version must be 1")
	}
	if strings.TrimSpace(contract.Source.WorkspaceType) == "" {
		issues = append(issues, "guardrails.json source.workspace_type is required")
	}
	if strings.TrimSpace(contract.Source.Workspace) == "" {
		issues = append(issues, "guardrails.json source.workspace is required")
	}
	if strings.TrimSpace(contract.Source.Request) == "" {
		issues = append(issues, "guardrails.json source.request is required")
	}
	if strings.TrimSpace(contract.Workflow.ExecutionMode) == "" {
		issues = append(issues, "guardrails.json workflow.execution_mode is required")
	}
	if contract.Workflow.RequiresValidationBeforeDone && len(contract.Validation.EvidenceRequired) == 0 && len(contract.Validation.RequiredCommands) == 0 {
		issues = append(issues, "guardrails.json requires validation but lists no evidence or commands")
	}
	if len(contract.DriftPolicy.BlockOn) == 0 && strings.TrimSpace(contract.DriftPolicy.Legacy) == "" {
		issues = append(issues, "guardrails.json drift_policy.block_on is required")
	}
	return append(issues, guardrailExecutionReadinessIssues(contract)...)
}

func guardrailExecutionReadinessIssues(contract GuardrailContract) []string {
	var issues []string
	if len(contract.Constraints) == 0 {
		issues = append(issues, "guardrails.json constraints must include at least one modeled constraint")
	}
	if len(contract.allowedPaths()) == 0 {
		issues = append(issues, "guardrails.json change_bounds.allowed_paths must include the selected model's expected files or surfaces")
	}
	return issues
}

func changedPathIssues(root, workspace string, contract GuardrailContract) []string {
	projectRootDir, err := projectRootForWorkspace(root, workspace)
	if err != nil {
		return []string{"could not resolve project root for path guardrails: " + err.Error()}
	}
	changed, err := changedGitPaths(projectRootDir)
	if err != nil {
		return []string{"could not inspect changed paths: " + err.Error()}
	}
	if len(changed) == 0 {
		return nil
	}
	allowed := contract.allowedPaths()
	blocked := contract.blockedPaths()
	baseline, hasBaseline, baselineErr := readApprovalBaseline(workspace)
	var issues []string
	if baselineErr != nil {
		issues = append(issues, "approval baseline could not be read: "+baselineErr.Error())
	}
	for _, path := range changed {
		if isKKTPath(path) {
			continue
		}
		if hasBaseline && unchangedFromApprovalBaseline(projectRootDir, baseline, path) {
			continue
		}
		if matchesAnyPathPattern(path, blocked) {
			issues = append(issues, "changed blocked path: "+path)
			continue
		}
		// Paths outside the modeled allowed bounds are treated as unrelated branch
		// work. Guardrails enforce the implementation scope and explicit blocks;
		// they should not require an otherwise clean working tree.
		if len(allowed) > 0 && !matchesAnyPathPattern(path, allowed) {
			continue
		}
	}
	return issues
}

func writeApprovalBaseline(workspace string) error {
	projectRootDir, err := projectRootForWorkspace(".", workspace)
	if err != nil {
		return err
	}
	changed, err := changedGitPaths(projectRootDir)
	if err != nil {
		return err
	}
	baseline := ApprovalBaseline{
		SchemaVersion: 1,
		RecordedAt:    time.Now().UTC().Format(time.RFC3339),
		Paths:         map[string]string{},
	}
	for _, path := range changed {
		if isKKTPath(path) {
			continue
		}
		fingerprint, err := pathFingerprint(projectRootDir, path)
		if err != nil {
			return err
		}
		baseline.Paths[path] = fingerprint
	}
	payload, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(workspace, "approval-baseline.json"), append(payload, '\n'), 0o644)
}

func readApprovalBaseline(workspace string) (ApprovalBaseline, bool, error) {
	payload, err := os.ReadFile(filepath.Join(workspace, "approval-baseline.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return ApprovalBaseline{}, false, nil
		}
		return ApprovalBaseline{}, false, err
	}
	var baseline ApprovalBaseline
	if err := json.Unmarshal(payload, &baseline); err != nil {
		return ApprovalBaseline{}, true, err
	}
	if baseline.Paths == nil {
		baseline.Paths = map[string]string{}
	}
	return baseline, true, nil
}

func unchangedFromApprovalBaseline(projectRootDir string, baseline ApprovalBaseline, path string) bool {
	approvedFingerprint, ok := baseline.Paths[path]
	if !ok {
		return false
	}
	currentFingerprint, err := pathFingerprint(projectRootDir, path)
	return err == nil && currentFingerprint == approvedFingerprint
}

func isKKTPath(path string) bool {
	path = normalizeRepoPath(path)
	return path == ".kkt" || strings.HasPrefix(path, ".kkt/")
}

func pathFingerprint(projectRootDir, path string) (string, error) {
	fullPath := filepath.Join(projectRootDir, filepath.FromSlash(path))
	payload, err := os.ReadFile(fullPath)
	switch {
	case err == nil:
		sum := sha256.Sum256(payload)
		return hex.EncodeToString(sum[:]), nil
	case os.IsNotExist(err):
		return "<deleted>", nil
	default:
		return "", err
	}
}

func (contract GuardrailContract) allowedPaths() []string {
	paths := append([]string{}, contract.ChangeBounds.AllowedPaths...)
	paths = append(paths, contract.ChangeBounds.AllowedPathsOrSurfaces...)
	for _, constraint := range contract.Constraints {
		paths = append(paths, constraint.AllowedPaths...)
	}
	return uniqueNonEmpty(paths)
}

func (contract GuardrailContract) blockedPaths() []string {
	paths := append([]string{}, contract.ChangeBounds.BlockedPaths...)
	paths = append(paths, contract.ChangeBounds.BlockedPathsOrSurfaces...)
	for _, constraint := range contract.Constraints {
		paths = append(paths, constraint.BlockedPaths...)
	}
	return uniqueNonEmpty(paths)
}

func projectRootForWorkspace(root, workspace string) (string, error) {
	clean := filepath.Clean(workspace)
	for {
		if filepath.Base(clean) == ".kkt" {
			return filepath.Dir(clean), nil
		}
		parent := filepath.Dir(clean)
		if parent == clean {
			break
		}
		clean = parent
	}
	return projectRoot(root)
}

func changedGitPaths(projectRootDir string) ([]string, error) {
	if err := runGit(projectRootDir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, nil
	}
	commands := [][]string{
		{"diff", "--name-only"},
		{"diff", "--name-only", "--cached"},
		{"ls-files", "--others", "--exclude-standard"},
	}
	seen := map[string]bool{}
	var paths []string
	for _, args := range commands {
		output, err := gitOutput(projectRootDir, args...)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(output, "\n") {
			path := normalizeRepoPath(line)
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func runGit(projectRootDir string, args ...string) error {
	_, err := gitOutput(projectRootDir, args...)
	return err
}

func gitOutput(projectRootDir string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", projectRootDir}, args...)...)
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func matchesAnyPathPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPathPattern(path, pattern) {
			return true
		}
	}
	return false
}

func matchPathPattern(path, pattern string) bool {
	path = normalizeRepoPath(path)
	pattern = normalizeRepoPath(pattern)
	if path == "" || pattern == "" {
		return false
	}
	if !strings.ContainsAny(pattern, "*?[") {
		return path == pattern || strings.HasPrefix(path, strings.TrimSuffix(pattern, "/")+"/")
	}
	expression := globPatternRegexp(pattern)
	matched, err := regexp.MatchString(expression, path)
	return err == nil && matched
}

func globPatternRegexp(pattern string) string {
	var builder strings.Builder
	builder.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		char := pattern[i]
		if char == '*' {
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				builder.WriteString(".*")
				i++
			} else {
				builder.WriteString(`[^/]*`)
			}
			continue
		}
		if char == '?' {
			builder.WriteString(`[^/]`)
			continue
		}
		builder.WriteString(regexp.QuoteMeta(string(char)))
	}
	builder.WriteString("$")
	return builder.String()
}

func normalizeRepoPath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	path = strings.TrimPrefix(path, "./")
	return strings.Trim(path, "/")
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	var unique []string
	for _, value := range values {
		value = normalizeRepoPath(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func createExecutionFromModel(root, modelWorkspace, profile string) (Workspace, error) {
	if profile != "run" && profile != "loop" {
		return Workspace{}, fmt.Errorf("from-model does not support profile %q", profile)
	}
	if modelWorkspace == "" {
		resolved, err := ResolveWorkspace(root, "")
		if err != nil {
			return Workspace{}, err
		}
		modelWorkspace = resolved
	}
	modelState, err := ReadState(modelWorkspace)
	if err != nil {
		return Workspace{}, err
	}
	if modelState.WorkspaceType != "model" {
		return Workspace{}, fmt.Errorf("from-model requires a model workspace, got %q", modelState.WorkspaceType)
	}
	if modelState.Status != "complete" {
		return Workspace{}, fmt.Errorf("from-model requires a complete model workspace, got status %q", modelState.Status)
	}
	projectRootDir, err := projectRoot(root)
	if err != nil {
		return Workspace{}, err
	}
	request, err := stateRequest(modelWorkspace)
	if err != nil {
		return Workspace{}, err
	}
	if request == "" {
		request = "Execute selected KKT model"
	}
	now := time.Now().UTC()
	slug := fmt.Sprintf("%s-%s-%s", now.Format("20060102-150405"), profile, filepath.Base(modelWorkspace))
	base := filepath.Join(projectRootDir, ".kkt")
	executionWorkspace := filepath.Join(base, profile, slug)
	if err := os.MkdirAll(executionWorkspace, 0o755); err != nil {
		return Workspace{}, err
	}
	sourceWorkspace := normalizeRepoPath(filepath.Join(".kkt", currentPointer(profile, slug)))
	files := workspaceFiles(request, profile, now, sourceWorkspace)
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(executionWorkspace, name), []byte(content), 0o644); err != nil {
			return Workspace{}, err
		}
	}
	for _, name := range []string{"intent.md", "discovery.md", "model.md", "guardrails.json"} {
		if err := copyWorkspaceFile(modelWorkspace, executionWorkspace, name); err != nil {
			if errors.Is(err, os.ErrNotExist) && name == "guardrails.json" {
				if writeErr := os.WriteFile(filepath.Join(executionWorkspace, name), []byte(defaultGuardrailsJSON(request, profile, workspaceSourcePath(projectRootDir, modelWorkspace))), 0o644); writeErr != nil {
					return Workspace{}, writeErr
				}
				continue
			}
			return Workspace{}, err
		}
	}
	if err := updateExecutionSource(executionWorkspace, modelWorkspace, profile); err != nil {
		return Workspace{}, err
	}
	if err := markImportedModelLayers(executionWorkspace, modelWorkspace); err != nil {
		return Workspace{}, err
	}
	if err := os.WriteFile(filepath.Join(base, "current"), []byte(currentPointer(profile, slug)+"\n"), 0o644); err != nil {
		return Workspace{}, err
	}
	return Workspace{Path: executionWorkspace, Profile: profile}, nil
}

func copyWorkspaceFile(sourceWorkspace, targetWorkspace, name string) error {
	payload, err := os.ReadFile(filepath.Join(sourceWorkspace, name))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(targetWorkspace, name), payload, 0o644)
}

func updateExecutionSource(executionWorkspace, modelWorkspace, profile string) error {
	if err := updateTopLevelState(executionWorkspace, "active_layer", "execution"); err != nil {
		return err
	}
	// Importing a completed model preserves its decision; mutation still requires
	// the user-facing approval gate before edits in either execution profile.
	if err := updateApproval(executionWorkspace, "pending", ""); err != nil {
		return err
	}
	contract, err := readGuardrails(executionWorkspace)
	if err != nil {
		return nil
	}
	contract.Source.WorkspaceType = "model"
	projectRootDir, rootErr := projectRootForWorkspace(".", executionWorkspace)
	if rootErr == nil {
		contract.Source.Workspace = workspaceSourcePath(projectRootDir, modelWorkspace)
	} else {
		contract.Source.Workspace = modelWorkspace
	}
	contract.Workflow.ExecutionMode = profile
	payload, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(executionWorkspace, "guardrails.json"), append(payload, '\n'), 0o644)
}

func markImportedModelLayers(executionWorkspace, modelWorkspace string) error {
	if sourceState, err := ReadState(modelWorkspace); err == nil && sourceState.ContractVersion != "2" {
		if err := updateTopLevelState(executionWorkspace, "contract_version", "legacy"); err != nil {
			return err
		}
	}
	source := modelWorkspace
	if projectRootDir, err := projectRootForWorkspace(".", executionWorkspace); err == nil {
		source = workspaceSourcePath(projectRootDir, modelWorkspace)
	}
	for _, layer := range []string{"intent", "discovery", "modeling"} {
		summary := "Imported from " + source
		if err := updateLayerState(executionWorkspace, layer, "complete", "imported", summary); err != nil {
			return err
		}
		if err := appendMethodInvocation(executionWorkspace, layer, "imported", summary); err != nil {
			return err
		}
	}
	return nil
}

func workspaceSourcePath(projectRootDir, workspace string) string {
	absoluteRoot, rootErr := filepath.Abs(projectRootDir)
	absoluteWorkspace, workspaceErr := filepath.Abs(workspace)
	if rootErr == nil && workspaceErr == nil {
		if rel, err := filepath.Rel(absoluteRoot, absoluteWorkspace); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return normalizeRepoPath(rel)
		}
	}
	return normalizeRepoPath(workspace)
}

func stateRequest(workspace string) (string, error) {
	payload, err := os.ReadFile(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(payload), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "request:") {
			return unquoteYAML(strings.TrimSpace(strings.TrimPrefix(trimmed, "request:"))), nil
		}
	}
	return "", nil
}

package workflow

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var Version = "dev"

const usageText = `KKT Workflow CLI

Usage:
  kkt --version
  kkt start plan|model|run|loop <request>
  kkt status [path]
  kkt next [--json] [path]
  kkt show [artifact]
  kkt intent|discovery|model [--method method] [content]
  kkt plan|progress [content]
  kkt evidence [--for criterion] [--command command] [content]
  kkt notes [content]
  kkt guardrails show|set|validate [content]
  kkt judge --checkpoint checkpoint [--json] [path]
  kkt run from-model [model-workspace]
  kkt approve [scope]
  kkt criteria [add|satisfy|block] [criterion]
  kkt task [add|start|done|skip|block] [task]
  kkt block [reason]
  kkt validate [path]
  kkt done [summary]
  kkt resume [path]
  kkt replay --check [path]
  kkt uninstall [codex|claude|opencode|pi|all]

KKT skills own the workflow. This CLI handles deterministic .kkt state
scaffolding, workflow state, artifacts, evidence, validation, and cleanup.
`

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprint(stdout, usageText)
		return nil
	}
	if args[0] == "--version" {
		fmt.Fprintf(stdout, "kkt %s\n", Version)
		return nil
	}

	switch args[0] {
	case "start":
		return runStart(args[1:], stdout)
	case "status":
		return runStatus(args[1:], stdout)
	case "next":
		return runNext(args[1:], stdout)
	case "show":
		return runShow(args[1:], stdout)
	case "intent", "discovery", "model", "plan", "progress", "evidence", "notes":
		return runArtifact(args[0], args[1:], stdout)
	case "guardrails":
		return runGuardrails(args[1:], stdout)
	case "judge":
		return runJudge(args[1:], stdout)
	case "run":
		return runRun(args[1:], stdout)
	case "approve":
		return runApprove(args[1:], stdout)
	case "criteria":
		return runCriteria(args[1:], stdout)
	case "task":
		return runTask(args[1:], stdout)
	case "block":
		return runBlock(args[1:], stdout)
	case "validate":
		return runValidate(args[1:], stdout)
	case "done":
		return runDone(args[1:], stdout)
	case "resume":
		return runResume(args[1:], stdout)
	case "replay":
		return runReplay(args[1:], stdout)
	case "uninstall":
		return runUninstall(args[1:], stdout)
	default:
		return fmt.Errorf("unsupported command %q\n\n%s", args[0], usageText)
	}
}

func runUninstall(args []string, stdout io.Writer) error {
	if err := rejectFlags("uninstall", args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.New("uninstall accepts at most one agent: codex, claude, opencode, pi, or all")
	}
	agent := strings.TrimSpace(firstArg(args))
	if agent == "" {
		agent = "all"
	}

	plans, err := UninstallPlans(agent)
	if err != nil {
		return err
	}
	for _, plan := range plans {
		fmt.Fprintf(stdout, "target: %s\n", plan.Agent)
		fmt.Fprintf(stdout, "file: %s\n", plan.Path)
		changed, err := RemoveInstruction(plan.Path)
		if err != nil {
			return err
		}
		if changed {
			fmt.Fprintln(stdout, "removed")
		} else {
			fmt.Fprintln(stdout, "already current")
		}
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "binary: %s\n", executable)
	if err := os.Remove(executable); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(stdout, "already current")
			return nil
		}
		return err
	}
	fmt.Fprintln(stdout, "removed")
	return nil
}

func runStart(args []string, stdout io.Writer) error {
	if err := rejectFlags("start", args); err != nil {
		return err
	}
	if len(args) < 2 {
		return errors.New("start requires a profile and request: plan, model, run, or loop")
	}
	selectedProfile := strings.TrimSpace(args[0])
	request := strings.TrimSpace(strings.Join(args[1:], " "))
	if request == "" {
		return errors.New("start requires a request")
	}

	workspace, err := StartWorkflow(".", request, selectedProfile)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "created: %s\n", workspace.Path)
	fmt.Fprintf(stdout, "profile: %s\n", workspace.Profile)
	fmt.Fprintf(stdout, "next: %s\n", startInstruction(workspace.Profile))
	return nil
}

func runStatus(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", firstArg(args))
	if err != nil {
		return err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "workspace: %s\n", workspace)
	fmt.Fprintf(stdout, "status: %s\n", state.Status)
	fmt.Fprintf(stdout, "active_layer: %s\n", state.ActiveLayer)
	fmt.Fprintf(stdout, "profile: %s\n", state.Profile)
	fmt.Fprintf(stdout, "approval: %s\n", state.ApprovalStatus)
	if loop, loopErr := readLoopState(workspace); loopErr == nil && loop.CurrentTask != "" {
		fmt.Fprintf(stdout, "current_task: %s\n", loop.CurrentTask)
	}
	return nil
}

func runNext(args []string, stdout io.Writer) error {
	jsonOutput, path, err := parseJSONPathArgs("next", args)
	if err != nil {
		return err
	}
	workspace, err := ResolveWorkspace(".", path)
	if err != nil {
		return err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	action := nextActionForWorkspace(workspace, state)
	if jsonOutput {
		return writeJSON(stdout, action)
	}
	fmt.Fprintln(stdout, action.Instruction)
	return nil
}

func runValidate(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", firstArg(args))
	if err != nil {
		return err
	}
	result, err := ValidateWorkspace(workspace)
	if err != nil {
		return err
	}
	if result.OK {
		if err := appendValidationEvent(workspace, result); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "valid: %s\n", workspace)
		return nil
	}
	if err := appendValidationEvent(workspace, result); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "invalid: %s\n", workspace)
	for _, issue := range result.Issues {
		fmt.Fprintf(stdout, "- %s\n", issue)
	}
	return errors.New("workspace validation failed")
}

func rejectFlags(command string, args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("%s does not accept flags: %s", command, arg)
		}
	}
	return nil
}

func parseJSONPathArgs(command string, args []string) (bool, string, error) {
	jsonOutput := false
	path := ""
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return false, "", fmt.Errorf("%s does not accept flag: %s", command, arg)
		}
		if path != "" {
			return false, "", fmt.Errorf("%s accepts at most one path", command)
		}
		path = arg
	}
	return jsonOutput, path, nil
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func startInstruction(profile string) string {
	switch profile {
	case "plan":
		return "inspect relevant code/docs, record objective_function, files_to_modify, constraint_functions, decision_variables, and validation_proof with kkt model, then request approval before edits"
	case "model":
		return "record adaptive intent with kkt intent --method <method>, then inspect relevant code/docs and record discovery"
	case "run":
		return "record or import the selected model, run kkt judge --checkpoint model-ready, then request approval before edits"
	default:
		return "record adaptive intent with kkt intent --method <method>, then inspect relevant code/docs and record discovery/model/plan"
	}
}

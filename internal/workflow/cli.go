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
  kkt start plan|model|loop <request>
  kkt status [path]
  kkt next [path]
  kkt validate [path]
  kkt uninstall [codex|claude|opencode|pi|all]

KKT skills own the workflow. This CLI handles deterministic .kkt state
scaffolding, status, next-step hints, validation, and legacy cleanup.
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
	case "validate":
		return runValidate(args[1:], stdout)
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
		return errors.New("start requires a profile and request: plan, model, or loop")
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
	return nil
}

func runNext(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", firstArg(args))
	if err != nil {
		return err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, NextInstruction(state))
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
		fmt.Fprintf(stdout, "valid: %s\n", workspace)
		return nil
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

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func startInstruction(profile string) string {
	switch profile {
	case "plan":
		return "inspect relevant code/docs, keep compact state in .kkt/kkt.yaml, then request approval before edits"
	case "model":
		return "inspect relevant code/docs, complete discovery.md, then update model.md with the selected model"
	default:
		return "inspect relevant code/docs, complete discovery.md, then update model.md before requesting approval"
	}
}

package workflow

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const usageText = `KKT Workflow CLI

Usage:
  kkt classify [--json] <request>
  kkt start [--profile daily|loop|model] <request>
  kkt status [path]
  kkt next [path]
  kkt validate [path]
  kkt init [--dry-run] [--command path] codex|claude|opencode|pi|all

KKT coordinates one existing coding agent session. It does not own the TUI,
spawn subagents, route models, or run a detached harness.
`

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(stdout, usageText)
		return nil
	}

	switch args[0] {
	case "classify":
		return runClassify(args[1:], stdout)
	case "start":
		return runStart(args[1:], stdout)
	case "status":
		return runStatus(args[1:], stdout)
	case "next":
		return runNext(args[1:], stdout)
	case "validate":
		return runValidate(args[1:], stdout)
	case "init":
		return runInit(args[1:], stdout)
	default:
		return fmt.Errorf("unsupported command %q\n\n%s", args[0], usageText)
	}
}

func runClassify(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("classify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	request := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if request == "" {
		return errors.New("classify requires a request")
	}

	result := ClassifyWithCommand(request, os.Args[0])
	if *jsonOut {
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(encoded))
		return nil
	}

	fmt.Fprintf(stdout, "decision: %s\n", result.Decision)
	fmt.Fprintf(stdout, "profile: %s\n", result.Profile)
	fmt.Fprintf(stdout, "reason: %s\n", result.Reason)
	if result.NextCommand != "" {
		fmt.Fprintf(stdout, "next: %s\n", result.NextCommand)
	}
	return nil
}

func runStart(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profile := fs.String("profile", "", "daily, loop, or model")
	if err := fs.Parse(args); err != nil {
		return err
	}
	request := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if request == "" {
		return errors.New("start requires a request")
	}

	selectedProfile := strings.TrimSpace(*profile)
	if selectedProfile == "" {
		selectedProfile = Classify(request).Profile
	}
	workspace, err := StartWorkflow(".", request, selectedProfile)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "created: %s\n", workspace.Path)
	fmt.Fprintf(stdout, "profile: %s\n", workspace.Profile)
	fmt.Fprintln(stdout, "next: inspect relevant code/docs, complete discovery, then update model.md before requesting approval")
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

func runInit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dryRun := fs.Bool("dry-run", false, "print integration instructions without writing")
	command := fs.String("command", "kkt", "kkt command agents should run")
	flagArgs, positionalArgs, err := splitInitArgs(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(positionalArgs) > 1 {
		return errors.New("init accepts exactly one agent: codex, claude, opencode, pi, or all")
	}
	agent := strings.TrimSpace(firstArg(positionalArgs))
	if agent == "" {
		return errors.New("init requires an agent: codex, claude, opencode, pi, or all")
	}

	plans, err := InitPlans(agent, *command)
	if err != nil {
		return err
	}
	for _, plan := range plans {
		fmt.Fprintf(stdout, "target: %s\n", plan.Agent)
		fmt.Fprintf(stdout, "file: %s\n", plan.Path)
		if *dryRun {
			fmt.Fprintln(stdout, "---")
			fmt.Fprint(stdout, plan.Content)
			if !strings.HasSuffix(plan.Content, "\n") {
				fmt.Fprintln(stdout)
			}
			continue
		}
		changed, err := WriteInstruction(plan.Path, plan.Content)
		if err != nil {
			return err
		}
		if changed {
			fmt.Fprintln(stdout, "updated")
		} else {
			fmt.Fprintln(stdout, "already current")
		}
	}
	return nil
}

func splitInitArgs(args []string) ([]string, []string, error) {
	flags := []string{}
	positionals := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dry-run":
			flags = append(flags, arg)
		case "--command":
			if i+1 >= len(args) {
				return nil, nil, errors.New("--command requires a value")
			}
			flags = append(flags, arg, args[i+1])
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				flags = append(flags, arg)
			} else {
				positionals = append(positionals, arg)
			}
		}
	}
	return flags, positionals, nil
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

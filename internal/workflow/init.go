package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	instructionStart = "<!-- kkt-workflow:start -->"
	instructionEnd   = "<!-- kkt-workflow:end -->"
)

type InitPlan struct {
	Agent   string
	Path    string
	Content string
}

func InitPlans(agent, command string) ([]InitPlan, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return InitPlansWithHome(agent, home, command)
}

func InitPlansWithHome(agent, home, command string) ([]InitPlan, error) {
	targets, err := expandAgent(agent)
	if err != nil {
		return nil, err
	}
	grouped := map[string][]string{}
	for _, target := range targets {
		path := filepath.Join(home, instructionPath(target))
		grouped[path] = append(grouped[path], target)
	}

	plans := make([]InitPlan, 0, len(grouped))
	for _, target := range targets {
		path := filepath.Join(home, instructionPath(target))
		agents, ok := grouped[path]
		if !ok {
			continue
		}
		delete(grouped, path)
		agentLabel := strings.Join(agents, ", ")
		plans = append(plans, InitPlan{
			Agent:   agentLabel,
			Path:    path,
			Content: instructionContent(agentLabel, command),
		})
	}
	return plans, nil
}

func expandAgent(agent string) ([]string, error) {
	switch agent {
	case "all":
		return []string{"codex", "claude", "opencode", "pi"}, nil
	case "codex", "claude", "opencode", "pi":
		return []string{agent}, nil
	default:
		return nil, fmt.Errorf("unsupported agent %q", agent)
	}
}

func instructionPath(agent string) string {
	switch agent {
	case "claude":
		return filepath.Join(".claude", "CLAUDE.md")
	case "opencode":
		return filepath.Join(".agents", "AGENTS.md")
	case "pi":
		return filepath.Join(".agents", "AGENTS.md")
	default:
		return filepath.Join(".agents", "AGENTS.md")
	}
}

func instructionContent(agent, command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		command = "kkt"
	}
	return fmt.Sprintf(`%s
# KKT Workflow

Use KKT as an advisory workflow tool for non-trivial coding work.

Before implementation-heavy requests, run:

`+"```bash"+`
%s classify "<user request>"
`+"```"+`

If the decision is `+"`invoke`"+`, run the suggested start command, inspect the generated `.kkt/model/<run>/`, `.kkt/loop/<run>/`, or compact `.kkt/kkt.yaml` workspace, and follow its state contract.

During KKT-managed work:

- keep using this coding-agent session as the active coding agent;
- do not spawn KKT subagents or assume detached harness behavior;
- complete discovery before modeling;
- show the selected model and get explicit approval before file edits;
- for loop work, update `.kkt/loop/<run>/progress.md` and `.kkt/loop/<run>/evidence.md` as work proceeds;
- run `+"`%s validate`"+` before the final response when a KKT workspace is active;
- if KKT fails, continue normally and report the failure.

Target integration: %s.
%s
`, instructionStart, command, command, agent, instructionEnd)
}

func WriteInstruction(path, content string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	existingBytes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	existing := string(existingBytes)
	if strings.Contains(existing, instructionStart) && strings.Contains(existing, instructionEnd) {
		start := strings.Index(existing, instructionStart)
		end := strings.Index(existing, instructionEnd)
		end += len(instructionEnd)
		next := existing[:start] + strings.TrimSpace(content) + existing[end:]
		if next == existing {
			return false, nil
		}
		return true, os.WriteFile(path, []byte(next), 0o644)
	}

	next := strings.TrimRight(existing, "\n")
	if next != "" {
		next += "\n\n"
	}
	next += strings.TrimSpace(content) + "\n"
	if next == existing {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(next), 0o644)
}

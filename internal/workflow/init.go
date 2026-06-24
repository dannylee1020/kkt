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
	Agent  string
	Path   string
	Remove bool
}

func UninstallPlans(agent string) ([]InitPlan, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return UninstallPlansWithHome(agent, home)
}

func UninstallPlansWithHome(agent, home string) ([]InitPlan, error) {
	targets, err := expandAgent(agent)
	if err != nil {
		return nil, err
	}

	plans := []InitPlan{}
	for _, target := range targets {
		plans = append(plans, uninstallPlansForTarget(target, home)...)
	}
	if needsLegacyCleanup(targets) {
		plans = append(plans, legacyCleanupPlans(home)...)
	}
	return dedupePlans(plans), nil
}

func uninstallPlansForTarget(agent, home string) []InitPlan {
	plans := []InitPlan{
		{
			Agent:  agent,
			Path:   filepath.Join(home, instructionPath(agent)),
			Remove: true,
		},
	}
	if usesInstructionReference(agent) {
		plans = append(plans, InitPlan{
			Agent:  agent,
			Path:   filepath.Join(home, kktInstructionPath(agent)),
			Remove: true,
		})
	}
	return plans
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
	case "codex":
		return filepath.Join(".codex", "AGENTS.md")
	case "claude":
		return filepath.Join(".claude", "CLAUDE.md")
	case "opencode":
		return filepath.Join(".config", "opencode", "AGENTS.md")
	case "pi":
		return filepath.Join(".pi", "agent", "AGENTS.md")
	}
	return ""
}

func kktInstructionPath(agent string) string {
	switch agent {
	case "codex":
		return filepath.Join(".codex", "KKT.md")
	case "claude":
		return filepath.Join(".claude", "KKT.md")
	}
	return ""
}

func usesInstructionReference(agent string) bool {
	return agent == "codex" || agent == "claude"
}

func needsLegacyCleanup(agents []string) bool {
	for _, agent := range agents {
		switch agent {
		case "codex", "opencode", "pi":
			return true
		}
	}
	return false
}

func legacyCleanupPlans(home string) []InitPlan {
	return []InitPlan{
		{
			Agent:  "legacy",
			Path:   filepath.Join(home, ".agents", "AGENTS.md"),
			Remove: true,
		},
		{
			Agent:  "legacy",
			Path:   filepath.Join(home, ".agents", "KKT.md"),
			Remove: true,
		},
	}
}

func dedupePlans(plans []InitPlan) []InitPlan {
	seen := map[string]bool{}
	deduped := []InitPlan{}
	for _, plan := range plans {
		key := fmt.Sprintf("%t:%s", plan.Remove, plan.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, plan)
	}
	return deduped
}

func RemoveInstruction(path string) (bool, error) {
	existingBytes, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	existing := string(existingBytes)
	start := strings.Index(existing, instructionStart)
	end := strings.Index(existing, instructionEnd)
	if start < 0 || end < 0 || end < start {
		return false, nil
	}
	end += len(instructionEnd)
	next := strings.Trim(existing[:start]+existing[end:], "\n")
	if next != "" {
		next += "\n"
	}
	if next == existing {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(next), 0o644)
}

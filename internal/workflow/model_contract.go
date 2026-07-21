package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type optimizationContractField struct {
	Name     string
	Headings []string
}

var optimizationContractFields = []optimizationContractField{
	{Name: "objective_function", Headings: []string{"objective function"}},
	{Name: "decision_variables", Headings: []string{"decision variables", "decision variables and affected surfaces"}},
	{Name: "affected_surfaces", Headings: []string{"decision variables and affected surfaces", "affected surfaces", "files to modify"}},
	{Name: "constraint_functions", Headings: []string{"constraint functions"}},
	{Name: "candidate_feasibility", Headings: []string{"candidate feasibility"}},
	{Name: "selected_optimum", Headings: []string{"selected optimum", "selected plan"}},
	{Name: "binding_constraints", Headings: []string{"binding constraints"}},
	{Name: "validation_proof", Headings: []string{"validation plan and certificate", "validation plan and proof", "validation proof"}},
}

type markdownSection struct {
	Heading string
	Body    string
}

// optimizationContractIssues validates the minimum constrained-optimization
// kernel shared by compact and deep model representations. It intentionally
// checks structure and non-empty content, not the agent's semantic judgment.
func workspaceModelContractIssues(workspace string) []string {
	state, err := ReadState(workspace)
	if err != nil {
		return []string{err.Error()}
	}
	if state.ContractVersion != "2" {
		return nil
	}
	statuses, err := layerStatuses(workspace)
	if err != nil {
		return []string{err.Error()}
	}
	if statuses["modeling"] != "complete" {
		return nil
	}
	content, err := os.ReadFile(filepath.Join(workspace, "model.md"))
	if err != nil {
		return []string{"model.md could not be read: " + err.Error()}
	}
	return optimizationContractIssues(string(content))
}

func optimizationContractIssues(content string) []string {
	sections := markdownSections(content)
	var issues []string
	for _, field := range optimizationContractFields {
		section, ok := findMarkdownSection(sections, field.Headings)
		if !ok {
			issues = append(issues, fmt.Sprintf("missing constrained optimization field %s", field.Name))
			continue
		}
		if isEmptyContractSection(section.Body) {
			issues = append(issues, fmt.Sprintf("constrained optimization field %s is empty", field.Name))
		}
		if field.Name == "constraint_functions" {
			lower := strings.ToLower(section.Body)
			if !strings.Contains(lower, "hard") {
				issues = append(issues, "constraint_functions must state hard constraints")
			}
			if !strings.Contains(lower, "soft") {
				issues = append(issues, "constraint_functions must state soft constraints")
			}
		}
		if field.Name == "candidate_feasibility" {
			lower := strings.ToLower(section.Body)
			if !strings.Contains(lower, "feasible") && !strings.Contains(lower, "reject") && !strings.Contains(lower, "viable") {
				issues = append(issues, "candidate_feasibility must identify feasible or rejected candidates")
			}
		}
	}
	return issues
}

func markdownSections(content string) []markdownSection {
	lines := strings.Split(content, "\n")
	sections := []markdownSection{}
	current := -1
	for _, line := range lines {
		heading, ok := markdownHeading(line)
		if ok {
			sections = append(sections, markdownSection{Heading: heading})
			current = len(sections) - 1
			continue
		}
		if current >= 0 {
			if sections[current].Body != "" {
				sections[current].Body += "\n"
			}
			sections[current].Body += line
		}
	}
	return sections
}

func markdownHeading(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	text := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
	text = strings.TrimSpace(strings.TrimRight(text, "#"))
	if text == "" {
		return "", false
	}
	return strings.ToLower(strings.Join(strings.Fields(text), " ")), true
}

func findMarkdownSection(sections []markdownSection, headings []string) (markdownSection, bool) {
	wanted := map[string]bool{}
	for _, heading := range headings {
		wanted[strings.ToLower(strings.Join(strings.Fields(heading), " "))] = true
	}
	for i := len(sections) - 1; i >= 0; i-- {
		if wanted[sections[i].Heading] {
			return sections[i], true
		}
	}
	return markdownSection{}, false
}

func isEmptyContractSection(body string) bool {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return true
	}
	for _, marker := range []string{"status: pending", "todo", "tbd", "to be determined", "fill this in"} {
		if strings.EqualFold(trimmed, marker) {
			return true
		}
	}
	return false
}

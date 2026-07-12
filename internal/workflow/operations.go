package workflow

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LoopTask struct {
	ID     string
	Title  string
	Status string
}

type LoopCriterion struct {
	ID     string
	Text   string
	Status string
}

type LoopEvidence struct {
	ID       string
	Summary  string
	Status   string
	Criteria []string
	Command  string
}

type LoopStopCondition struct {
	ID     string
	Text   string
	Status string
}

type LoopState struct {
	CurrentTask        string
	Tasks              []LoopTask
	AcceptanceCriteria []LoopCriterion
	Evidence           []LoopEvidence
	StopConditions     []LoopStopCondition
}

type NextAction struct {
	SchemaVersion    int      `json:"schema_version"`
	Action           string   `json:"action"`
	Reason           string   `json:"reason"`
	TaskID           string   `json:"task_id,omitempty"`
	CriterionID      string   `json:"criterion_id,omitempty"`
	StopCondition    string   `json:"stop_condition,omitempty"`
	Blocked          bool     `json:"blocked"`
	Requires         []string `json:"requires,omitempty"`
	EvidenceRequired []string `json:"evidence_required,omitempty"`
	Instruction      string   `json:"instruction"`
}

type EventEntry struct {
	SchemaVersion   int            `json:"schema_version"`
	Time            string         `json:"time"`
	Type            string         `json:"type"`
	WorkspaceStatus string         `json:"workspace_status,omitempty"`
	ActiveLayer     string         `json:"active_layer,omitempty"`
	Actor           string         `json:"actor"`
	Data            map[string]any `json:"data,omitempty"`
}

type EvidenceOptions struct {
	Criteria []string
	Command  string
	Content  string
}

type ArtifactOptions struct {
	Method   string
	Evidence EvidenceOptions
	Content  string
}

func runShow(args []string, stdout io.Writer) error {
	if len(args) > 1 {
		return errors.New("show accepts at most one artifact")
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	artifact := "state"
	if len(args) == 1 {
		artifact = args[0]
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	if state.WorkspaceType == "plan" && isWorkflowArtifact(artifact) {
		artifact = "state"
	}
	path, err := artifactPath(workspace, artifact)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = stdout.Write(content)
	return err
}

func runArtifact(artifact string, args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	options, err := parseArtifactArgs(artifact, args)
	if err != nil {
		return err
	}
	contentArgs := []string{options.Content}
	if options.Content == "" {
		contentArgs = nil
	}
	content, err := commandContent(contentArgs)
	if err != nil {
		return err
	}
	if state.WorkspaceType == "plan" {
		return runPlanArtifact(workspace, artifact, content, options.Method, stdout)
	}
	path, err := artifactPath(workspace, artifact)
	if err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		fileContent, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		_, writeErr := stdout.Write(fileContent)
		return writeErr
	}
	if err := appendArtifact(path, artifact, content); err != nil {
		return err
	}
	if err := markArtifactRecorded(path, artifact); err != nil {
		return err
	}
	if artifact == "evidence" {
		options.Evidence.Content = content
	}
	if err := updateStateForArtifact(workspace, artifact, content, options.Method, options.Evidence); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "recorded: %s\n", artifact)
	return nil
}

func runPlanArtifact(workspace, artifact, content, method string, stdout io.Writer) error {
	path := filepath.Join(workspace, "kkt.yaml")
	if strings.TrimSpace(content) == "" {
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, writeErr := stdout.Write(fileContent)
		return writeErr
	}
	if artifact == "model" {
		missing := missingPlanModelFields(content)
		if len(missing) > 0 {
			return fmt.Errorf("plan model missing required fields: %s", strings.Join(missing, ", "))
		}
	}
	if err := appendPlanStateEntry(workspace, artifact, content); err != nil {
		return err
	}
	if artifact == "model" {
		if err := markPlanContractComplete(workspace); err != nil {
			return err
		}
	}
	if err := updateStateForArtifact(workspace, artifact, content, method, EvidenceOptions{}); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "recorded: %s\n", artifact)
	return nil
}

func missingPlanModelFields(content string) []string {
	text := strings.ToLower(content)
	var missing []string
	for _, field := range requiredPlanContractFields() {
		if field == "planning_contract" {
			continue
		}
		if !strings.Contains(text, field) {
			missing = append(missing, field)
		}
	}
	return missing
}

func runApprove(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	if state.WorkspaceType == "run" || state.WorkspaceType == "loop" {
		result, judgeErr := judgeWorkspace(".", workspace, "model-ready")
		if judgeErr != nil {
			return judgeErr
		}
		if result.Verdict != "allow" {
			return fmt.Errorf("approval requires a model-ready execution contract: %s", result.Reason)
		}
	}
	scope := strings.TrimSpace(strings.Join(args, " "))
	if scope == "" {
		scope = "Approved selected KKT model."
	}
	if err := updateApproval(workspace, "approved", scope); err != nil {
		return err
	}
	if err := writeApprovalBaseline(workspace); err != nil {
		return err
	}
	if err := updateTopLevelState(workspace, "status", "approved"); err != nil {
		return err
	}
	if err := appendEvent(workspace, "approval_granted", map[string]any{"scope": scope}); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "approved: %s\n", scope)
	return nil
}

func invalidateExecutionApproval(workspace, reason string) error {
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	if (state.WorkspaceType != "run" && state.WorkspaceType != "loop") || state.ApprovalStatus != "approved" {
		return nil
	}
	if err := updateApproval(workspace, "pending", ""); err != nil {
		return err
	}
	if err := updateTopLevelState(workspace, "status", "modeling"); err != nil {
		return err
	}
	if err := disarmHooksForWorkspace(workspace); err != nil {
		return err
	}
	return appendEvent(workspace, "approval_invalidated", map[string]any{"reason": reason})
}

func runBlock(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	reason, err := commandContent(args)
	if err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("block requires a reason")
	}
	if err := updateTopLevelState(workspace, "status", "blocked"); err != nil {
		return err
	}
	if err := disarmHooksForWorkspace(workspace); err != nil {
		return err
	}
	loop, err := readLoopState(workspace)
	if err == nil {
		loop.StopConditions = append(loop.StopConditions, LoopStopCondition{
			ID:     uniqueID("block-" + slugify(reason)),
			Text:   reason,
			Status: "active",
		})
		if writeErr := writeLoopState(workspace, loop); writeErr != nil {
			return writeErr
		}
	}
	if err := appendEvent(workspace, "blocked", map[string]any{"reason": reason}); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "blocked: %s\n", reason)
	return nil
}

func runResume(args []string, stdout io.Writer) error {
	if len(args) > 1 {
		return errors.New("resume accepts at most one workspace path")
	}
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
	fmt.Fprintf(stdout, "approval: %s\n", state.ApprovalStatus)
	if loop, loopErr := readLoopState(workspace); loopErr == nil {
		printResumeLoop(stdout, loop)
	}
	result, validationErr := ValidateWorkspace(workspace)
	if validationErr == nil {
		if result.OK {
			fmt.Fprintln(stdout, "validation: valid")
		} else {
			fmt.Fprintln(stdout, "validation: invalid")
			for _, issue := range result.Issues {
				fmt.Fprintf(stdout, "- %s\n", issue)
			}
		}
	}
	events, eventsErr := readEvents(workspace, 5)
	if eventsErr == nil && len(events) > 0 {
		fmt.Fprintln(stdout, "recent_events:")
		for _, event := range events {
			fmt.Fprintf(stdout, "- %s %s %s\n", event.Time, event.Type, eventDataSummary(event.Data))
		}
	}
	fmt.Fprintln(stdout, nextInstructionForWorkspace(workspace, state))
	return nil
}

func runDone(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	result, err := ValidateWorkspace(workspace)
	if err != nil {
		return err
	}
	if !result.OK {
		for _, issue := range result.Issues {
			fmt.Fprintf(stdout, "- %s\n", issue)
		}
		return errors.New("workspace validation failed")
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	if issues := completeLayerIssues(workspace, state.WorkspaceType); len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(stdout, "- %s\n", issue)
		}
		return errors.New("workspace validation failed")
	}
	summary := strings.TrimSpace(strings.Join(args, " "))
	if summary == "" {
		summary = "KKT workflow complete."
	}
	if err := updateTopLevelState(workspace, "status", "complete"); err != nil {
		return err
	}
	if err := disarmHooksForWorkspace(workspace); err != nil {
		return err
	}
	if err := updateTopLevelState(workspace, "active_layer", "validation"); err != nil {
		return err
	}
	if err := appendEvent(workspace, "done", map[string]any{"summary": summary}); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "complete: %s\n", workspace)
	return nil
}

func runTask(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	if state.WorkspaceType != "loop" {
		return errors.New("task requires a loop workspace")
	}
	loop, err := readLoopState(workspace)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return printTasks(stdout, loop)
	}
	action := args[0]
	switch action {
	case "add":
		title := strings.TrimSpace(strings.Join(args[1:], " "))
		if title == "" {
			return errors.New("task add requires a title")
		}
		task := LoopTask{ID: uniqueTaskID(loop, slugify(title)), Title: title, Status: "pending"}
		loop.Tasks = append(loop.Tasks, task)
		if err := writeLoopState(workspace, loop); err != nil {
			return err
		}
		if err := invalidateExecutionApproval(workspace, "loop tasks changed"); err != nil {
			return err
		}
		if err := appendEvent(workspace, "task_added", map[string]any{"task": task.ID, "title": title}); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "task added: %s\n", task.ID)
		return nil
	case "start", "done", "skip", "block":
		id := strings.TrimSpace(strings.Join(args[1:], " "))
		if id == "" {
			id = loop.CurrentTask
		}
		if id == "" {
			return fmt.Errorf("task %s requires a task id", action)
		}
		nextStatus := map[string]string{
			"start": "active",
			"done":  "done",
			"skip":  "skipped",
			"block": "blocked",
		}[action]
		if err := setTaskStatus(&loop, id, nextStatus); err != nil {
			return err
		}
		if nextStatus == "active" {
			loop.CurrentTask = id
		}
		if loop.CurrentTask == id && (nextStatus == "done" || nextStatus == "skipped" || nextStatus == "blocked") {
			loop.CurrentTask = ""
		}
		if nextStatus == "blocked" {
			if err := updateTopLevelState(workspace, "status", "blocked"); err != nil {
				return err
			}
		}
		if err := writeLoopState(workspace, loop); err != nil {
			return err
		}
		if err := appendEvent(workspace, "task_"+action, map[string]any{"task": id}); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "task %s: %s\n", action, id)
		return nil
	default:
		return fmt.Errorf("unsupported task action %q", action)
	}
}

func runCriteria(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	if state.WorkspaceType != "loop" {
		return errors.New("criteria requires a loop workspace")
	}
	loop, err := readLoopState(workspace)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		if len(loop.AcceptanceCriteria) == 0 {
			fmt.Fprintln(stdout, "criteria: none")
			return nil
		}
		for _, criterion := range loop.AcceptanceCriteria {
			fmt.Fprintf(stdout, "- %s [%s] %s\n", criterion.ID, criterion.Status, criterion.Text)
		}
		return nil
	}
	action := args[0]
	switch action {
	case "add":
		text := strings.TrimSpace(strings.Join(args[1:], " "))
		if text == "" {
			return errors.New("criteria add requires text")
		}
		criterion := LoopCriterion{ID: uniqueCriterionID(loop, slugify(text)), Text: text, Status: "pending"}
		loop.AcceptanceCriteria = append(loop.AcceptanceCriteria, criterion)
		if err := writeLoopState(workspace, loop); err != nil {
			return err
		}
		if err := invalidateExecutionApproval(workspace, "loop acceptance criteria changed"); err != nil {
			return err
		}
		if err := appendEvent(workspace, "criterion_added", map[string]any{"criterion": criterion.ID, "text": text}); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "criterion added: %s\n", criterion.ID)
		return nil
	case "satisfy", "block":
		id := strings.TrimSpace(strings.Join(args[1:], " "))
		if id == "" {
			return fmt.Errorf("criteria %s requires a criterion id", action)
		}
		status := "satisfied"
		if action == "block" {
			status = "blocked"
		}
		if err := setCriterionStatus(&loop, id, status); err != nil {
			return err
		}
		if status == "blocked" {
			if err := updateTopLevelState(workspace, "status", "blocked"); err != nil {
				return err
			}
		}
		if err := writeLoopState(workspace, loop); err != nil {
			return err
		}
		if err := appendEvent(workspace, "criterion_"+action, map[string]any{"criterion": id}); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "criterion %s: %s\n", action, id)
		return nil
	default:
		return fmt.Errorf("unsupported criteria action %q", action)
	}
}

func commandContent(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	stdin, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if stdin.Mode()&os.ModeCharDevice == 0 {
		var builder strings.Builder
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			builder.WriteString(scanner.Text())
			builder.WriteByte('\n')
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return strings.TrimRight(builder.String(), "\n"), nil
	}
	return strings.Join(args, " "), nil
}

func parseArtifactArgs(artifact string, args []string) (ArtifactOptions, error) {
	options := ArtifactOptions{}
	content := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--method":
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return ArtifactOptions{}, fmt.Errorf("%s --method requires a method", artifact)
			}
			if !artifactAcceptsMethod(artifact) {
				return ArtifactOptions{}, fmt.Errorf("%s does not accept flag: --method", artifact)
			}
			method := strings.TrimSpace(args[i])
			if !validLayerMethod(artifact, method) {
				return ArtifactOptions{}, fmt.Errorf("unsupported %s method: %s", artifact, method)
			}
			options.Method = method
		case "--for":
			if artifact != "evidence" {
				return ArtifactOptions{}, fmt.Errorf("%s does not accept flag: --for", artifact)
			}
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return ArtifactOptions{}, errors.New("evidence --for requires a criterion id")
			}
			options.Evidence.Criteria = append(options.Evidence.Criteria, splitCSV(args[i])...)
		case "--command":
			if artifact != "evidence" {
				return ArtifactOptions{}, fmt.Errorf("%s does not accept flag: --command", artifact)
			}
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return ArtifactOptions{}, errors.New("evidence --command requires a command")
			}
			options.Evidence.Command = strings.TrimSpace(args[i])
		default:
			if strings.HasPrefix(arg, "-") {
				return ArtifactOptions{}, fmt.Errorf("%s does not accept flag: %s", artifact, arg)
			}
			content = append(content, arg)
		}
	}
	options.Evidence.Criteria = uniqueStrings(options.Evidence.Criteria)
	options.Content = strings.TrimSpace(strings.Join(content, " "))
	return options, nil
}

func artifactAcceptsMethod(artifact string) bool {
	switch artifact {
	case "intent", "discovery", "model":
		return true
	default:
		return false
	}
}

func validLayerMethod(artifact, method string) bool {
	allowed := map[string][]string{
		"intent":    {"goal_anti_goal", "why_how", "obstacle_questions", "pairwise_questions"},
		"discovery": {"naive", "traceability_matrix", "coupling_map", "dsm_lite"},
		"model":     {"lexicographic", "decision_tree", "shortest_path", "ordinal_mcda", "pairwise_ahp", "outranking"},
	}
	for _, candidate := range allowed[artifact] {
		if method == candidate {
			return true
		}
	}
	return false
}

func artifactPath(workspace, artifact string) (string, error) {
	switch artifact {
	case "state", "yaml", "kkt":
		return filepath.Join(workspace, "kkt.yaml"), nil
	case "guardrails":
		return filepath.Join(workspace, "guardrails.json"), nil
	case "intent", "discovery", "model", "plan", "progress", "evidence", "notes":
		return filepath.Join(workspace, artifact+".md"), nil
	case "events", "log":
		return filepath.Join(workspace, "events.jsonl"), nil
	default:
		return "", fmt.Errorf("unsupported artifact %q", artifact)
	}
}

func isWorkflowArtifact(artifact string) bool {
	switch artifact {
	case "intent", "discovery", "model", "plan", "progress", "evidence", "notes":
		return true
	default:
		return false
	}
}

func appendArtifact(path, artifact, content string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		title := artifactTitle(artifact)
		if err := os.WriteFile(path, []byte("# "+title+"\n\n"), 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	entry := fmt.Sprintf("\n## %s\n\n%s\n", time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(content))
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(entry)
	return err
}

func markArtifactRecorded(path, artifact string) error {
	if artifact == "notes" {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	next := strings.Replace(string(content), "Status: pending", "Status: recorded", 1)
	if next == string(content) {
		return nil
	}
	return os.WriteFile(path, []byte(next), 0o644)
}

func updateStateForArtifact(workspace, artifact, content, method string, evidenceOptions EvidenceOptions) error {
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	layerForArtifact := map[string]string{
		"intent":    "intent",
		"discovery": "discovery",
		"model":     "modeling",
		"plan":      "execution",
		"evidence":  "validation",
	}
	if layer, ok := layerForArtifact[artifact]; ok {
		if err := updateLayerState(workspace, layer, "complete", method, firstLine(content)); err != nil {
			return err
		}
		if method != "" {
			if err := appendMethodInvocation(workspace, layer, method, content); err != nil {
				return err
			}
		}
		if err := updateTopLevelState(workspace, "active_layer", nextActiveLayer(state, artifact)); err != nil {
			return err
		}
	}
	if artifact == "model" && (state.WorkspaceType == "run" || state.WorkspaceType == "loop") {
		if err := updateLayerState(workspace, "execution", "pending", "pending", "Model changed; record an updated execution plan before approval."); err != nil {
			return err
		}
	}
	if artifact == "model" || artifact == "plan" {
		if err := invalidateExecutionApproval(workspace, artifact+" changed"); err != nil {
			return err
		}
	}
	if artifact == "evidence" {
		loop, err := readLoopState(workspace)
		if err == nil {
			summary := firstLine(content)
			if summary == "" {
				summary = "Evidence recorded."
			}
			loop.Evidence = append(loop.Evidence, LoopEvidence{
				ID:       uniqueID("evidence"),
				Summary:  summary,
				Status:   "recorded",
				Criteria: evidenceOptions.Criteria,
				Command:  evidenceOptions.Command,
			})
			if writeErr := writeLoopState(workspace, loop); writeErr != nil {
				return writeErr
			}
		}
	}
	data := map[string]any{"summary": firstLine(content)}
	if method != "" {
		data["method"] = method
	}
	if artifact == "evidence" {
		if len(evidenceOptions.Criteria) > 0 {
			data["criteria"] = evidenceOptions.Criteria
		}
		if evidenceOptions.Command != "" {
			data["command"] = evidenceOptions.Command
		}
	}
	return appendEvent(workspace, artifact+"_recorded", data)
}

func nextActiveLayer(state State, artifact string) string {
	switch artifact {
	case "intent":
		return "discovery"
	case "discovery":
		return "modeling"
	case "model":
		if state.WorkspaceType == "model" {
			return "validation"
		}
		if state.WorkspaceType == "loop" || state.WorkspaceType == "run" {
			return "execution"
		}
		return "modeling"
	case "plan":
		return "execution"
	case "evidence":
		return "validation"
	default:
		return state.ActiveLayer
	}
}

func updateLayerState(workspace, layer, status, method, summary string) error {
	path := filepath.Join(workspace, "kkt.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	layerLine := "  " + layer + ":"
	start := -1
	end := len(lines)
	for i, line := range lines {
		if line == layerLine {
			start = i
			continue
		}
		if start >= 0 && i > start {
			if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
				end = i
				break
			}
			if line != "" && !strings.HasPrefix(line, " ") {
				end = i
				break
			}
		}
	}
	if start < 0 {
		return nil
	}
	if summary == "" {
		summary = "Recorded " + layer + "."
	}
	fields := map[string]string{
		"status":  status,
		"summary": summary,
	}
	if method != "" {
		fields["method"] = method
	}
	seen := map[string]bool{}
	for i := start + 1; i < end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		key, _, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		value, exists := fields[key]
		if !exists {
			continue
		}
		lines[i] = "    " + key + ": " + quoteYAML(value)
		seen[key] = true
	}
	insert := []string{}
	for _, key := range []string{"status", "method", "summary"} {
		value, exists := fields[key]
		if !exists || seen[key] {
			continue
		}
		insert = append(insert, "    "+key+": "+quoteYAML(value))
	}
	if len(insert) > 0 {
		next := append([]string{}, lines[:end]...)
		next = append(next, insert...)
		next = append(next, lines[end:]...)
		lines = next
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func markValidationLayerComplete(workspace string) error {
	if err := updateLayerState(workspace, "validation", "complete", "validated", "Workspace validation passed."); err != nil {
		return err
	}
	return updateTopLevelState(workspace, "active_layer", "validation")
}

func appendMethodInvocation(workspace, layer, method, content string) error {
	path := filepath.Join(workspace, "kkt.yaml")
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	summary := firstLine(content)
	if summary == "" {
		summary = "Recorded " + layer + "."
	}
	entry := []string{
		"  - layer: " + quoteYAML(layer),
		"    method: " + quoteYAML(method),
		"    reason: " + quoteYAML(summary),
		"    inputs: " + quoteYAML("current workspace state"),
		"    outputs: " + quoteYAML(layerArtifact(layer)),
	}
	lines := strings.Split(string(fileContent), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "method_invocations: []" {
			next := append([]string{}, lines[:i]...)
			next = append(next, "method_invocations:")
			next = append(next, entry...)
			next = append(next, lines[i+1:]...)
			return os.WriteFile(path, []byte(strings.Join(next, "\n")), 0o644)
		}
		if strings.TrimSpace(line) == "method_invocations:" {
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if lines[j] != "" && !strings.HasPrefix(lines[j], " ") {
					end = j
					break
				}
			}
			next := append([]string{}, lines[:end]...)
			next = append(next, entry...)
			next = append(next, lines[end:]...)
			return os.WriteFile(path, []byte(strings.Join(next, "\n")), 0o644)
		}
	}
	lines = append(lines, "method_invocations:")
	lines = append(lines, entry...)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func layerArtifact(layer string) string {
	switch layer {
	case "modeling":
		return "model.md"
	case "execution":
		return "plan.md"
	case "validation":
		return "evidence.md"
	default:
		return layer + ".md"
	}
}

func updateTopLevelState(workspace, key, value string) error {
	path := filepath.Join(workspace, "kkt.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, " ") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		name, _, ok := strings.Cut(trimmed, ":")
		if ok && name == key {
			lines[i] = fmt.Sprintf("%s: %s", key, quoteYAML(value))
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, fmt.Sprintf("%s: %s", key, quoteYAML(value)))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func updateApproval(workspace, status, scope string) error {
	path := filepath.Join(workspace, "kkt.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	inApproval := false
	statusSet := false
	scopeSet := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if line == "approval:" {
			inApproval = true
			continue
		}
		if inApproval && line != "" && !strings.HasPrefix(line, " ") {
			inApproval = false
		}
		if !inApproval {
			continue
		}
		if strings.HasPrefix(trimmed, "status:") {
			lines[i] = "  status: " + quoteYAML(status)
			statusSet = true
		}
		if strings.HasPrefix(trimmed, "approved_scope:") {
			lines[i] = "  approved_scope: " + quoteYAML(scope)
			scopeSet = true
		}
	}
	if !statusSet || !scopeSet {
		lines = append(lines, "approval:")
		if !statusSet {
			lines = append(lines, "  status: "+quoteYAML(status))
		}
		if !scopeSet {
			lines = append(lines, "  approved_scope: "+quoteYAML(scope))
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func appendPlanStateEntry(workspace, artifact, content string) error {
	path := filepath.Join(workspace, "kkt.yaml")
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	summary := firstLine(content)
	if summary == "" {
		summary = strings.TrimSpace(content)
	}
	if summary == "" {
		summary = "Recorded " + artifact + "."
	}
	entry := []string{
		"  - time: " + quoteYAML(time.Now().UTC().Format(time.RFC3339)),
		"    artifact: " + quoteYAML(artifact),
		"    summary: " + quoteYAML(summary),
	}
	lines := strings.Split(string(fileContent), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "decision_log: []" {
			next := append([]string{}, lines[:i]...)
			next = append(next, "decision_log:")
			next = append(next, entry...)
			next = append(next, lines[i+1:]...)
			return os.WriteFile(path, []byte(strings.Join(next, "\n")), 0o644)
		}
		if strings.TrimSpace(line) == "decision_log:" {
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if lines[j] != "" && !strings.HasPrefix(lines[j], " ") {
					end = j
					break
				}
			}
			next := append([]string{}, lines[:end]...)
			next = append(next, entry...)
			next = append(next, lines[end:]...)
			return os.WriteFile(path, []byte(strings.Join(next, "\n")), 0o644)
		}
	}
	lines = append(lines, "decision_log:")
	lines = append(lines, entry...)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func markPlanContractComplete(workspace string) error {
	path := filepath.Join(workspace, "kkt.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fields := map[string]bool{}
	for _, field := range requiredPlanContractFields() {
		if field != "planning_contract" {
			fields[field] = true
		}
	}
	lines := strings.Split(string(content), "\n")
	activeField := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			field := strings.TrimSuffix(trimmed, ":")
			if fields[field] {
				activeField = field
			} else {
				activeField = ""
			}
			continue
		}
		if activeField != "" && strings.HasPrefix(strings.TrimSpace(line), "status:") {
			lines[i] = "    status: complete"
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func appendEvent(workspace, eventType string, data map[string]any) error {
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	if state.WorkspaceType != "loop" {
		return nil
	}
	return appendWorkspaceEvent(workspace, state, eventType, data)
}

func appendWorkspaceEvent(workspace string, state State, eventType string, data map[string]any) error {
	entry := EventEntry{
		SchemaVersion:   1,
		Time:            time.Now().UTC().Format(time.RFC3339),
		Type:            eventType,
		WorkspaceStatus: state.Status,
		ActiveLayer:     state.ActiveLayer,
		Actor:           "cli",
		Data:            data,
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(workspace, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(payload); err != nil {
		return err
	}
	_, err = file.WriteString("\n")
	return err
}

func appendValidationEvent(workspace string, result ValidationResult) error {
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	eventType := "validation_passed"
	data := map[string]any{"ok": result.OK}
	if !result.OK {
		eventType = "validation_failed"
		data["issues"] = result.Issues
	}
	return appendWorkspaceEvent(workspace, state, eventType, data)
}

func readEvents(workspace string, limit int) ([]EventEntry, error) {
	file, err := os.Open(filepath.Join(workspace, "events.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	var events []EventEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		event, parseErr := parseEventLine(line)
		if parseErr != nil {
			return nil, parseErr
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(events) > limit {
		return events[len(events)-limit:], nil
	}
	return events, nil
}

func parseEventLine(line string) (EventEntry, error) {
	raw := map[string]any{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return EventEntry{}, err
	}
	event := EventEntry{SchemaVersion: 1, Actor: "cli", Data: map[string]any{}}
	if value, ok := raw["schema_version"].(float64); ok {
		event.SchemaVersion = int(value)
	}
	if value, ok := raw["time"].(string); ok {
		event.Time = value
	}
	if value, ok := raw["type"].(string); ok {
		event.Type = value
	}
	if value, ok := raw["workspace_status"].(string); ok {
		event.WorkspaceStatus = value
	}
	if value, ok := raw["active_layer"].(string); ok {
		event.ActiveLayer = value
	}
	if value, ok := raw["actor"].(string); ok {
		event.Actor = value
	}
	if data, ok := raw["data"].(map[string]any); ok {
		event.Data = data
	} else {
		for key, value := range raw {
			switch key {
			case "schema_version", "time", "type", "workspace_status", "active_layer", "actor":
				continue
			default:
				event.Data[key] = value
			}
		}
	}
	return event, nil
}

func readLoopState(workspace string) (LoopState, error) {
	state, err := ReadState(workspace)
	if err != nil {
		return LoopState{}, err
	}
	if state.WorkspaceType != "loop" {
		return LoopState{}, errors.New("loop state is only available for loop workspaces")
	}
	content, err := os.ReadFile(filepath.Join(workspace, "kkt.yaml"))
	if err != nil {
		return LoopState{}, err
	}
	loop := LoopState{}
	lines := strings.Split(string(content), "\n")
	inLoop := false
	section := ""
	var task *LoopTask
	var criterion *LoopCriterion
	var evidence *LoopEvidence
	var stop *LoopStopCondition
	flush := func() {
		if task != nil {
			loop.Tasks = append(loop.Tasks, *task)
			task = nil
		}
		if criterion != nil {
			loop.AcceptanceCriteria = append(loop.AcceptanceCriteria, *criterion)
			criterion = nil
		}
		if evidence != nil {
			loop.Evidence = append(loop.Evidence, *evidence)
			evidence = nil
		}
		if stop != nil {
			loop.StopConditions = append(loop.StopConditions, *stop)
			stop = nil
		}
	}
	for _, line := range lines {
		if line == "loop_state:" {
			inLoop = true
			continue
		}
		if inLoop && line != "" && !strings.HasPrefix(line, " ") {
			flush()
			break
		}
		if !inLoop {
			continue
		}
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "tasks:":
			flush()
			section = "tasks"
			continue
		case "evidence:":
			flush()
			section = "evidence"
			continue
		case "acceptance_criteria:":
			flush()
			section = "acceptance_criteria"
			continue
		case "stop_conditions:":
			flush()
			section = "stop_conditions"
			continue
		}
		if strings.HasPrefix(trimmed, "current_task:") {
			loop.CurrentTask = unquoteYAML(strings.TrimSpace(strings.TrimPrefix(trimmed, "current_task:")))
			continue
		}
		if strings.HasPrefix(trimmed, "- id:") {
			flush()
			id := unquoteYAML(strings.TrimSpace(strings.TrimPrefix(trimmed, "- id:")))
			switch section {
			case "tasks":
				task = &LoopTask{ID: id}
			case "acceptance_criteria":
				criterion = &LoopCriterion{ID: id}
			case "evidence":
				evidence = &LoopEvidence{ID: id}
			case "stop_conditions":
				stop = &LoopStopCondition{ID: id}
			}
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		value = unquoteYAML(strings.TrimSpace(value))
		switch section {
		case "tasks":
			if task == nil {
				continue
			}
			if key == "title" {
				task.Title = value
			}
			if key == "status" {
				task.Status = value
			}
		case "acceptance_criteria":
			if criterion == nil {
				continue
			}
			if key == "text" {
				criterion.Text = value
			}
			if key == "status" {
				criterion.Status = value
			}
		case "evidence":
			if evidence == nil {
				continue
			}
			if key == "summary" {
				evidence.Summary = value
			}
			if key == "status" {
				evidence.Status = value
			}
			if key == "criteria" {
				evidence.Criteria = splitCSV(value)
			}
			if key == "command" {
				evidence.Command = value
			}
		case "stop_conditions":
			if stop == nil {
				continue
			}
			if key == "text" {
				stop.Text = value
			}
			if key == "status" {
				stop.Status = value
			}
		}
	}
	flush()
	return loop, nil
}

func writeLoopState(workspace string, loop LoopState) error {
	path := filepath.Join(workspace, "kkt.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	start := -1
	end := len(lines)
	for i, line := range lines {
		if line == "loop_state:" {
			start = i
			continue
		}
		if start >= 0 && i > start && line != "" && !strings.HasPrefix(line, " ") {
			end = i
			break
		}
	}
	block := renderLoopState(loop)
	if start < 0 {
		lines = append(lines, strings.Split(block, "\n")...)
	} else {
		next := append([]string{}, lines[:start]...)
		next = append(next, strings.Split(block, "\n")...)
		next = append(next, lines[end:]...)
		lines = next
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func renderLoopState(loop LoopState) string {
	lines := []string{
		"loop_state:",
		"  current_task: " + quoteYAML(loop.CurrentTask),
		"  tasks:",
	}
	for _, task := range loop.Tasks {
		lines = append(lines,
			"    - id: "+quoteYAML(task.ID),
			"      title: "+quoteYAML(task.Title),
			"      status: "+quoteYAML(task.Status),
		)
	}
	lines = append(lines, "  acceptance_criteria:")
	for _, criterion := range loop.AcceptanceCriteria {
		lines = append(lines,
			"    - id: "+quoteYAML(criterion.ID),
			"      text: "+quoteYAML(criterion.Text),
			"      status: "+quoteYAML(criterion.Status),
		)
	}
	lines = append(lines, "  evidence:")
	for _, evidence := range loop.Evidence {
		lines = append(lines,
			"    - id: "+quoteYAML(evidence.ID),
			"      summary: "+quoteYAML(evidence.Summary),
			"      status: "+quoteYAML(evidence.Status),
		)
		if len(evidence.Criteria) > 0 {
			lines = append(lines, "      criteria: "+quoteYAML(strings.Join(evidence.Criteria, ",")))
		}
		if evidence.Command != "" {
			lines = append(lines, "      command: "+quoteYAML(evidence.Command))
		}
	}
	lines = append(lines, "  stop_conditions:")
	for _, stop := range loop.StopConditions {
		lines = append(lines,
			"    - id: "+quoteYAML(stop.ID),
			"      text: "+quoteYAML(stop.Text),
			"      status: "+quoteYAML(stop.Status),
		)
	}
	return strings.Join(lines, "\n")
}

func setTaskStatus(loop *LoopState, id, status string) error {
	for i := range loop.Tasks {
		if loop.Tasks[i].ID == id {
			loop.Tasks[i].Status = status
			return nil
		}
	}
	return fmt.Errorf("unknown task %q", id)
}

func setCriterionStatus(loop *LoopState, id, status string) error {
	for i := range loop.AcceptanceCriteria {
		if loop.AcceptanceCriteria[i].ID == id {
			loop.AcceptanceCriteria[i].Status = status
			return nil
		}
	}
	return fmt.Errorf("unknown criterion %q", id)
}

func printTasks(stdout io.Writer, loop LoopState) error {
	if loop.CurrentTask != "" {
		fmt.Fprintf(stdout, "current_task: %s\n", loop.CurrentTask)
	}
	if len(loop.Tasks) == 0 {
		fmt.Fprintln(stdout, "tasks: none")
		return nil
	}
	for _, task := range loop.Tasks {
		fmt.Fprintf(stdout, "- %s [%s] %s\n", task.ID, task.Status, task.Title)
	}
	return nil
}

func printResumeLoop(stdout io.Writer, loop LoopState) {
	if loop.CurrentTask != "" {
		fmt.Fprintf(stdout, "current_task: %s\n", loop.CurrentTask)
	}
	printTaskGroup(stdout, "pending_tasks", loop.Tasks, "pending")
	printTaskGroup(stdout, "blocked_tasks", loop.Tasks, "blocked")
	printCriterionGroup(stdout, "unsatisfied_criteria", loop.AcceptanceCriteria, "pending")
	printCriterionGroup(stdout, "blocked_criteria", loop.AcceptanceCriteria, "blocked")
	if len(loop.Evidence) > 0 {
		fmt.Fprintln(stdout, "latest_evidence:")
		start := len(loop.Evidence) - 3
		if start < 0 {
			start = 0
		}
		for _, evidence := range loop.Evidence[start:] {
			fmt.Fprintf(stdout, "- %s [%s] %s", evidence.ID, evidence.Status, evidence.Summary)
			if len(evidence.Criteria) > 0 {
				fmt.Fprintf(stdout, " (criteria: %s)", strings.Join(evidence.Criteria, ","))
			}
			if evidence.Command != "" {
				fmt.Fprintf(stdout, " (command: %s)", evidence.Command)
			}
			fmt.Fprintln(stdout)
		}
	}
}

func printTaskGroup(stdout io.Writer, title string, tasks []LoopTask, status string) {
	var matches []LoopTask
	for _, task := range tasks {
		if task.Status == status {
			matches = append(matches, task)
		}
	}
	if len(matches) == 0 {
		return
	}
	fmt.Fprintf(stdout, "%s:\n", title)
	for _, task := range matches {
		fmt.Fprintf(stdout, "- %s %s\n", task.ID, task.Title)
	}
}

func printCriterionGroup(stdout io.Writer, title string, criteria []LoopCriterion, status string) {
	var matches []LoopCriterion
	for _, criterion := range criteria {
		if criterion.Status == status {
			matches = append(matches, criterion)
		}
	}
	if len(matches) == 0 {
		return
	}
	fmt.Fprintf(stdout, "%s:\n", title)
	for _, criterion := range matches {
		fmt.Fprintf(stdout, "- %s %s\n", criterion.ID, criterion.Text)
	}
}

func nextInstructionForWorkspace(workspace string, state State) string {
	return nextActionForWorkspace(workspace, state).Instruction
}

func statusReport(workspace string, state State) (StatusReport, error) {
	validation, err := ValidateWorkspace(workspace)
	if err != nil {
		return StatusReport{}, err
	}
	report := StatusReport{
		SchemaVersion: 1,
		Workspace:     workspace,
		Status:        state.Status,
		ActiveLayer:   state.ActiveLayer,
		Profile:       state.Profile,
		Approval:      state.ApprovalStatus,
		Validation:    validation,
		StaleComplete: state.Status == "complete" && !validation.OK,
		Next:          nextActionForWorkspace(workspace, state),
	}
	if loop, loopErr := readLoopState(workspace); loopErr == nil {
		report.CurrentTask = loop.CurrentTask
	}
	return report, nil
}

func continueLayerAction(state State) NextAction {
	instruction := NextInstruction(state)
	return NextAction{
		SchemaVersion: 1,
		Action:        "continue_layer",
		Reason:        "active layer is " + state.ActiveLayer,
		Blocked:       false,
		Instruction:   instruction,
	}
}

func nextActionForWorkspace(workspace string, state State) NextAction {
	if state.Status == "complete" {
		validation, err := ValidateWorkspace(workspace)
		if err == nil && !validation.OK {
			instruction := "next: repair stale complete state by satisfying validation, then run kkt judge --checkpoint finalize --json and kkt done"
			requires := []string{"kkt validate", "kkt judge --checkpoint finalize --json", "kkt done"}
			if hasRequiredValidationCommands(workspace) {
				instruction = "next: run kkt validate --run, then kkt judge --checkpoint finalize --json and kkt done"
				requires[0] = "kkt validate --run"
			}
			return NextAction{
				SchemaVersion: 1,
				Action:        "repair_invalid_completion",
				Reason:        "stored status is complete but validation is invalid",
				Blocked:       true,
				Requires:      requires,
				Instruction:   instruction,
			}
		}
	}
	if state.ActiveLayer == "validation" && (state.WorkspaceType == "run" || state.WorkspaceType == "loop") && hasRequiredValidationCommands(workspace) {
		instruction := "next: run kkt validate --run, then kkt judge --checkpoint finalize --json and kkt done"
		return NextAction{SchemaVersion: 1, Action: "run_required_validation", Reason: "guardrails define required validation commands", Blocked: false, Requires: []string{"kkt validate --run", "kkt judge --checkpoint finalize --json", "kkt done"}, Instruction: instruction}
	}
	if state.WorkspaceType == "run" && state.ActiveLayer == "execution" && state.ApprovalStatus != "approved" {
		if issues := executionContractReadinessIssues(workspace, state); len(issues) > 0 {
			instruction := "next: record the execution plan with kkt plan, then run kkt judge --checkpoint model-ready --json before requesting approval"
			return NextAction{SchemaVersion: 1, Action: "materialize_execution_contract", Reason: "execution contract is incomplete", Blocked: false, Requires: []string{"kkt plan", "kkt judge --checkpoint model-ready --json"}, Instruction: instruction}
		}
		instruction := "next: show the selected model and record approval with kkt approve before execution"
		return NextAction{SchemaVersion: 1, Action: "request_approval", Reason: "approval is " + state.ApprovalStatus, Blocked: true, Requires: []string{"kkt approve"}, Instruction: instruction}
	}
	if state.WorkspaceType != "loop" {
		return continueLayerAction(state)
	}
	loop, err := readLoopState(workspace)
	if err != nil {
		instruction := NextInstruction(state)
		return NextAction{
			SchemaVersion: 1,
			Action:        "inspect_state",
			Reason:        "loop state could not be read",
			Blocked:       false,
			Instruction:   instruction,
		}
	}
	for _, stop := range loop.StopConditions {
		if stop.Status == "active" {
			instruction := "next: resolve active stop condition: " + stop.Text
			return NextAction{SchemaVersion: 1, Action: "resolve_stop_condition", Reason: "stop condition is active", StopCondition: stop.Text, Blocked: true, Requires: []string{"user_or_system_unblock"}, Instruction: instruction}
		}
	}
	if state.ActiveLayer != "execution" && state.ActiveLayer != "validation" {
		return continueLayerAction(state)
	}
	if state.ActiveLayer == "execution" && state.ApprovalStatus != "approved" {
		if issues := executionContractReadinessIssues(workspace, state); len(issues) > 0 {
			instruction := "next: record the execution plan, add tasks and acceptance criteria, then run kkt judge --checkpoint model-ready --json before requesting approval"
			return NextAction{SchemaVersion: 1, Action: "materialize_execution_contract", Reason: "execution contract is incomplete", Blocked: false, Requires: []string{"kkt plan", "kkt task add", "kkt criteria add", "kkt judge --checkpoint model-ready --json"}, Instruction: instruction}
		}
		instruction := "next: show the selected model and record approval with kkt approve before execution"
		return NextAction{SchemaVersion: 1, Action: "request_approval", Reason: "approval is " + state.ApprovalStatus, Blocked: true, Requires: []string{"kkt approve"}, Instruction: instruction}
	}
	if loop.CurrentTask == "" && len(loop.Tasks) == 0 && len(loop.AcceptanceCriteria) == 0 {
		return continueLayerAction(state)
	}
	if state.ApprovalStatus != "approved" {
		instruction := "next: show the selected model and record approval with kkt approve before execution"
		return NextAction{SchemaVersion: 1, Action: "request_approval", Reason: "approval is " + state.ApprovalStatus, Blocked: true, Requires: []string{"kkt approve"}, Instruction: instruction}
	}
	if loop.CurrentTask != "" {
		instruction := "next: complete current task " + loop.CurrentTask + ", record progress and evidence, then run kkt validate"
		return NextAction{SchemaVersion: 1, Action: "complete_current_task", Reason: "current task is active", TaskID: loop.CurrentTask, Blocked: false, Requires: []string{"kkt progress", "kkt evidence", "kkt validate"}, Instruction: instruction}
	}
	for _, task := range loop.Tasks {
		if task.Status == "pending" {
			instruction := "next: run kkt task start " + task.ID
			return NextAction{SchemaVersion: 1, Action: "start_task", Reason: "first pending task", TaskID: task.ID, Blocked: false, Requires: []string{"kkt task start " + task.ID}, Instruction: instruction}
		}
		if task.Status == "active" {
			instruction := "next: complete active task " + task.ID + ", record progress and evidence, then run kkt validate"
			return NextAction{SchemaVersion: 1, Action: "complete_task", Reason: "task is active", TaskID: task.ID, Blocked: false, Requires: []string{"kkt progress", "kkt evidence", "kkt validate"}, Instruction: instruction}
		}
		if task.Status == "blocked" {
			instruction := "next: resolve blocked task " + task.ID
			return NextAction{SchemaVersion: 1, Action: "resolve_blocked_task", Reason: "task is blocked", TaskID: task.ID, Blocked: true, Requires: []string{"user_or_system_unblock"}, Instruction: instruction}
		}
	}
	for _, criterion := range loop.AcceptanceCriteria {
		if criterion.Status == "pending" {
			instruction := "next: satisfy acceptance criterion " + criterion.ID + " with evidence, then run kkt criteria satisfy " + criterion.ID
			return NextAction{SchemaVersion: 1, Action: "satisfy_criterion", Reason: "acceptance criterion is pending", CriterionID: criterion.ID, Blocked: false, Requires: []string{"kkt evidence --for " + criterion.ID, "kkt criteria satisfy " + criterion.ID}, EvidenceRequired: []string{criterion.ID}, Instruction: instruction}
		}
		if criterion.Status == "blocked" {
			instruction := "next: resolve blocked acceptance criterion " + criterion.ID
			return NextAction{SchemaVersion: 1, Action: "resolve_blocked_criterion", Reason: "acceptance criterion is blocked", CriterionID: criterion.ID, Blocked: true, Requires: []string{"user_or_system_unblock"}, Instruction: instruction}
		}
	}
	instruction := "next: run kkt validate, then kkt done when acceptance criteria and evidence are complete"
	requires := []string{"kkt validate", "kkt done"}
	if hasRequiredValidationCommands(workspace) {
		instruction = "next: run kkt validate --run, then kkt done when acceptance criteria and evidence are complete"
		requires[0] = "kkt validate --run"
	}
	return NextAction{SchemaVersion: 1, Action: "validate_then_done", Reason: "no open tasks or criteria remain", Blocked: false, Requires: requires, Instruction: instruction}
}

func hasRequiredValidationCommands(workspace string) bool {
	commands, err := requiredValidationCommands(workspace)
	return err == nil && len(commands) > 0
}

func uniqueTaskID(loop LoopState, base string) string {
	if base == "" {
		base = "task"
	}
	id := base
	n := 2
	for taskIDExists(loop, id) {
		id = fmt.Sprintf("%s-%d", base, n)
		n++
	}
	return id
}

func uniqueCriterionID(loop LoopState, base string) string {
	if base == "" {
		base = "criterion"
	}
	id := base
	n := 2
	for criterionIDExists(loop, id) {
		id = fmt.Sprintf("%s-%d", base, n)
		n++
	}
	return id
}

func criterionIDExists(loop LoopState, id string) bool {
	for _, criterion := range loop.AcceptanceCriteria {
		if criterion.ID == id {
			return true
		}
	}
	return false
}

func taskIDExists(loop LoopState, id string) bool {
	for _, task := range loop.Tasks {
		if task.ID == id {
			return true
		}
	}
	return false
}

func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, time.Now().UTC().Format("20060102-150405"))
}

func runReplay(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "--check" {
		return errors.New("replay requires --check")
	}
	if len(args) > 2 {
		return errors.New("replay --check accepts at most one path")
	}
	workspace, err := ResolveWorkspace(".", firstArg(args[1:]))
	if err != nil {
		return err
	}
	result, err := CheckReplay(workspace)
	if err != nil {
		return err
	}
	if result.OK {
		fmt.Fprintf(stdout, "replay ok: %s\n", workspace)
		return nil
	}
	fmt.Fprintf(stdout, "replay drift: %s\n", workspace)
	for _, issue := range result.Issues {
		fmt.Fprintf(stdout, "- %s\n", issue)
	}
	return errors.New("replay check failed")
}

func CheckReplay(workspace string) (ValidationResult, error) {
	state, err := ReadState(workspace)
	if err != nil {
		return ValidationResult{}, err
	}
	result := ValidationResult{OK: true}
	if state.WorkspaceType != "loop" {
		return result, nil
	}
	loop, err := readLoopState(workspace)
	if err != nil {
		return ValidationResult{}, err
	}
	events, err := readEvents(workspace, 0)
	if err != nil {
		result.OK = false
		result.Issues = append(result.Issues, "events.jsonl parse failed: "+err.Error())
		return result, nil
	}
	taskEvents := map[string]string{}
	criterionEvents := map[string]string{}
	for _, event := range events {
		switch event.Type {
		case "task_start":
			if id := eventString(event.Data, "task"); id != "" {
				taskEvents[id] = "active"
			}
		case "task_done":
			if id := eventString(event.Data, "task"); id != "" {
				taskEvents[id] = "done"
			}
		case "task_skip":
			if id := eventString(event.Data, "task"); id != "" {
				taskEvents[id] = "skipped"
			}
		case "task_block":
			if id := eventString(event.Data, "task"); id != "" {
				taskEvents[id] = "blocked"
			}
		case "criterion_satisfy":
			if id := eventString(event.Data, "criterion"); id != "" {
				criterionEvents[id] = "satisfied"
			}
		case "criterion_block":
			if id := eventString(event.Data, "criterion"); id != "" {
				criterionEvents[id] = "blocked"
			}
		}
	}
	for _, task := range loop.Tasks {
		if status, ok := taskEvents[task.ID]; ok && status != task.Status {
			result.OK = false
			result.Issues = append(result.Issues, fmt.Sprintf("task %s event status %s disagrees with kkt.yaml status %s", task.ID, status, task.Status))
		}
	}
	for _, criterion := range loop.AcceptanceCriteria {
		if status, ok := criterionEvents[criterion.ID]; ok && status != criterion.Status {
			result.OK = false
			result.Issues = append(result.Issues, fmt.Sprintf("criterion %s event status %s disagrees with kkt.yaml status %s", criterion.ID, status, criterion.Status))
		}
	}
	for _, issue := range evidenceMappingIssues(loop) {
		result.OK = false
		result.Issues = append(result.Issues, issue)
	}
	return result, nil
}

func evidenceMappingIssues(loop LoopState) []string {
	var issues []string
	for _, criterion := range loop.AcceptanceCriteria {
		if criterion.Status != "satisfied" {
			continue
		}
		if !hasRecordedEvidenceForCriterion(loop, criterion.ID) {
			issues = append(issues, fmt.Sprintf("criterion %s is satisfied without mapped evidence", criterion.ID))
		}
	}
	for _, evidence := range loop.Evidence {
		if evidence.Status != "recorded" {
			issues = append(issues, fmt.Sprintf("evidence %s is %s", evidence.ID, evidence.Status))
		}
		if strings.TrimSpace(evidence.Summary) == "" {
			issues = append(issues, fmt.Sprintf("evidence %s has empty summary", evidence.ID))
		}
	}
	return issues
}

func hasRecordedEvidenceForCriterion(loop LoopState, criterionID string) bool {
	for _, evidence := range loop.Evidence {
		if evidence.Status != "recorded" {
			continue
		}
		for _, mapped := range evidence.Criteria {
			if mapped == criterionID {
				return true
			}
		}
	}
	return false
}

func splitCSV(value string) []string {
	var parts []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func eventString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func eventDataSummary(data map[string]any) string {
	if len(data) == 0 {
		return ""
	}
	if summary := eventString(data, "summary"); summary != "" {
		return summary
	}
	if task := eventString(data, "task"); task != "" {
		return task
	}
	if criterion := eventString(data, "criterion"); criterion != "" {
		return criterion
	}
	if reason := eventString(data, "reason"); reason != "" {
		return reason
	}
	return fmt.Sprintf("%v", data)
}

func artifactTitle(artifact string) string {
	parts := strings.Fields(strings.ReplaceAll(artifact, "-", " "))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func quoteYAML(value string) string {
	return strconv.Quote(value)
}

func unquoteYAML(value string) string {
	if value == "" {
		return ""
	}
	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return value
	}
	return unquoted
}

func firstLine(value string) string {
	scanner := bufio.NewScanner(strings.NewReader(value))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return ""
}

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
	ID      string
	Summary string
	Status  string
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
	content, err := commandContent(args)
	if err != nil {
		return err
	}
	if state.WorkspaceType == "plan" {
		return runPlanArtifact(workspace, artifact, content, stdout)
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
	if err := updateStateForArtifact(workspace, artifact, content); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "recorded: %s\n", artifact)
	return nil
}

func runPlanArtifact(workspace, artifact, content string, stdout io.Writer) error {
	path := filepath.Join(workspace, "kkt.yaml")
	if strings.TrimSpace(content) == "" {
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, writeErr := stdout.Write(fileContent)
		return writeErr
	}
	if err := appendPlanStateEntry(workspace, artifact, content); err != nil {
		return err
	}
	if err := updateStateForArtifact(workspace, artifact, content); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "recorded: %s\n", artifact)
	return nil
}

func runApprove(args []string, stdout io.Writer) error {
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return err
	}
	scope := strings.TrimSpace(strings.Join(args, " "))
	if scope == "" {
		scope = "Approved selected KKT model."
	}
	if err := updateApproval(workspace, "approved", scope); err != nil {
		return err
	}
	if err := updateTopLevelState(workspace, "status", "approved"); err != nil {
		return err
	}
	if err := appendEvent(workspace, "approval_granted", map[string]string{"scope": scope}); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "approved: %s\n", scope)
	return nil
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
	if err := appendEvent(workspace, "blocked", map[string]string{"reason": reason}); err != nil {
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
	if loop, loopErr := readLoopState(workspace); loopErr == nil && loop.CurrentTask != "" {
		fmt.Fprintf(stdout, "current_task: %s\n", loop.CurrentTask)
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
	summary := strings.TrimSpace(strings.Join(args, " "))
	if summary == "" {
		summary = "KKT workflow complete."
	}
	if err := updateTopLevelState(workspace, "status", "complete"); err != nil {
		return err
	}
	if err := updateTopLevelState(workspace, "active_layer", "validation"); err != nil {
		return err
	}
	if err := appendEvent(workspace, "done", map[string]string{"summary": summary}); err != nil {
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
		if err := appendEvent(workspace, "task_added", map[string]string{"task": task.ID, "title": title}); err != nil {
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
		if err := appendEvent(workspace, "task_"+action, map[string]string{"task": id}); err != nil {
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
		if err := appendEvent(workspace, "criterion_added", map[string]string{"criterion": criterion.ID, "text": text}); err != nil {
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
		if err := appendEvent(workspace, "criterion_"+action, map[string]string{"criterion": id}); err != nil {
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

func artifactPath(workspace, artifact string) (string, error) {
	switch artifact {
	case "state", "yaml", "kkt":
		return filepath.Join(workspace, "kkt.yaml"), nil
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
	if artifact != "evidence" && artifact != "progress" {
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

func updateStateForArtifact(workspace, artifact, content string) error {
	layerForArtifact := map[string]string{
		"intent":    "intent",
		"discovery": "discovery",
		"model":     "modeling",
		"plan":      "execution",
		"evidence":  "validation",
	}
	if layer, ok := layerForArtifact[artifact]; ok {
		if err := updateTopLevelState(workspace, "active_layer", layer); err != nil {
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
				ID:      uniqueID("evidence"),
				Summary: summary,
				Status:  "recorded",
			})
			if writeErr := writeLoopState(workspace, loop); writeErr != nil {
				return writeErr
			}
		}
	}
	return appendEvent(workspace, artifact+"_recorded", map[string]string{"summary": firstLine(content)})
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

func appendEvent(workspace, eventType string, values map[string]string) error {
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	if state.WorkspaceType != "loop" {
		return nil
	}
	entry := map[string]string{
		"time": time.Now().UTC().Format(time.RFC3339),
		"type": eventType,
	}
	for key, value := range values {
		entry[key] = value
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

func nextInstructionForWorkspace(workspace string, state State) string {
	if state.WorkspaceType != "loop" {
		return NextInstruction(state)
	}
	loop, err := readLoopState(workspace)
	if err != nil {
		return NextInstruction(state)
	}
	for _, stop := range loop.StopConditions {
		if stop.Status == "active" {
			return "next: resolve active stop condition: " + stop.Text
		}
	}
	if loop.CurrentTask != "" {
		return "next: complete current task " + loop.CurrentTask + ", record progress and evidence, then run kkt validate"
	}
	for _, task := range loop.Tasks {
		if task.Status == "pending" {
			return "next: run kkt task start " + task.ID
		}
		if task.Status == "active" {
			return "next: complete active task " + task.ID + ", record progress and evidence, then run kkt validate"
		}
		if task.Status == "blocked" {
			return "next: resolve blocked task " + task.ID
		}
	}
	for _, criterion := range loop.AcceptanceCriteria {
		if criterion.Status == "pending" {
			return "next: satisfy acceptance criterion " + criterion.ID + " with evidence, then run kkt criteria satisfy " + criterion.ID
		}
		if criterion.Status == "blocked" {
			return "next: resolve blocked acceptance criterion " + criterion.ID
		}
	}
	return "next: run kkt validate, then kkt done when acceptance criteria and evidence are complete"
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

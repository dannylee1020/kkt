package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultHookTTL = 8 * time.Hour

type HookState struct {
	SchemaVersion int          `json:"schema_version"`
	Armed         bool         `json:"armed"`
	Mode          string       `json:"mode"`
	ProjectRoot   string       `json:"project_root,omitempty"`
	Workspace     string       `json:"workspace,omitempty"`
	WorkspaceType string       `json:"workspace_type,omitempty"`
	ArmedAt       string       `json:"armed_at,omitempty"`
	ExpiresAt     string       `json:"expires_at,omitempty"`
	Baseline      HookBaseline `json:"baseline,omitempty"`
}

type HookBaseline struct {
	RecordedAt string            `json:"recorded_at,omitempty"`
	Paths      map[string]string `json:"paths,omitempty"`
}

type HookResult struct {
	SchemaVersion int      `json:"schema_version"`
	Verdict       string   `json:"verdict"`
	Mode          string   `json:"mode"`
	Event         string   `json:"event"`
	Reason        string   `json:"reason"`
	Workspace     string   `json:"workspace,omitempty"`
	WorkspaceType string   `json:"workspace_type,omitempty"`
	Repair        []string `json:"repair,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
}

type normalizedHookPayload struct {
	Agent    string
	Event    string
	CWD      string
	ToolName string
	Command  string
	Paths    []string
}

type hooksOptions struct {
	JSON    bool
	Mode    string
	TTL     time.Duration
	HasTTL  bool
	Payload string
}

type hookCommandOptions struct {
	Agent   string
	JSON    bool
	Payload string
}

func runHooks(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("hooks requires an action: status, arm, or disarm")
	}
	action := args[0]
	switch action {
	case "status":
		options, err := parseHooksOptions("hooks status", args[1:])
		if err != nil {
			return err
		}
		projectRootDir, err := projectRoot(".")
		if err != nil {
			return err
		}
		state, _, err := readHookState(projectRootDir)
		if err != nil {
			return err
		}
		if options.JSON {
			return writeJSON(stdout, state)
		}
		if !state.Armed {
			fmt.Fprintln(stdout, "hooks: inactive")
			return nil
		}
		fmt.Fprintf(stdout, "hooks: armed\nmode: %s\nworkspace: %s\nexpires_at: %s\n", state.Mode, state.Workspace, state.ExpiresAt)
		return nil
	case "arm":
		options, err := parseHooksOptions("hooks arm", args[1:])
		if err != nil {
			return err
		}
		state, err := armHooks(options.Mode, options.TTL)
		if err != nil {
			return err
		}
		if options.JSON {
			return writeJSON(stdout, state)
		}
		fmt.Fprintf(stdout, "hooks: armed\nmode: %s\nworkspace: %s\nexpires_at: %s\n", state.Mode, state.Workspace, state.ExpiresAt)
		return nil
	case "disarm":
		options, err := parseHooksOptions("hooks disarm", args[1:])
		if err != nil {
			return err
		}
		projectRootDir, err := projectRoot(".")
		if err != nil {
			return err
		}
		if err := disarmHooks(projectRootDir); err != nil {
			return err
		}
		if options.JSON {
			return writeJSON(stdout, HookState{SchemaVersion: 1, Armed: false, Mode: "inactive"})
		}
		fmt.Fprintln(stdout, "hooks: disarmed")
		return nil
	default:
		return fmt.Errorf("unsupported hooks action %q", action)
	}
}

func parseHooksOptions(command string, args []string) (hooksOptions, error) {
	options := hooksOptions{Mode: "enforce", TTL: defaultHookTTL}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			options.JSON = true
		case "--mode":
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return hooksOptions{}, fmt.Errorf("%s --mode requires observe or enforce", command)
			}
			mode := strings.TrimSpace(args[i])
			if mode != "observe" && mode != "enforce" {
				return hooksOptions{}, fmt.Errorf("unsupported hook mode: %s", mode)
			}
			options.Mode = mode
		case "--ttl":
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return hooksOptions{}, fmt.Errorf("%s --ttl requires a duration", command)
			}
			ttl, err := time.ParseDuration(args[i])
			if err != nil {
				return hooksOptions{}, fmt.Errorf("invalid hook ttl %q: %w", args[i], err)
			}
			if ttl <= 0 {
				return hooksOptions{}, errors.New("hook ttl must be greater than zero")
			}
			options.TTL = ttl
			options.HasTTL = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return hooksOptions{}, fmt.Errorf("%s does not accept flag: %s", command, args[i])
			}
			if options.Payload != "" {
				return hooksOptions{}, fmt.Errorf("%s accepts at most one payload", command)
			}
			options.Payload = args[i]
		}
	}
	return options, nil
}

func armHooks(mode string, ttl time.Duration) (HookState, error) {
	if mode == "" {
		mode = "enforce"
	}
	if mode != "observe" && mode != "enforce" {
		return HookState{}, fmt.Errorf("unsupported hook mode: %s", mode)
	}
	if ttl <= 0 {
		ttl = defaultHookTTL
	}
	workspace, err := ResolveWorkspace(".", "")
	if err != nil {
		return HookState{}, err
	}
	state, err := ReadState(workspace)
	if err != nil {
		return HookState{}, err
	}
	if state.WorkspaceType != "run" && state.WorkspaceType != "loop" {
		return HookState{}, fmt.Errorf("hooks can only arm run or loop workspaces, got %q", state.WorkspaceType)
	}
	if state.ApprovalStatus != "approved" {
		return HookState{}, errors.New("hooks require approved workspace before arming")
	}
	guardrailIssues := validateGuardrails(workspace)
	if len(guardrailIssues) > 0 {
		return HookState{}, fmt.Errorf("hooks require valid guardrails: %s", strings.Join(guardrailIssues, "; "))
	}
	contract, err := readGuardrails(workspace)
	if err != nil {
		return HookState{}, err
	}
	if issues := guardrailExecutionReadinessIssues(contract); len(issues) > 0 {
		return HookState{}, fmt.Errorf("hooks require execution-ready guardrails: %s", strings.Join(issues, "; "))
	}
	projectRootDir, err := projectRootForWorkspace(".", workspace)
	if err != nil {
		return HookState{}, err
	}
	baseline, err := currentHookBaseline(projectRootDir)
	if err != nil {
		return HookState{}, err
	}
	now := time.Now().UTC()
	hookState := HookState{
		SchemaVersion: 1,
		Armed:         true,
		Mode:          mode,
		ProjectRoot:   projectRootDir,
		Workspace:     workspaceSourcePath(projectRootDir, workspace),
		WorkspaceType: state.WorkspaceType,
		ArmedAt:       now.Format(time.RFC3339),
		ExpiresAt:     now.Add(ttl).Format(time.RFC3339),
		Baseline:      baseline,
	}
	return hookState, writeHookState(projectRootDir, hookState)
}

func runHook(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("hook requires an event: pre-tool, post-tool, pre-compact, post-compact, or stop")
	}
	event := args[0]
	options, err := parseHookCommandOptions(args[1:])
	if err != nil {
		return err
	}
	if strings.TrimSpace(options.Payload) == "" {
		content, contentErr := commandContent(nil)
		if contentErr != nil {
			return contentErr
		}
		options.Payload = content
	}
	result := evaluateHook(".", event, options.Agent, options.Payload)
	if options.JSON {
		return writeJSON(stdout, result)
	}
	return writeAgentHookResult(stdout, event, result)
}

func parseHookCommandOptions(args []string) (hookCommandOptions, error) {
	options := hookCommandOptions{}
	content := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return hookCommandOptions{}, errors.New("hook --agent requires a value")
			}
			options.Agent = strings.TrimSpace(args[i])
		case "--json":
			options.JSON = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return hookCommandOptions{}, fmt.Errorf("hook does not accept flag: %s", args[i])
			}
			content = append(content, args[i])
		}
	}
	options.Payload = strings.TrimSpace(strings.Join(content, " "))
	return options, nil
}

func evaluateHook(root, event, agent, payload string) HookResult {
	if !validHookEvent(event) {
		return hookResult(event, "inactive", "allow", "unsupported hook event", nil, nil, "", "")
	}
	projectRootDir, err := projectRoot(root)
	if err != nil {
		return hookResult(event, "inactive", "allow", "project root could not be resolved", nil, nil, "", "")
	}
	state, exists, err := readHookState(projectRootDir)
	if err != nil {
		return hookResult(event, "inactive", "allow", "hook state could not be read", nil, []string{err.Error()}, "", "")
	}
	if !exists || !state.Armed {
		return hookResult(event, "inactive", "allow", "no armed KKT workspace", nil, nil, "", "")
	}
	if isHookExpired(state) {
		_ = disarmHooks(projectRootDir)
		return hookResult(event, "inactive", "allow", "armed KKT hooks expired and were disarmed", nil, nil, state.Workspace, state.WorkspaceType)
	}
	if state.ProjectRoot != "" && !samePath(projectRootDir, state.ProjectRoot) {
		return hookResult(event, "inactive", "allow", "armed KKT workspace belongs to another project", nil, nil, state.Workspace, state.WorkspaceType)
	}
	workspace := state.Workspace
	if workspace == "" {
		return activeHookResult(state, event, "block", "armed KKT hook state is missing workspace", []string{"run kkt hooks disarm or re-arm hooks from the active workspace"}, nil)
	}
	if !filepath.IsAbs(workspace) {
		workspace = filepath.Join(projectRootDir, filepath.FromSlash(workspace))
	}
	if info, statErr := os.Stat(workspace); statErr != nil || !info.IsDir() {
		return activeHookResult(state, event, "block", "armed KKT workspace is missing", []string{"run kkt hooks disarm or recreate the run/loop workspace"}, errorEvidence(statErr))
	}
	workspaceState, err := ReadState(workspace)
	if err != nil {
		return activeHookResult(state, event, "block", "workspace state could not be read", []string{"repair kkt.yaml or run kkt hooks disarm"}, []string{err.Error()})
	}
	contract, err := readGuardrails(workspace)
	if err != nil {
		return activeHookResult(state, event, "block", "guardrails could not be read", []string{"record valid guardrails or run kkt hooks disarm"}, []string{err.Error()})
	}
	if workspaceState.ApprovalStatus != "approved" && (event == "pre-tool" || event == "post-tool") {
		return activeHookResult(state, event, "block", "mutation requires approved KKT workspace", []string{"show the selected model and run kkt approve, or run kkt hooks disarm"}, nil)
	}

	normalized := normalizeHookPayload(agent, event, payload)
	if normalized.CWD == "" {
		normalized.CWD = projectRootDir
	}
	switch event {
	case "pre-tool":
		return evaluatePreToolHook(projectRootDir, state, normalized, contract)
	case "post-tool":
		return evaluatePostToolHook(projectRootDir, state, event, contract)
	case "pre-compact", "post-compact":
		return evaluatePostToolHook(projectRootDir, state, event, contract)
	case "stop":
		result, judgeErr := judgeWorkspace(root, state.Workspace, "finalize")
		if judgeErr != nil {
			return activeHookResult(state, event, "block", "finalize checkpoint failed", []string{"repair validation issues or run kkt hooks disarm"}, []string{judgeErr.Error()})
		}
		if result.Verdict == "block" {
			return activeHookResult(state, event, "block", result.Reason, result.Repair, result.Evidence)
		}
		return activeHookResult(state, event, "allow", "finalize checkpoint allowed", nil, result.Evidence)
	default:
		return activeHookResult(state, event, "allow", "hook event allowed", nil, nil)
	}
}

func evaluatePreToolHook(projectRootDir string, state HookState, payload normalizedHookPayload, contract GuardrailContract) HookResult {
	if isDirectMutationTool(payload.ToolName) {
		if len(payload.Paths) == 0 {
			return activeHookResult(state, payload.Event, "block", "mutating tool target path could not be determined", []string{"use a tool payload with a target path or run kkt hooks disarm"}, nil)
		}
		if issues := hookPathScopeIssues(projectRootDir, payload.CWD, payload.Paths, contract); len(issues) > 0 {
			return activeHookResult(state, payload.Event, "block", "tool target violates KKT guardrails", []string{"use only modeled allowed paths, update guardrails after re-modeling, or run kkt hooks disarm"}, issues)
		}
		return activeHookResult(state, payload.Event, "allow", "tool target is inside KKT guardrails", nil, nil)
	}
	if isShellTool(payload.ToolName) && looksMutatingShellCommand(payload.Command) && commandMentionsBlockedPath(payload.Command, contract.blockedPaths()) {
		return activeHookResult(state, payload.Event, "block", "shell command targets a blocked path", []string{"avoid blocked paths or re-model with explicit approval"}, []string{payload.Command})
	}
	return activeHookResult(state, payload.Event, "allow", "pre-tool guardrails allowed", nil, nil)
}

func evaluatePostToolHook(projectRootDir string, state HookState, event string, contract GuardrailContract) HookResult {
	issues := hookChangedPathIssues(projectRootDir, state, contract)
	if len(issues) > 0 {
		return activeHookResult(state, event, "block", "changed paths violate KKT hook baseline", []string{"revert out-of-scope changes, re-model and update guardrails, or run kkt hooks disarm"}, issues)
	}
	return activeHookResult(state, event, "allow", "changed paths respect KKT hook baseline", nil, nil)
}

func hookPathScopeIssues(projectRootDir, cwd string, paths []string, contract GuardrailContract) []string {
	allowed := contract.allowedPaths()
	blocked := contract.blockedPaths()
	var issues []string
	for _, rawPath := range paths {
		path := normalizeHookPath(projectRootDir, cwd, rawPath)
		if path == "" || isKKTPath(path) {
			continue
		}
		if matchesAnyPathPattern(path, blocked) {
			issues = append(issues, "target is blocked: "+path)
			continue
		}
		if len(allowed) > 0 && !matchesAnyPathPattern(path, allowed) {
			issues = append(issues, "target is outside allowed paths: "+path)
		}
	}
	return issues
}

func hookChangedPathIssues(projectRootDir string, state HookState, contract GuardrailContract) []string {
	changed, err := changedGitPaths(projectRootDir)
	if err != nil {
		return []string{"could not inspect changed paths: " + err.Error()}
	}
	paths := map[string]bool{}
	for _, path := range changed {
		paths[path] = true
	}
	for path := range state.Baseline.Paths {
		paths[path] = true
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)

	allowed := contract.allowedPaths()
	blocked := contract.blockedPaths()
	var issues []string
	for _, path := range ordered {
		path = normalizeRepoPath(path)
		if path == "" || isKKTPath(path) {
			continue
		}
		currentFingerprint, fpErr := pathFingerprint(projectRootDir, path)
		if fpErr != nil {
			issues = append(issues, "could not fingerprint path: "+path+": "+fpErr.Error())
			continue
		}
		if baselineFingerprint, ok := state.Baseline.Paths[path]; ok && baselineFingerprint == currentFingerprint {
			continue
		}
		if matchesAnyPathPattern(path, blocked) {
			issues = append(issues, "changed blocked path after hook baseline: "+path)
			continue
		}
		if len(allowed) > 0 && !matchesAnyPathPattern(path, allowed) {
			issues = append(issues, "changed out-of-scope path after hook baseline: "+path)
		}
	}
	return issues
}

func currentHookBaseline(projectRootDir string) (HookBaseline, error) {
	changed, err := changedGitPaths(projectRootDir)
	if err != nil {
		return HookBaseline{}, err
	}
	paths := map[string]string{}
	for _, path := range changed {
		path = normalizeRepoPath(path)
		if path == "" || isKKTPath(path) {
			continue
		}
		fingerprint, err := pathFingerprint(projectRootDir, path)
		if err != nil {
			return HookBaseline{}, err
		}
		paths[path] = fingerprint
	}
	return HookBaseline{RecordedAt: time.Now().UTC().Format(time.RFC3339), Paths: paths}, nil
}

func normalizeHookPayload(agent, event, payload string) normalizedHookPayload {
	normalized := normalizedHookPayload{Agent: agent, Event: event}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return normalized
	}
	raw := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return normalized
	}
	normalized.CWD = firstString(raw, "cwd", "directory", "worktree", "project_root")
	normalized.ToolName = firstString(raw, "tool_name", "toolName", "tool", "name")
	normalized.Command = firstString(raw, "command", "cmd")
	normalized.Paths = append(normalized.Paths, stringList(raw["paths"])...)
	normalized.Paths = appendStringIfNotEmpty(normalized.Paths, firstString(raw, "path", "file_path", "filePath", "target_file", "targetFile"))

	for _, key := range []string{"tool_input", "toolInput", "input", "args", "arguments"} {
		child, ok := raw[key].(map[string]any)
		if !ok {
			continue
		}
		if normalized.ToolName == "" {
			normalized.ToolName = firstString(child, "tool_name", "toolName", "tool", "name")
		}
		if normalized.Command == "" {
			normalized.Command = firstString(child, "command", "cmd")
		}
		if normalized.CWD == "" {
			normalized.CWD = firstString(child, "cwd", "directory", "worktree")
		}
		normalized.Paths = append(normalized.Paths, stringList(child["paths"])...)
		normalized.Paths = appendStringIfNotEmpty(normalized.Paths, firstString(child, "path", "file_path", "filePath", "target_file", "targetFile"))
	}
	if strings.Contains(strings.ToLower(normalized.ToolName), "patch") || strings.Contains(normalized.Command, "*** Begin Patch") || strings.Contains(normalized.Command, "diff --git ") {
		normalized.Paths = append(normalized.Paths, patchPathsFromCommand(normalized.Command)...)
	}
	normalized.Paths = uniqueStrings(normalized.Paths)
	return normalized
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		var values []string
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				values = append(values, strings.TrimSpace(text))
			}
		}
		return values
	default:
		return nil
	}
}

func appendStringIfNotEmpty(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	return append(values, strings.TrimSpace(value))
}

func normalizeHookPath(projectRootDir, cwd, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	var absolute string
	if filepath.IsAbs(path) {
		absolute = filepath.Clean(path)
	} else if cwd != "" && filepath.IsAbs(cwd) {
		absolute = filepath.Clean(filepath.Join(cwd, path))
	} else {
		absolute = filepath.Clean(filepath.Join(projectRootDir, path))
	}
	if rel, err := filepath.Rel(projectRootDir, absolute); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		return normalizeRepoPath(rel)
	}
	if filepath.IsAbs(path) {
		return normalizeRepoPath(path)
	}
	return normalizeRepoPath(path)
}

func patchPathsFromCommand(command string) []string {
	var paths []string
	for _, line := range strings.Split(command, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"*** Update File:", "*** Add File:", "*** Delete File:", "*** Move to:"} {
			if strings.HasPrefix(line, prefix) {
				paths = appendStringIfNotEmpty(paths, cleanPatchPath(strings.TrimSpace(strings.TrimPrefix(line, prefix))))
			}
		}
		if strings.HasPrefix(line, "diff --git ") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				paths = appendStringIfNotEmpty(paths, cleanPatchPath(fields[2]))
				paths = appendStringIfNotEmpty(paths, cleanPatchPath(fields[3]))
			}
			continue
		}
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] != "/dev/null" {
				paths = appendStringIfNotEmpty(paths, cleanPatchPath(fields[1]))
			}
		}
	}
	return uniqueStrings(paths)
}

func cleanPatchPath(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "\"'")
	if strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/") {
		path = path[2:]
	}
	return path
}

func isDirectMutationTool(toolName string) bool {
	name := strings.ToLower(strings.TrimSpace(toolName))
	if name == "" {
		return false
	}
	return strings.Contains(name, "edit") || strings.Contains(name, "write") || strings.Contains(name, "patch")
}

func isShellTool(toolName string) bool {
	name := strings.ToLower(strings.TrimSpace(toolName))
	return name == "bash" || name == "powershell" || name == "pwsh" || name == "shell" || name == "exec" || strings.Contains(name, "terminal")
}

func looksMutatingShellCommand(command string) bool {
	command = strings.ToLower(command)
	mutatingNeedles := []string{"rm ", "rm\t", "mv ", "cp ", "touch ", "mkdir ", "rmdir ", "sed -i", "perl -pi", "truncate ", "tee ", ">", ">>", "git reset", "git clean", "git checkout", "git restore"}
	for _, needle := range mutatingNeedles {
		if strings.Contains(command, needle) {
			return true
		}
	}
	return false
}

func commandMentionsBlockedPath(command string, patterns []string) bool {
	command = strings.ToLower(command)
	for _, pattern := range patterns {
		needle := blockedPatternNeedle(pattern)
		if needle != "" && strings.Contains(command, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func blockedPatternNeedle(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	cut := len(pattern)
	for _, marker := range []string{"*", "?", "[", "{"} {
		if index := strings.Index(pattern, marker); index >= 0 && index < cut {
			cut = index
		}
	}
	pattern = strings.TrimRight(pattern[:cut], "/")
	if pattern == "" || pattern == "." {
		return ""
	}
	return pattern
}

func validHookEvent(event string) bool {
	switch event {
	case "pre-tool", "post-tool", "pre-compact", "post-compact", "stop":
		return true
	default:
		return false
	}
}

func hookResult(event, mode, verdict, reason string, repair, evidence []string, workspace, workspaceType string) HookResult {
	return HookResult{
		SchemaVersion: 1,
		Verdict:       verdict,
		Mode:          mode,
		Event:         event,
		Reason:        reason,
		Workspace:     workspace,
		WorkspaceType: workspaceType,
		Repair:        repair,
		Evidence:      evidence,
	}
}

func activeHookResult(state HookState, event, verdict, reason string, repair, evidence []string) HookResult {
	mode := state.Mode
	if mode == "" {
		mode = "enforce"
	}
	if mode == "observe" && verdict == "block" {
		verdict = "warn"
	}
	return hookResult(event, mode, verdict, reason, repair, evidence, state.Workspace, state.WorkspaceType)
}

func writeAgentHookResult(stdout io.Writer, event string, result HookResult) error {
	if result.Verdict == "warn" {
		return writeAgentHookWarning(stdout, event, result)
	}
	if result.Verdict != "block" {
		return nil
	}
	hookEventName := hookEventName(event)
	if event == "pre-tool" {
		payload := map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":            hookEventName,
				"permissionDecision":       "deny",
				"permissionDecisionReason": result.Reason,
				"additionalContext":        strings.Join(result.Evidence, "\n"),
			},
		}
		return writeJSON(stdout, payload)
	}
	payload := map[string]any{"decision": "block", "reason": result.Reason}
	if hookEventName != "" {
		payload["hookSpecificOutput"] = map[string]any{
			"hookEventName":     hookEventName,
			"additionalContext": strings.Join(result.Evidence, "\n"),
		}
	}
	return writeJSON(stdout, payload)
}

func writeAgentHookWarning(stdout io.Writer, event string, result HookResult) error {
	hookEventName := hookEventName(event)
	payload := map[string]any{
		"systemMessage": "KKT hook warning: " + result.Reason,
	}
	if hookEventName != "" {
		payload["hookSpecificOutput"] = map[string]any{
			"hookEventName":     hookEventName,
			"additionalContext": strings.Join(result.Evidence, "\n"),
		}
	}
	return writeJSON(stdout, payload)
}

func hookEventName(event string) string {
	switch event {
	case "pre-tool":
		return "PreToolUse"
	case "post-tool":
		return "PostToolUse"
	case "pre-compact":
		return "PreCompact"
	case "post-compact":
		return "PostCompact"
	case "stop":
		return "Stop"
	default:
		return ""
	}
}

func hookStatePath(projectRootDir string) string {
	return filepath.Join(projectRootDir, ".kkt", "hooks.json")
}

func readHookState(projectRootDir string) (HookState, bool, error) {
	path := hookStatePath(projectRootDir)
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return HookState{SchemaVersion: 1, Armed: false, Mode: "inactive"}, false, nil
		}
		return HookState{}, false, err
	}
	var state HookState
	if err := json.Unmarshal(payload, &state); err != nil {
		return HookState{}, true, err
	}
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	if state.Mode == "" {
		if state.Armed {
			state.Mode = "enforce"
		} else {
			state.Mode = "inactive"
		}
	}
	if state.Baseline.Paths == nil {
		state.Baseline.Paths = map[string]string{}
	}
	return state, true, nil
}

func writeHookState(projectRootDir string, state HookState) error {
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	if state.Baseline.Paths == nil {
		state.Baseline.Paths = map[string]string{}
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := hookStatePath(projectRootDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func disarmHooks(projectRootDir string) error {
	err := os.Remove(hookStatePath(projectRootDir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func disarmHooksForWorkspace(workspace string) error {
	projectRootDir, err := projectRootForWorkspace(".", workspace)
	if err != nil {
		return err
	}
	state, exists, err := readHookState(projectRootDir)
	if err != nil || !exists || !state.Armed {
		return err
	}
	workspaceSource := workspaceSourcePath(projectRootDir, workspace)
	if state.Workspace != "" && state.Workspace != workspaceSource {
		return nil
	}
	return disarmHooks(projectRootDir)
}

func isHookExpired(state HookState) bool {
	if strings.TrimSpace(state.ExpiresAt) == "" {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, state.ExpiresAt)
	if err != nil {
		return true
	}
	return time.Now().UTC().After(expiresAt)
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil {
		left = leftAbs
	}
	if rightErr == nil {
		right = rightAbs
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func errorEvidence(err error) []string {
	if err == nil {
		return nil
	}
	return []string{err.Error()}
}

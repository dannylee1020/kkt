package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type validationCommandProof struct {
	Command     string
	Status      string
	ExitCode    int
	DurationMS  int64
	Timestamp   string
	Log         string
	Fingerprint string
}

func RunRequiredValidationCommands(root, workspace string, timeout time.Duration, stdout io.Writer) error {
	commands, err := requiredValidationCommands(workspace)
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		fmt.Fprintln(stdout, "no required validation commands")
		return nil
	}
	projectRootDir, err := projectRootForWorkspace(root, workspace)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(workspace, "validation"), 0o755); err != nil {
		return err
	}

	var failed []string
	for _, command := range commands {
		proof, runErr := runValidationCommand(projectRootDir, workspace, command, timeout)
		if proof.Command == "" {
			return runErr
		}
		eventErr := appendValidationCommandEvent(workspace, proof)
		evidenceErr := appendValidationCommandEvidence(workspace, proof)
		fmt.Fprintf(stdout, "%s: %s\n", proof.Status, command)
		fmt.Fprintf(stdout, "log: %s\n", proof.Log)
		if runErr != nil {
			failed = append(failed, command)
		}
		if eventErr != nil {
			return eventErr
		}
		if evidenceErr != nil {
			return evidenceErr
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("validation command failed: %s", strings.Join(failed, ", "))
	}
	return markEvidenceRecorded(workspace)
}

func runValidationCommand(projectRootDir, workspace, command string, timeout time.Duration) (validationCommandProof, error) {
	start := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var output bytes.Buffer
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Dir = projectRootDir
	cmd.Stdout = &output
	cmd.Stderr = &output
	runErr := cmd.Run()

	exitCode := 0
	status := "passed"
	if runErr != nil {
		status = "failed"
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		if ctx.Err() == context.DeadlineExceeded {
			output.WriteString(fmt.Sprintf("\ncommand timed out after %s\n", timeout))
		}
	}

	timestamp := start.Format("20060102-150405")
	logPath := filepath.Join(workspace, "validation", fmt.Sprintf("%s-%s.log", timestamp, slugify(command)))
	if err := os.WriteFile(logPath, output.Bytes(), 0o644); err != nil {
		return validationCommandProof{}, err
	}
	fingerprint, err := workingTreeFingerprint(projectRootDir)
	if err != nil {
		return validationCommandProof{}, err
	}
	proof := validationCommandProof{
		Command:     command,
		Status:      status,
		ExitCode:    exitCode,
		DurationMS:  time.Since(start).Milliseconds(),
		Timestamp:   start.Format(time.RFC3339),
		Log:         logPath,
		Fingerprint: fingerprint,
	}
	return proof, runErr
}

func requiredValidationCommands(workspace string) ([]string, error) {
	contract, err := readGuardrails(workspace)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return uniqueNonEmpty(contract.Validation.RequiredCommands), nil
}

func validationCommandProofIssues(workspace string) []string {
	commands, err := requiredValidationCommands(workspace)
	if err != nil {
		return []string{"guardrails.json could not be read: " + err.Error()}
	}
	if len(commands) == 0 {
		return nil
	}
	projectRootDir, err := projectRootForWorkspace(".", workspace)
	if err != nil {
		return []string{"could not resolve project root for validation proof: " + err.Error()}
	}
	fingerprint, err := workingTreeFingerprint(projectRootDir)
	if err != nil {
		return []string{"could not fingerprint working tree for validation proof: " + err.Error()}
	}
	proofs, err := latestValidationCommandProofs(workspace)
	if err != nil {
		return []string{"could not read validation command proof: " + err.Error()}
	}
	var issues []string
	for _, command := range commands {
		proof, ok := proofs[command]
		if !ok {
			issues = append(issues, "required command not run: "+command)
			continue
		}
		if proof.Status != "passed" || proof.ExitCode != 0 {
			issues = append(issues, "required command failed: "+command)
			continue
		}
		if proof.Fingerprint != fingerprint {
			issues = append(issues, "required command proof is stale: "+command)
		}
	}
	return issues
}

func latestValidationCommandProofs(workspace string) (map[string]validationCommandProof, error) {
	events, err := readEvents(workspace, 0)
	if err != nil {
		return nil, err
	}
	proofs := map[string]validationCommandProof{}
	for _, event := range events {
		if event.Type != "validation_command_passed" && event.Type != "validation_command_failed" {
			continue
		}
		proof := validationCommandProof{
			Command:     eventDataString(event.Data, "command"),
			Status:      eventDataString(event.Data, "status"),
			ExitCode:    eventDataInt(event.Data, "exit_code"),
			DurationMS:  int64(eventDataInt(event.Data, "duration_ms")),
			Timestamp:   eventDataString(event.Data, "timestamp"),
			Log:         eventDataString(event.Data, "log"),
			Fingerprint: eventDataString(event.Data, "fingerprint"),
		}
		if proof.Command != "" {
			proofs[proof.Command] = proof
		}
	}
	return proofs, nil
}

func appendValidationCommandEvent(workspace string, proof validationCommandProof) error {
	state, err := ReadState(workspace)
	if err != nil {
		return err
	}
	eventType := "validation_command_passed"
	if proof.Status != "passed" || proof.ExitCode != 0 {
		eventType = "validation_command_failed"
	}
	return appendWorkspaceEvent(workspace, state, eventType, map[string]any{
		"command":     proof.Command,
		"status":      proof.Status,
		"exit_code":   proof.ExitCode,
		"duration_ms": proof.DurationMS,
		"timestamp":   proof.Timestamp,
		"log":         proof.Log,
		"fingerprint": proof.Fingerprint,
	})
}

func appendValidationCommandEvidence(workspace string, proof validationCommandProof) error {
	file, err := os.OpenFile(filepath.Join(workspace, "evidence.md"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "\n## Validation Command\n\n- command: `%s`\n- status: %s\n- exit_code: %d\n- duration_ms: %d\n- log: %s\n- timestamp: %s\n", proof.Command, proof.Status, proof.ExitCode, proof.DurationMS, proof.Log, proof.Timestamp)
	return err
}

func markEvidenceRecorded(workspace string) error {
	path := filepath.Join(workspace, "evidence.md")
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

func workingTreeFingerprint(projectRootDir string) (string, error) {
	changed, err := changedGitPaths(projectRootDir)
	if err != nil {
		return "", err
	}
	var entries []string
	for _, path := range changed {
		if path == ".kkt" || strings.HasPrefix(path, ".kkt/") {
			continue
		}
		fullPath := filepath.Join(projectRootDir, filepath.FromSlash(path))
		payload, readErr := os.ReadFile(fullPath)
		switch {
		case readErr == nil:
			sum := sha256.Sum256(payload)
			entries = append(entries, path+"="+hex.EncodeToString(sum[:]))
		case os.IsNotExist(readErr):
			entries = append(entries, path+"=<deleted>")
		default:
			return "", readErr
		}
	}
	sort.Strings(entries)
	sum := sha256.Sum256([]byte(strings.Join(entries, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func eventDataString(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

func eventDataInt(data map[string]any, key string) int {
	switch value := data[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

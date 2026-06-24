package workflow

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPrintsVersion(t *testing.T) {
	previous := Version
	Version = "vtest"
	defer func() {
		Version = previous
	}()

	var stdout bytes.Buffer
	if err := Run([]string{"--version"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "kkt vtest"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRunRejectsRemovedAliasesAndFlags(t *testing.T) {
	tests := [][]string{
		{"-h"},
		{"-v"},
		{"version"},
		{"classify", "implement a feature"},
		{"start", "--profile", "plan", "implement a feature"},
		{"init", "codex"},
		{"uninstall", "--dry-run"},
		{"uninstall", "--keep-binary"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			if err := Run(test, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
				t.Fatal("expected removed alias or flag to be rejected")
			}
		})
	}
}

func TestRunStartRequiresExplicitProfile(t *testing.T) {
	var stdout bytes.Buffer
	err := Run([]string{"start", "implement", "a", "feature"}, &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected start without explicit profile to fail")
	}
	if !strings.Contains(err.Error(), "unsupported profile") {
		t.Fatalf("error = %q, want unsupported profile", err.Error())
	}
}

func TestRunRejectsDailyProfile(t *testing.T) {
	err := Run([]string{"start", "daily", "implement", "a", "feature"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected daily profile to be rejected")
	}
	if !strings.Contains(err.Error(), "unsupported profile") {
		t.Fatalf("error = %q, want unsupported profile", err.Error())
	}
}

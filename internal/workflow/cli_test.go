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
		{"classify", "--json", "implement a feature"},
		{"start", "--profile", "daily", "implement a feature"},
		{"init", "--dry-run", "codex"},
		{"init", "--command", "/tmp/kkt", "codex"},
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

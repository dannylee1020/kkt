package workflow

import "testing"

func TestClassifyInvokesForImplementation(t *testing.T) {
	result := Classify("implement a Go CLI for KKT workflow")
	if result.Decision != "invoke" {
		t.Fatalf("Decision = %q, want invoke", result.Decision)
	}
	if result.Profile != "daily" {
		t.Fatalf("Profile = %q, want daily", result.Profile)
	}
	if result.NextCommand == "" {
		t.Fatal("NextCommand should be set")
	}
}

func TestClassifyLoopProfile(t *testing.T) {
	result := Classify("build a multi-step migration workflow")
	if result.Decision != "invoke" {
		t.Fatalf("Decision = %q, want invoke", result.Decision)
	}
	if result.Profile != "loop" {
		t.Fatalf("Profile = %q, want loop", result.Profile)
	}
}

func TestClassifySkipsInformationalRequest(t *testing.T) {
	result := Classify("explain what this file does")
	if result.Decision != "skip" {
		t.Fatalf("Decision = %q, want skip", result.Decision)
	}
}

func TestClassifyUsesProvidedCommand(t *testing.T) {
	result := ClassifyWithCommand("implement a CLI feature", "/tmp/kkt")
	if result.NextCommand == "" {
		t.Fatal("NextCommand should be set")
	}
	if got, want := result.NextCommand[:len("/tmp/kkt ")], "/tmp/kkt "; got != want {
		t.Fatalf("NextCommand prefix = %q, want %q", got, want)
	}
}

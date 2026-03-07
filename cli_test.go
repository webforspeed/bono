package main

import "testing"

func TestParseCLIArgsHeadlessPrompt(t *testing.T) {
	opts, err := parseCLIArgs([]string{"-p", "Find and fix the bug"})
	if err != nil {
		t.Fatalf("parseCLIArgs returned error: %v", err)
	}
	if opts.Prompt != "Find and fix the bug" {
		t.Fatalf("Prompt = %q, want %q", opts.Prompt, "Find and fix the bug")
	}
	if !opts.Headless() {
		t.Fatalf("Headless() = false, want true")
	}
}

func TestParseCLIArgsRejectsUnexpectedArgs(t *testing.T) {
	if _, err := parseCLIArgs([]string{"extra"}); err == nil {
		t.Fatalf("parseCLIArgs error = nil, want non-nil")
	}
}

func TestParseCLIArgsEmptyPromptIsNotHeadless(t *testing.T) {
	opts, err := parseCLIArgs([]string{"--prompt", ""})
	if err != nil {
		t.Fatalf("parseCLIArgs returned error: %v", err)
	}
	if opts.Headless() {
		t.Fatalf("Headless() = true, want false")
	}
}

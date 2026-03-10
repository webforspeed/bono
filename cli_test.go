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

func TestParseCLIArgsSkipApprovals(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "default false",
			args: []string{},
			want: false,
		},
		{
			name: "flag enabled",
			args: []string{"--skip-approvals"},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := parseCLIArgs(tc.args)
			if err != nil {
				t.Fatalf("parseCLIArgs returned error: %v", err)
			}
			if opts.SkipApprovals != tc.want {
				t.Fatalf("SkipApprovals = %v, want %v", opts.SkipApprovals, tc.want)
			}
		})
	}
}

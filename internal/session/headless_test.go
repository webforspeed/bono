package session

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHeadlessFrontendApprovesToolPrompt(t *testing.T) {
	var out bytes.Buffer
	frontend := NewHeadlessFrontend(&out, strings.NewReader("y\n"))

	ok := frontend.RequestApproval(context.Background(), ApprovalRequest{
		Kind:     ApprovalTool,
		ToolName: "run_shell",
		ToolArgs: map[string]any{
			"command":     "go test ./...",
			"description": "run tests",
			"safety":      "modify",
		},
	})
	if !ok {
		t.Fatalf("RequestApproval returned false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "● Bash('go test ./...') # run tests, modify") {
		t.Fatalf("output %q missing formatted tool line", output)
	}
	if !strings.Contains(output, "  ↳ Approve? [y/N]: ") {
		t.Fatalf("output %q missing approval prompt", output)
	}
}

func TestHeadlessFrontendRejectsEOFApproval(t *testing.T) {
	var out bytes.Buffer
	frontend := NewHeadlessFrontend(&out, strings.NewReader(""))

	ok := frontend.RequestApproval(context.Background(), ApprovalRequest{
		Kind:        ApprovalChangeBatch,
		ChangeCount: 2,
	})
	if ok {
		t.Fatalf("RequestApproval returned true, want false")
	}

	output := out.String()
	if !strings.Contains(output, "● Approve 2 changes or Undo => undone") {
		t.Fatalf("output %q missing rejected batch status", output)
	}
}

func TestHeadlessFrontendFinalizesStreamingBeforeToolLines(t *testing.T) {
	var out bytes.Buffer
	frontend := NewHeadlessFrontend(&out, strings.NewReader(""))
	ctx := context.Background()

	frontend.HandleEvent(ctx, ContentDeltaEvent{Delta: "partial response"})
	frontend.HandleEvent(ctx, ToolCallEvent{
		Name: "read_file",
		Args: map[string]any{"path": "auth.py"},
	})
	frontend.HandleEvent(ctx, ToolDoneEvent{
		Name:   "read_file",
		Args:   map[string]any{"path": "auth.py"},
		Status: "success",
	})

	output := out.String()
	contentIdx := strings.Index(output, "partial response")
	callIdx := strings.Index(output, "● Read('auth.py')")
	doneIdx := strings.LastIndex(output, "● Read('auth.py') => success")
	if contentIdx == -1 || callIdx == -1 || doneIdx == -1 {
		t.Fatalf("unexpected output: %q", output)
	}
	if !(contentIdx < callIdx && callIdx < doneIdx) {
		t.Fatalf("expected content before tool call before tool done, got %q", output)
	}
}

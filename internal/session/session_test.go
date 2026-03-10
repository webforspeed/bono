package session

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/webforspeed/bono-core"
	"github.com/webforspeed/bono/hooks"
	"github.com/webforspeed/bono/internal/changebatch"
)

func TestStopHandlerUndoesRejectedBatch(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(cwd, "notes.txt")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	frontend := NewHeadlessFrontend(&out, strings.NewReader("n\n"))
	sess := &Session{
		dispatcher:     hooks.NewDispatcher(),
		frontend:       frontend,
		config:         Config{CWD: cwd},
		changeBatchMgr: changebatch.NewManager(),
	}
	if _, err := sess.changeBatchMgr.BeginChange(cwd, "edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := sess.changeBatchMgr.CompleteChange("edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatalf("CompleteChange ok = false, want true")
	}

	sess.StopHandler().Handle(context.Background(), hooks.Stop, hooks.StopPayload{})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before\n" {
		t.Fatalf("file contents = %q, want %q", string(data), "before\n")
	}
	if !strings.Contains(out.String(), "● Approve 1 change or Undo => undone") {
		t.Fatalf("output %q missing undo status", out.String())
	}
}

func TestBindSkipApprovalsBypassesToolApproval(t *testing.T) {
	frontend := &mockFrontend{
		approvalResult: false,
	}
	sess := &Session{
		agent:          &core.Agent{},
		dispatcher:     hooks.NewDispatcher(),
		frontend:       frontend,
		config:         Config{SkipApprovals: true},
		changeBatchMgr: changebatch.NewManager(),
	}
	sess.Bind(context.Background())

	ok := sess.agent.OnToolCall("danger_tool", map[string]any{"k": "v"})
	if !ok {
		t.Fatalf("OnToolCall returned false, want true")
	}
	if frontend.requestApprovalCount != 0 {
		t.Fatalf("RequestApproval called %d times, want 0", frontend.requestApprovalCount)
	}
	if len(frontend.events) == 0 {
		t.Fatalf("expected at least one event")
	}
	if _, isToolCall := frontend.events[0].(ToolCallEvent); !isToolCall {
		t.Fatalf("first event type = %T, want ToolCallEvent", frontend.events[0])
	}
}

func TestStopHandlerSkipApprovalsKeepsBatch(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(cwd, "notes.txt")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	frontend := &mockFrontend{
		approvalResult: false,
	}
	sess := &Session{
		dispatcher:     hooks.NewDispatcher(),
		frontend:       frontend,
		config:         Config{CWD: cwd, SkipApprovals: true},
		changeBatchMgr: changebatch.NewManager(),
	}
	if _, err := sess.changeBatchMgr.BeginChange(cwd, "edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := sess.changeBatchMgr.CompleteChange("edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatalf("CompleteChange ok = false, want true")
	}

	sess.StopHandler().Handle(context.Background(), hooks.Stop, hooks.StopPayload{})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "after\n" {
		t.Fatalf("file contents = %q, want %q", string(data), "after\n")
	}
	if frontend.requestApprovalCount != 0 {
		t.Fatalf("RequestApproval called %d times, want 0", frontend.requestApprovalCount)
	}
}

type mockFrontend struct {
	approvalResult       bool
	requestApprovalCount int
	events               []Event
}

func (m *mockFrontend) HandleEvent(_ context.Context, event Event) {
	m.events = append(m.events, event)
}

func (m *mockFrontend) RequestApproval(_ context.Context, _ ApprovalRequest) bool {
	m.requestApprovalCount++
	return m.approvalResult
}

func (m *mockFrontend) RequestSubAgentApproval(_ context.Context, _ core.SubAgentResult) core.SubAgentApprovalResponse {
	return core.SubAgentApprovalResponse{Action: core.SubAgentApprove}
}

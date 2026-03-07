package session

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

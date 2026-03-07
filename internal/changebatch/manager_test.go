package changebatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBeginChangeAndCompleteChange(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(cwd, "notes.txt")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager()
	change, err := mgr.BeginChange(cwd, "edit_file", "notes.txt")
	if err != nil {
		t.Fatalf("BeginChange returned error: %v", err)
	}
	if change.BeforeContent != "before" {
		t.Fatalf("BeforeContent = %q, want %q", change.BeforeContent, "before")
	}

	if err := os.WriteFile(path, []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}

	change, ok, err := mgr.CompleteChange("edit_file", "notes.txt")
	if err != nil {
		t.Fatalf("CompleteChange returned error: %v", err)
	}
	if !ok {
		t.Fatalf("CompleteChange ok = false, want true")
	}
	if change.AfterContent != "after" {
		t.Fatalf("AfterContent = %q, want %q", change.AfterContent, "after")
	}
}

func TestDrainCompletedKeepsFirstBeforeAndLastAfter(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(cwd, "notes.txt")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager()
	if _, err := mgr.BeginChange(cwd, "edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := mgr.CompleteChange("edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	}

	if _, err := mgr.BeginChange(cwd, "edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("three"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := mgr.CompleteChange("edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	}

	changes := mgr.DrainCompleted()
	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}
	if changes[0].BeforeContent != "one" {
		t.Fatalf("BeforeContent = %q, want %q", changes[0].BeforeContent, "one")
	}
	if changes[0].AfterContent != "three" {
		t.Fatalf("AfterContent = %q, want %q", changes[0].AfterContent, "three")
	}
}

func TestUndoBatchRestoresExistingFile(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(cwd, "notes.txt")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager()
	change, err := mgr.BeginChange(cwd, "edit_file", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}
	change, _, err = mgr.CompleteChange("edit_file", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.UndoBatch([]FileChange{change}); err != nil {
		t.Fatalf("UndoBatch returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before" {
		t.Fatalf("file contents = %q, want %q", string(data), "before")
	}
}

func TestUndoBatchDeletesNewFile(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(cwd, "new.txt")

	mgr := NewManager()
	change, err := mgr.BeginChange(cwd, "write_file", "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !change.WasNewFile {
		t.Fatalf("WasNewFile = false, want true")
	}
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	change, _, err = mgr.CompleteChange("write_file", "new.txt")
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.UndoBatch([]FileChange{change}); err != nil {
		t.Fatalf("UndoBatch returned error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat err = %v", err)
	}
}

func TestResetClearsState(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(cwd, "notes.txt")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager()
	if _, err := mgr.BeginChange(cwd, "edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	}
	mgr.Reset()
	if changes := mgr.DrainCompleted(); len(changes) != 0 {
		t.Fatalf("len(changes) = %d, want 0", len(changes))
	}
	if _, ok, err := mgr.CompleteChange("edit_file", "notes.txt"); err != nil || ok {
		t.Fatalf("CompleteChange after Reset = (_, %v, %v), want (_, false, nil)", ok, err)
	}
}

func TestDiscardChangeRemovesPendingEntry(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(cwd, "notes.txt")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager()
	if _, err := mgr.BeginChange(cwd, "edit_file", "notes.txt"); err != nil {
		t.Fatal(err)
	}
	if ok := mgr.DiscardChange("edit_file", "notes.txt"); !ok {
		t.Fatalf("DiscardChange ok = false, want true")
	}
	if _, ok, err := mgr.CompleteChange("edit_file", "notes.txt"); err != nil || ok {
		t.Fatalf("CompleteChange after DiscardChange = (_, %v, %v), want (_, false, nil)", ok, err)
	}
}

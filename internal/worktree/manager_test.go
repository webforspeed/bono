package worktree

import (
	"path/filepath"
	"testing"
)

func TestNormalizeRepoPath_Relative(t *testing.T) {
	repo := t.TempDir()
	rel, err := NormalizeRepoPath(repo, "foo/bar.go")
	if err != nil {
		t.Fatalf("NormalizeRepoPath returned error: %v", err)
	}
	if rel != "foo/bar.go" {
		t.Fatalf("rel = %q, want %q", rel, "foo/bar.go")
	}
}

func TestNormalizeRepoPath_AbsoluteInsideRepo(t *testing.T) {
	repo := t.TempDir()
	abs := filepath.Join(repo, "pkg", "x.go")
	rel, err := NormalizeRepoPath(repo, abs)
	if err != nil {
		t.Fatalf("NormalizeRepoPath returned error: %v", err)
	}
	if rel != "pkg/x.go" {
		t.Fatalf("rel = %q, want %q", rel, "pkg/x.go")
	}
}

func TestNormalizeRepoPath_OutsideRepoRejected(t *testing.T) {
	repo := t.TempDir()
	outside := filepath.Join(filepath.Dir(repo), "outside.go")
	if _, err := NormalizeRepoPath(repo, outside); err == nil {
		t.Fatalf("expected error for outside-repo path, got nil")
	}
}

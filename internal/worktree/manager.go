package worktree

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Session struct {
	RepoRoot     string
	BranchName   string
	WorktreeRoot string
	ID           string
	CreatedAt    time.Time
}

type PathRewrite struct {
	ToolName      string
	OriginalPath  string
	RelPath       string
	RewrittenAbs  string
	WasNewFile    bool
	BeforeContent string
	AfterContent  string // post-write content captured in OnToolDone
	RepoRoot      string // repo root this rewrite belongs to (for multi-repo support)
}

type Manager struct {
	mu                sync.Mutex
	sessions          map[string]*Session // keyed by repo root
	rewrites          map[string][]PathRewrite
	completedRewrites []PathRewrite   // accumulated for batch approval after loop
	completedSet      map[string]bool // dedup by RepoRoot|RelPath
	syncedFiles       map[string]bool // tracks files synced from working tree to worktree (keyed by repoRoot|relPath)
}

func NewManager() *Manager {
	return &Manager{
		sessions:     make(map[string]*Session),
		rewrites:     make(map[string][]PathRewrite),
		completedSet: make(map[string]bool),
		syncedFiles:  make(map[string]bool),
	}
}

// EnsureSession returns (or creates) a worktree session for the repo that
// contains filePath. The cwd is used to resolve relative file paths.
func (m *Manager) EnsureSession(ctx context.Context, cwd, filePath string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	repoRoot, err := resolveRepoRoot(ctx, cwd, filePath)
	if err != nil {
		return nil, err
	}

	// Return existing session if still valid.
	if s, ok := m.sessions[repoRoot]; ok {
		if _, err := os.Stat(s.WorktreeRoot); err == nil {
			return s, nil
		}
		delete(m.sessions, repoRoot) // stale
	}

	id, err := randomWorktreeID()
	if err != nil {
		return nil, err
	}
	branch := "bono/" + id
	worktreeRoot := filepath.Join(repoRoot, ".bono", "worktrees", id)
	if err := os.MkdirAll(filepath.Dir(worktreeRoot), 0o755); err != nil {
		return nil, fmt.Errorf("create worktree parent dir: %w", err)
	}

	if _, err := GitOutput(ctx, repoRoot, "worktree", "add", "-b", branch, worktreeRoot, "HEAD"); err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	s := &Session{
		RepoRoot:     repoRoot,
		BranchName:   branch,
		WorktreeRoot: worktreeRoot,
		ID:           id,
		CreatedAt:    time.Now(),
	}
	m.sessions[repoRoot] = s
	return s, nil
}

// resolveRepoRoot finds the git repo root for a file path.
func resolveRepoRoot(ctx context.Context, cwd, filePath string) (string, error) {
	var abs string
	if filepath.IsAbs(filePath) {
		abs = filepath.Clean(filePath)
	} else {
		abs = filepath.Clean(filepath.Join(cwd, filePath))
	}

	// Walk up from the file's directory to find an existing directory.
	dir := filepath.Dir(abs)
	for {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no existing parent directory for %s", filePath)
		}
		dir = parent
	}

	root, err := GitOutput(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve repo root for %s: %w", filePath, err)
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("empty repo root for %s", filePath)
	}
	return root, nil
}

func (m *Manager) RewritePathForWorktree(session *Session, inputPath string) (relPath, rewrittenAbs string, err error) {
	relPath, err = NormalizeRepoPath(session.RepoRoot, inputPath)
	if err != nil {
		return "", "", err
	}
	rewrittenAbs = filepath.Join(session.WorktreeRoot, filepath.FromSlash(relPath))
	return relPath, rewrittenAbs, nil
}

func (m *Manager) RegisterRewrite(meta PathRewrite) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := rewriteKey(meta.ToolName, meta.RewrittenAbs)
	m.rewrites[k] = append(m.rewrites[k], meta)
}

// SyncToWorktree copies the working tree version of a file into the worktree
// on first access. git worktree checks out at HEAD, so uncommitted changes
// would be missing — causing edit_file to fail with "string not found".
// Returns true if a sync was performed, false if already synced or new file.
func (m *Manager) SyncToWorktree(session *Session, relPath string, wasNewFile bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := session.RepoRoot + "|" + relPath
	if m.syncedFiles[key] {
		return false // already synced or written
	}
	m.syncedFiles[key] = true

	if wasNewFile {
		return false // nothing to sync for new files
	}

	src := filepath.Join(session.RepoRoot, filepath.FromSlash(relPath))
	dst := filepath.Join(session.WorktreeRoot, filepath.FromSlash(relPath))

	data, err := os.ReadFile(src)
	if err != nil {
		return false
	}
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, data, 0o644)
	return true
}

// Sessions returns all active worktree sessions (for cleanup, hooks, etc.).
func (m *Manager) Sessions() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

func (m *Manager) ConsumeRewrite(toolName, rewrittenAbs string) (PathRewrite, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := rewriteKey(toolName, rewrittenAbs)
	q := m.rewrites[k]
	if len(q) == 0 {
		return PathRewrite{}, false
	}
	item := q[0]
	if len(q) == 1 {
		delete(m.rewrites, k)
	} else {
		m.rewrites[k] = q[1:]
	}
	return item, true
}

// RecordCompleted stores a completed rewrite for batch approval after the loop.
// Deduped by RepoRoot+RelPath: first BeforeContent wins, latest AfterContent overwrites.
func (m *Manager) RecordCompleted(meta PathRewrite) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := meta.RepoRoot + "|" + meta.RelPath
	if m.completedSet[key] {
		// Update AfterContent for existing entry (same file edited again).
		for i := range m.completedRewrites {
			if m.completedRewrites[i].RepoRoot == meta.RepoRoot && m.completedRewrites[i].RelPath == meta.RelPath {
				m.completedRewrites[i].AfterContent = meta.AfterContent
				return
			}
		}
		return
	}
	m.completedSet[key] = true
	m.completedRewrites = append(m.completedRewrites, meta)
}

// RemoveAllSessions removes all worktrees and branches, clearing all state.
func (m *Manager) RemoveAllSessions(ctx context.Context) {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = make(map[string]*Session)
	m.rewrites = make(map[string][]PathRewrite)
	m.completedRewrites = nil
	m.completedSet = make(map[string]bool)
	m.syncedFiles = make(map[string]bool)
	m.mu.Unlock()

	for _, s := range sessions {
		GitOutput(ctx, s.RepoRoot, "worktree", "remove", "--force", s.WorktreeRoot)
		GitOutput(ctx, s.RepoRoot, "branch", "-D", s.BranchName)
	}
}

// DrainCompleted returns and clears all accumulated completed rewrites.
func (m *Manager) DrainCompleted() []PathRewrite {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := m.completedRewrites
	m.completedRewrites = nil
	m.completedSet = make(map[string]bool)
	return result
}

func ReadFileOrEmpty(path string) (content string, wasNew bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", true, nil
		}
		return "", false, err
	}
	return string(data), false, nil
}

func BuildFileDiff(ctx context.Context, worktreeRoot, relPath string) (string, error) {
	// First try git diff against worktree HEAD for tracked files/known paths.
	out, err := GitOutput(ctx, worktreeRoot, "diff", "--", relPath)
	if err == nil && strings.TrimSpace(out) != "" {
		return out, nil
	}

	// Fallback for newly-created/untracked files: diff against /dev/null.
	absPath := filepath.Join(worktreeRoot, filepath.FromSlash(relPath))
	if _, statErr := os.Stat(absPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return "", nil
		}
		return "", statErr
	}

	cmd := exec.CommandContext(ctx, "git", "-C", worktreeRoot, "diff", "--no-index", "--", "/dev/null", absPath)
	raw, diffErr := cmd.CombinedOutput()
	text := string(raw)
	if diffErr == nil {
		return text, nil
	}
	if ee, ok := diffErr.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		// git diff --no-index exits with 1 when differences are found.
		return text, nil
	}
	return "", fmt.Errorf("fallback diff: %w", diffErr)
}

func RevertRewrite(meta PathRewrite) error {
	if meta.WasNewFile {
		if err := os.Remove(meta.RewrittenAbs); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(meta.RewrittenAbs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(meta.RewrittenAbs, []byte(meta.BeforeContent), 0o644)
}

// PromoteRewrite copies the approved worktree file content into the main repo working tree.
func PromoteRewrite(meta PathRewrite, repoRoot string) error {
	target := filepath.Join(repoRoot, filepath.FromSlash(meta.RelPath))
	data, err := os.ReadFile(meta.RewrittenAbs)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func NormalizeRepoPath(repoRoot, inputPath string) (string, error) {
	if strings.TrimSpace(inputPath) == "" {
		return "", fmt.Errorf("empty path")
	}

	repoAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", err
	}

	var abs string
	if filepath.IsAbs(inputPath) {
		abs = filepath.Clean(inputPath)
	} else {
		abs = filepath.Join(repoAbs, inputPath)
	}
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(repoAbs, abs)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "", fmt.Errorf("path cannot be repo root")
	}
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("path outside repo: %s", inputPath)
	}
	return rel, nil
}

func GitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return string(out), nil
}

func rewriteKey(toolName, rewrittenAbs string) string {
	return toolName + "|" + rewrittenAbs
}

func randomWorktreeID() (string, error) {
	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("random id: %w", err)
	}
	return fmt.Sprintf("wt-%s-%s", time.Now().Format("20060102150405"), hex.EncodeToString(buf)), nil
}

package changebatch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileChange struct {
	ToolName      string
	InputPath     string
	AbsolutePath  string
	DisplayPath   string
	WasNewFile    bool
	BeforeContent string
	AfterContent  string
}

type BatchTracker interface {
	BeginChange(cwd, toolName, inputPath string) (FileChange, error)
	CompleteChange(toolName, inputPath string) (FileChange, bool, error)
	DiscardChange(toolName, inputPath string) bool
	DrainCompleted() []FileChange
	UndoBatch(changes []FileChange) error
	Reset()
}

// ChangeLog is reserved for future approved-batch history.
type ChangeLog interface {
	RecordApprovedBatch(changes []FileChange) error
}

type Manager struct {
	mu             sync.Mutex
	pending        map[string][]FileChange
	completed      []FileChange
	completedIndex map[string]int
}

func NewManager() *Manager {
	return &Manager{
		pending:        make(map[string][]FileChange),
		completedIndex: make(map[string]int),
	}
}

func (m *Manager) BeginChange(cwd, toolName, inputPath string) (FileChange, error) {
	absPath, displayPath, err := resolvePath(cwd, inputPath)
	if err != nil {
		return FileChange{}, err
	}

	beforeContent, wasNewFile, err := ReadFileOrEmpty(absPath)
	if err != nil {
		return FileChange{}, err
	}

	change := FileChange{
		ToolName:      toolName,
		InputPath:     inputPath,
		AbsolutePath:  absPath,
		DisplayPath:   displayPath,
		WasNewFile:    wasNewFile,
		BeforeContent: beforeContent,
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	key := pendingKey(toolName, inputPath)
	m.pending[key] = append(m.pending[key], change)
	return change, nil
}

func (m *Manager) CompleteChange(toolName, inputPath string) (FileChange, bool, error) {
	m.mu.Lock()
	change, ok := m.consumePendingLocked(toolName, inputPath)
	m.mu.Unlock()
	if !ok {
		return FileChange{}, false, nil
	}

	afterContent, _, err := ReadFileOrEmpty(change.AbsolutePath)
	if err != nil {
		return FileChange{}, false, err
	}
	change.AfterContent = afterContent

	m.mu.Lock()
	defer m.mu.Unlock()
	if idx, ok := m.completedIndex[change.AbsolutePath]; ok {
		m.completed[idx].AfterContent = change.AfterContent
		m.completed[idx].ToolName = change.ToolName
		m.completed[idx].InputPath = change.InputPath
		m.completed[idx].DisplayPath = change.DisplayPath
		return m.completed[idx], true, nil
	}

	m.completedIndex[change.AbsolutePath] = len(m.completed)
	m.completed = append(m.completed, change)
	return change, true, nil
}

func (m *Manager) DiscardChange(toolName, inputPath string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.consumePendingLocked(toolName, inputPath)
	return ok
}

func (m *Manager) DrainCompleted() []FileChange {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := append([]FileChange(nil), m.completed...)
	m.completed = nil
	m.completedIndex = make(map[string]int)
	return result
}

func (m *Manager) UndoBatch(changes []FileChange) error {
	var errs []string
	for _, change := range changes {
		if err := restoreOriginal(change); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", change.DisplayPath, err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("undo changes: %s", strings.Join(errs, "; "))
}

func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pending = make(map[string][]FileChange)
	m.completed = nil
	m.completedIndex = make(map[string]int)
}

func (m *Manager) consumePendingLocked(toolName, inputPath string) (FileChange, bool) {
	key := pendingKey(toolName, inputPath)
	queue := m.pending[key]
	if len(queue) == 0 {
		return FileChange{}, false
	}
	change := queue[0]
	if len(queue) == 1 {
		delete(m.pending, key)
	} else {
		m.pending[key] = queue[1:]
	}
	return change, true
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

func restoreOriginal(change FileChange) error {
	if change.WasNewFile {
		if err := os.Remove(change.AbsolutePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(change.AbsolutePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(change.AbsolutePath, []byte(change.BeforeContent), 0o644)
}

func resolvePath(cwd, inputPath string) (absPath, displayPath string, err error) {
	if strings.TrimSpace(inputPath) == "" {
		return "", "", fmt.Errorf("empty path")
	}

	if filepath.IsAbs(inputPath) {
		absPath = filepath.Clean(inputPath)
	} else {
		absPath = filepath.Join(cwd, inputPath)
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", "", err
	}

	displayPath = inputPath
	if filepath.IsAbs(inputPath) {
		if rel, relErr := filepath.Rel(cwd, absPath); relErr == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			displayPath = filepath.ToSlash(rel)
		} else {
			displayPath = filepath.ToSlash(absPath)
		}
	} else {
		displayPath = filepath.ToSlash(filepath.Clean(inputPath))
	}
	return absPath, displayPath, nil
}

func pendingKey(toolName, inputPath string) string {
	return toolName + "|" + inputPath
}

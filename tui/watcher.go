package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watcherDebounce = 2 * time.Second

// FileWatcher monitors a directory for file changes and reports change counts.
// It does NOT trigger indexing — only tracks which files have changed.
type FileWatcher struct {
	watcher      *fsnotify.Watcher
	changedFiles map[string]bool
	mu           sync.Mutex
	rootDir      string
}

// NewFileWatcher creates a watcher for rootDir.
func NewFileWatcher(rootDir string) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &FileWatcher{
		watcher:      w,
		changedFiles: make(map[string]bool),
		rootDir:      rootDir,
	}, nil
}

// Start begins watching and calls notify when change count updates (after debounce).
// Blocks until ctx is cancelled.
func (fw *FileWatcher) Start(ctx context.Context, notify func(count int)) {
	// Add root directory and subdirectories
	filepath.Walk(fw.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if shouldIgnoreWatchDir(name) {
				return filepath.SkipDir
			}
			fw.watcher.Add(path)
		}
		return nil
	})

	var debounceTimer *time.Timer

	for {
		select {
		case <-ctx.Done():
			fw.watcher.Close()
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Filter to relevant events
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			relPath, err := filepath.Rel(fw.rootDir, event.Name)
			if err != nil {
				continue
			}
			relPath = filepath.ToSlash(relPath)

			// Skip ignored paths
			if shouldIgnoreWatchPath(relPath) {
				continue
			}

			// If a new directory was created, start watching it
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if !shouldIgnoreWatchDir(info.Name()) {
						fw.watcher.Add(event.Name)
					}
					continue
				}
			}

			fw.mu.Lock()
			fw.changedFiles[relPath] = true
			count := len(fw.changedFiles)
			fw.mu.Unlock()

			// Debounce notifications
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(watcherDebounce, func() {
				notify(count)
			})

		case _, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			// Silently ignore watcher errors
		}
	}
}

// Stop closes the watcher.
func (fw *FileWatcher) Stop() {
	fw.watcher.Close()
}

// ChangedCount returns the number of changed files since last Reset.
func (fw *FileWatcher) ChangedCount() int {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return len(fw.changedFiles)
}

// Reset clears the changed file set (typically called after /index completes).
func (fw *FileWatcher) Reset() {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.changedFiles = make(map[string]bool)
}

func shouldIgnoreWatchDir(name string) bool {
	switch name {
	case ".git", ".bono", ".hg", ".svn", "node_modules", "__pycache__",
		".tox", ".mypy_cache", ".pytest_cache", "vendor", ".DS_Store":
		return true
	}
	return strings.HasPrefix(name, ".")
}

func shouldIgnoreWatchPath(relPath string) bool {
	parts := strings.Split(relPath, "/")
	for _, p := range parts {
		if shouldIgnoreWatchDir(p) {
			return true
		}
	}
	return false
}

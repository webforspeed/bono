package logging

import (
	"log/slog"
	"os"
	"path/filepath"
)

// New creates a JSON structured logger writing to the given file path.
// It ensures parent directories exist.
// Returns the logger, a close function for the underlying file, and any error.
func New(path string) (*slog.Logger, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(handler), f.Close, nil
}

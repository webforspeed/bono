package hooks

import (
	"context"
	"log/slog"
)

// LogHandler logs every hook event via a structured logger.
type LogHandler struct {
	Logger *slog.Logger
}

// NewLogHandler creates a LogHandler backed by the given logger.
func NewLogHandler(logger *slog.Logger) *LogHandler {
	return &LogHandler{Logger: logger}
}

func (h *LogHandler) Handle(ctx context.Context, event Event, payload any) {
	h.Logger.InfoContext(ctx, "hook",
		"event", string(event),
		"payload", payload,
	)
}

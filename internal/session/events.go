package session

// Event is a transport-neutral session event emitted by the agent session.
type Event interface {
	isSessionEvent()
}

type UserPromptEvent struct {
	Prompt string
}

func (UserPromptEvent) isSessionEvent() {}

type MessageEvent struct {
	Content string
}

func (MessageEvent) isSessionEvent() {}

type ToolCallEvent struct {
	Name            string
	Args            map[string]any
	Sandboxed       bool
	ExecutionReason string
}

func (ToolCallEvent) isSessionEvent() {}

type ToolDoneEvent struct {
	Name      string
	Args      map[string]any
	Status    string
	Sandboxed bool
}

func (ToolDoneEvent) isSessionEvent() {}

type DiffPreviewEvent struct {
	RelPath    string
	OldContent string
	NewContent string
}

func (DiffPreviewEvent) isSessionEvent() {}

type PreTaskStartEvent struct {
	Name string
}

func (PreTaskStartEvent) isSessionEvent() {}

type PreTaskEndEvent struct {
	Name string
}

func (PreTaskEndEvent) isSessionEvent() {}

type ErrorEvent struct {
	Err error
}

func (ErrorEvent) isSessionEvent() {}

type ContextUsageEvent struct {
	Pct       float64
	TotalCost float64
}

func (ContextUsageEvent) isSessionEvent() {}

type ContentDeltaEvent struct {
	Delta string
}

func (ContentDeltaEvent) isSessionEvent() {}

type ReasoningDeltaEvent struct {
	Delta string
}

func (ReasoningDeltaEvent) isSessionEvent() {}

type ResponseModelEvent struct {
	ModelID string
}

func (ResponseModelEvent) isSessionEvent() {}

type RefreshGitStatusEvent struct{}

func (RefreshGitStatusEvent) isSessionEvent() {}

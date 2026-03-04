package tui

// AgentMessageMsg is sent when the agent produces a message response.
type AgentMessageMsg string

// AgentToolCallMsg is sent when the agent wants to call a tool.
type AgentToolCallMsg struct {
	Name      string
	Args      map[string]any
	Approved  chan bool // nil for auto-approved tools, otherwise TUI sends approval here
	Sandboxed bool      // true if running in sandbox (shell/python)
}

// AgentToolDoneMsg is sent when a tool call completes.
type AgentToolDoneMsg struct {
	Name      string
	Args      map[string]any // needed to format the complete line
	Status    string
	Sandboxed bool // true if ran in sandbox (shell/python)
}

// AgentPreTaskStartMsg is sent when a pre-task agent starts.
type AgentPreTaskStartMsg string

// AgentPreTaskEndMsg is sent when a pre-task agent completes.
type AgentPreTaskEndMsg string

// AgentPreTaskDoneMsg is sent when a pre-task run completes (manual trigger).
type AgentPreTaskDoneMsg struct {
	Err error
}

// AgentSandboxFallbackMsg is sent when sandbox blocks a command and fallback is requested.
type AgentSandboxFallbackMsg struct {
	Command  string
	Reason   string
	Approved chan bool // TUI sends approval here
}

// AgentErrorMsg is sent when an error occurs during agent processing.
type AgentErrorMsg struct {
	Err error
}

// AgentContextUsageMsg is sent when context usage is updated after an LLM response.
type AgentContextUsageMsg struct {
	Pct       float64
	TotalCost float64
}

// AgentContentDeltaMsg carries a text content fragment from streaming.
type AgentContentDeltaMsg string

// AgentReasoningDeltaMsg carries a reasoning text fragment from streaming.
type AgentReasoningDeltaMsg string

// AgentResponseModelMsg is sent when core reports the concrete model used for a response.
type AgentResponseModelMsg struct {
	ModelID string
}

// ModelWarmDoneMsg is sent after background warm-up of usage limits for a switched model.
type ModelWarmDoneMsg struct {
	ModelID string
	Err     error
}

// SubmitInputMsg is sent internally when the user submits input.
type SubmitInputMsg struct {
	Value string
}

// IndexProgressMsg is sent during codebase indexing to report progress.
type IndexProgressMsg struct {
	Phase      string
	FilesDone  int
	FilesTotal int
}

// IndexDoneMsg is sent when indexing completes (or fails).
type IndexDoneMsg struct {
	Err         error
	TotalFiles  int
	TotalChunks int
	Duration    float64 // seconds
}

// WatcherNotifyMsg is sent when the file watcher detects changes since last index.
type WatcherNotifyMsg struct {
	ChangedCount int
}

// UpdateBannerMsg updates the status-bar banner with release update information.
type UpdateBannerMsg struct {
	Text string
}

package tui

// AgentMessageMsg is sent when the agent produces a message response.
type AgentMessageMsg string

// AgentToolCallMsg is sent when the agent wants to call a tool.
type AgentToolCallMsg struct {
	Name      string
	Args      map[string]any
	Approved  chan bool // nil for auto-approved tools, otherwise TUI sends approval here
	Sandboxed bool      // true if running in sandbox (shell only)
}

// AgentToolDoneMsg is sent when a tool call completes.
type AgentToolDoneMsg struct {
	Name      string
	Args      map[string]any // needed to format the complete line
	Status    string
	Sandboxed bool // true if ran in sandbox (shell only)
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

// SubmitInputMsg is sent internally when the user submits input.
type SubmitInputMsg struct {
	Value string
}

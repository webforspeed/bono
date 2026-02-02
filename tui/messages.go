package tui

// AgentMessageMsg is sent when the agent produces a message response.
type AgentMessageMsg string

// AgentToolCallMsg is sent when the agent wants to call a tool.
type AgentToolCallMsg struct {
	Name     string
	Args     map[string]any
	Approved chan bool // nil for auto-approved tools, otherwise TUI sends approval here
}

// AgentToolDoneMsg is sent when a tool call completes.
type AgentToolDoneMsg struct {
	Name   string
	Args   map[string]any // needed to format the complete line
	Status string
}

// AgentPreTaskStartMsg is sent when a pre-task agent starts.
type AgentPreTaskStartMsg string

// AgentPreTaskEndMsg is sent when a pre-task agent completes.
type AgentPreTaskEndMsg string

// AgentShellSubagentStartMsg is sent when a shell subagent starts execution.
// Contains the system prompt that defines the subagent's behavior.
type AgentShellSubagentStartMsg string

// AgentShellSubagentEndMsg is sent when a shell subagent completes.
type AgentShellSubagentEndMsg struct {
	Status string
}

// AgentSubagentToolCallMsg is sent when a subagent wants to call a tool.
type AgentSubagentToolCallMsg struct {
	Name     string
	Args     map[string]any
	Approved chan bool // TUI sends approval here
}

// AgentSubagentToolDoneMsg is sent when a subagent tool call completes.
type AgentSubagentToolDoneMsg struct {
	Name   string
	Args   map[string]any
	Status string
}

// AgentErrorMsg is sent when an error occurs during agent processing.
type AgentErrorMsg struct {
	Err error
}

// SubmitInputMsg is sent internally when the user submits input.
type SubmitInputMsg struct {
	Value string
}

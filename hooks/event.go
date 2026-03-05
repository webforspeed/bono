package hooks

// Event identifies a lifecycle hook point.
type Event string

const (
	SessionStart       Event = "SessionStart"
	SessionEnd         Event = "SessionEnd"
	UserPromptSubmit   Event = "UserPromptSubmit"
	PreToolUse         Event = "PreToolUse"
	PostToolUse        Event = "PostToolUse"
	PostToolUseFailure Event = "PostToolUseFailure"
	PermissionRequest  Event = "PermissionRequest"
	Stop               Event = "Stop"
	WorktreeCreate     Event = "WorktreeCreate"
	WorktreeRemove     Event = "WorktreeRemove"
)

// Payload structs — one per event that carries data.
// Handlers type-assert the payload to access fields.

type SessionStartPayload struct{}

type SessionEndPayload struct{}

type UserPromptSubmitPayload struct {
	Input string
}

type ToolPayload struct {
	ToolName string
	Args     map[string]any
}

type ToolResultPayload struct {
	ToolName string
	Args     map[string]any
	Status   string
	Success  bool
}

type PermissionPayload struct {
	ToolName string
	Args     map[string]any
}

type StopPayload struct {
	Response string
	Err      error
}

type WorktreePayload struct {
	Path string
}

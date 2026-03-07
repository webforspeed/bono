package session

import "context"

type ApprovalKind string

const (
	ApprovalTool            ApprovalKind = "tool"
	ApprovalSandboxFallback ApprovalKind = "sandbox_fallback"
	ApprovalChangeBatch     ApprovalKind = "change_batch"
)

// ApprovalRequest describes a user decision the frontend must resolve.
type ApprovalRequest struct {
	Kind            ApprovalKind
	ToolName        string
	ToolArgs        map[string]any
	ExecutionReason string
	Command         string
	Reason          string
	ChangeCount     int
}

// SessionFrontend is the narrow interface implemented by each transport.
type SessionFrontend interface {
	HandleEvent(ctx context.Context, event Event)
	RequestApproval(ctx context.Context, req ApprovalRequest) bool
}

package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	core "github.com/webforspeed/bono-core"
	"github.com/webforspeed/bono/internal/session"
)

type SessionFrontend struct {
	program *tea.Program
}

func NewSessionFrontend(program *tea.Program) *SessionFrontend {
	return &SessionFrontend{program: program}
}

func (f *SessionFrontend) HandleEvent(_ context.Context, event session.Event) {
	switch event := event.(type) {
	case session.MessageEvent:
		f.program.Send(AgentMessageMsg(event.Content))
	case session.ToolCallEvent:
		f.program.Send(AgentToolCallMsg{
			Name:            event.Name,
			Args:            event.Args,
			Sandboxed:       event.Sandboxed,
			ExecutionReason: event.ExecutionReason,
		})
	case session.ToolDoneEvent:
		f.program.Send(AgentToolDoneMsg{
			Name:      event.Name,
			Args:      event.Args,
			Status:    event.Status,
			Sandboxed: event.Sandboxed,
		})
	case session.DiffPreviewEvent:
		f.program.Send(AgentDiffPreviewMsg{
			RelPath:    event.RelPath,
			OldContent: event.OldContent,
			NewContent: event.NewContent,
		})
	case session.PreTaskStartEvent:
		f.program.Send(AgentPreTaskStartMsg(event.Name))
	case session.PreTaskEndEvent:
		f.program.Send(AgentPreTaskEndMsg(event.Name))
	case session.SubAgentStartEvent:
		f.program.Send(SubAgentStartMsg(event.Name))
	case session.SubAgentEndEvent:
		f.program.Send(SubAgentEndMsg(event.Name))
	case session.ErrorEvent:
		f.program.Send(AgentErrorMsg{Err: event.Err})
	case session.ContextUsageEvent:
		f.program.Send(AgentContextUsageMsg{Pct: event.Pct, TotalCost: event.TotalCost})
	case session.ContentDeltaEvent:
		f.program.Send(AgentContentDeltaMsg(event.Delta))
	case session.ReasoningDeltaEvent:
		f.program.Send(AgentReasoningDeltaMsg(event.Delta))
	case session.ResponseModelEvent:
		f.program.Send(AgentResponseModelMsg{ModelID: event.ModelID})
	case session.RefreshGitStatusEvent:
		f.program.Send(GitStatusMsg{Status: FetchGitStatus()})
	case session.UserPromptEvent:
	default:
	}
}

func (f *SessionFrontend) RequestApproval(ctx context.Context, req session.ApprovalRequest) bool {
	approved := make(chan bool, 1)

	switch req.Kind {
	case session.ApprovalTool:
		f.program.Send(AgentToolCallMsg{
			Name:            req.ToolName,
			Args:            req.ToolArgs,
			Approved:        approved,
			ExecutionReason: req.ExecutionReason,
		})
	case session.ApprovalSandboxFallback:
		f.program.Send(AgentSandboxFallbackMsg{
			Command:  req.Command,
			Reason:   req.Reason,
			Approved: approved,
		})
	case session.ApprovalChangeBatch:
		f.program.Send(AgentChangeBatchApprovalMsg{
			Count:    req.ChangeCount,
			Approved: approved,
		})
	default:
		return false
	}

	select {
	case result := <-approved:
		return result
	case <-ctx.Done():
		return false
	}
}

func (f *SessionFrontend) RequestSubAgentApproval(_ context.Context, result core.SubAgentResult) core.SubAgentApprovalResponse {
	ch := make(chan planApprovalResponse, 1)

	f.program.Send(AgentPlanApprovalMsg{
		OutputPath: result.Meta["output_path"],
		Response:   ch,
	})

	resp := <-ch
	var action core.SubAgentApprovalAction
	switch resp.Action {
	case 0:
		action = core.SubAgentApprove
	case 1:
		action = core.SubAgentReject
	case 2:
		action = core.SubAgentRevise
	}
	return core.SubAgentApprovalResponse{Action: action, Feedback: resp.Feedback}
}

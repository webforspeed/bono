package session

import (
	"context"
	"fmt"

	core "github.com/webforspeed/bono-core"
	"github.com/webforspeed/bono/hooks"
	"github.com/webforspeed/bono/internal/changebatch"
)

type Config struct {
	CWD         string
	ShellPolicy core.ShellPolicy
}

// Session owns frontend-neutral agent callback wiring and per-session change tracking.
type Session struct {
	agent      *core.Agent
	dispatcher *hooks.Dispatcher
	frontend   SessionFrontend
	config     Config

	changeBatchMgr changebatch.BatchTracker
}

func New(agent *core.Agent, dispatcher *hooks.Dispatcher, config Config, frontend SessionFrontend) *Session {
	return &Session{
		agent:          agent,
		dispatcher:     dispatcher,
		frontend:       frontend,
		config:         config,
		changeBatchMgr: changebatch.NewManager(),
	}
}

func (s *Session) Bind(ctx context.Context) {
	s.agent.OnToolCall = func(name string, args map[string]any) bool {
		s.dispatcher.Fire(ctx, hooks.PreToolUse, hooks.ToolPayload{ToolName: name, Args: args})

		if isReadOnlyTool(name) {
			s.frontend.HandleEvent(ctx, ToolCallEvent{Name: name, Args: args})
			return true
		}

		if isChangeTool(name) {
			originalPath, _ := args["path"].(string)
			if _, err := s.changeBatchMgr.BeginChange(s.config.CWD, name, originalPath); err != nil {
				s.frontend.HandleEvent(ctx, ErrorEvent{Err: fmt.Errorf("track %s change: %w", originalPath, err)})
				return false
			}
			s.frontend.HandleEvent(ctx, ToolCallEvent{Name: name, Args: args})
			return true
		}

		if name == "run_shell" || name == "python_runtime" {
			req := core.ShellRequestFromToolArgs(name, args)
			decision := core.DecideShellRequest(s.config.ShellPolicy, req)
			if decision.Route == core.ShellRouteHostDirect {
				s.dispatcher.Fire(ctx, hooks.PermissionRequest, hooks.PermissionPayload{ToolName: name, Args: args})
				return s.frontend.RequestApproval(ctx, ApprovalRequest{
					Kind:            ApprovalTool,
					ToolName:        name,
					ToolArgs:        args,
					ExecutionReason: decision.Reason,
				})
			}

			if core.IsSandboxEnabled() {
				s.frontend.HandleEvent(ctx, ToolCallEvent{Name: name, Args: args, Sandboxed: true})
				return true
			}

			s.dispatcher.Fire(ctx, hooks.PermissionRequest, hooks.PermissionPayload{ToolName: name, Args: args})
			return s.frontend.RequestApproval(ctx, ApprovalRequest{
				Kind:     ApprovalTool,
				ToolName: name,
				ToolArgs: args,
			})
		}

		s.dispatcher.Fire(ctx, hooks.PermissionRequest, hooks.PermissionPayload{ToolName: name, Args: args})
		return s.frontend.RequestApproval(ctx, ApprovalRequest{
			Kind:     ApprovalTool,
			ToolName: name,
			ToolArgs: args,
		})
	}

	s.agent.OnToolDone = func(name string, args map[string]any, result core.ToolResult) {
		payload := hooks.ToolResultPayload{ToolName: name, Args: args, Status: result.Status, Success: result.Success}
		if result.Success {
			s.dispatcher.Fire(ctx, hooks.PostToolUse, payload)
		} else {
			s.dispatcher.Fire(ctx, hooks.PostToolUseFailure, payload)
		}

		sandboxed := false
		if result.ExecMeta != nil {
			sandboxed = result.ExecMeta.Sandboxed
		}
		s.frontend.HandleEvent(ctx, ToolDoneEvent{
			Name:      name,
			Args:      args,
			Status:    result.Status,
			Sandboxed: sandboxed,
		})
		s.frontend.HandleEvent(ctx, RefreshGitStatusEvent{})

		if !isChangeTool(name) {
			return
		}
		originalPath, _ := args["path"].(string)
		if !result.Success {
			s.changeBatchMgr.DiscardChange(name, originalPath)
			return
		}
		if _, ok, err := s.changeBatchMgr.CompleteChange(name, originalPath); err != nil {
			s.frontend.HandleEvent(ctx, ErrorEvent{Err: fmt.Errorf("record final change for %s: %w", originalPath, err)})
		} else if !ok {
			return
		}
	}

	s.agent.OnMessage = func(content string) {
		s.frontend.HandleEvent(ctx, MessageEvent{Content: content})
	}
	s.agent.OnContentDelta = func(delta string) {
		s.frontend.HandleEvent(ctx, ContentDeltaEvent{Delta: delta})
	}
	s.agent.OnReasoningDelta = func(delta string) {
		s.frontend.HandleEvent(ctx, ReasoningDeltaEvent{Delta: delta})
	}
	s.agent.OnPreTaskStart = func(name string) {
		s.frontend.HandleEvent(ctx, PreTaskStartEvent{Name: name})
	}
	s.agent.OnPreTaskEnd = func(name string) {
		s.frontend.HandleEvent(ctx, PreTaskEndEvent{Name: name})
	}
	s.agent.OnSubAgentStart = func(name string) {
		s.frontend.HandleEvent(ctx, SubAgentStartEvent{Name: name})
	}
	s.agent.OnSubAgentEnd = func(name string) {
		s.frontend.HandleEvent(ctx, SubAgentEndEvent{Name: name})
	}
	s.agent.OnContextUsage = func(pct float64, totalCost float64) {
		s.frontend.HandleEvent(ctx, ContextUsageEvent{Pct: pct, TotalCost: totalCost})
	}
	s.agent.OnResponseModel = func(model string) {
		s.frontend.HandleEvent(ctx, ResponseModelEvent{ModelID: model})
	}
	s.agent.OnSandboxFallback = func(command string, reason string) bool {
		return s.frontend.RequestApproval(ctx, ApprovalRequest{
			Kind:    ApprovalSandboxFallback,
			Command: command,
			Reason:  reason,
		})
	}
}

func (s *Session) Reset() {
	s.changeBatchMgr.Reset()
}

func (s *Session) StopHandler() hooks.Handler {
	return hooks.HandlerFunc(func(ctx context.Context, _ hooks.Event, _ any) {
		completed := s.changeBatchMgr.DrainCompleted()
		if len(completed) == 0 {
			return
		}
		for _, change := range completed {
			s.frontend.HandleEvent(ctx, DiffPreviewEvent{
				RelPath:    change.DisplayPath,
				OldContent: change.BeforeContent,
				NewContent: change.AfterContent,
			})
		}

		ok := s.frontend.RequestApproval(ctx, ApprovalRequest{
			Kind:        ApprovalChangeBatch,
			ChangeCount: len(completed),
		})
		if !ok {
			if err := s.changeBatchMgr.UndoBatch(completed); err != nil {
				s.frontend.HandleEvent(ctx, ErrorEvent{Err: err})
			}
		}
		s.frontend.HandleEvent(ctx, RefreshGitStatusEvent{})
	})
}

func (s *Session) RunPrompt(ctx context.Context, prompt string) (string, error) {
	s.dispatcher.Fire(ctx, hooks.SessionStart, hooks.SessionStartPayload{})
	defer s.dispatcher.Fire(ctx, hooks.SessionEnd, hooks.SessionEndPayload{})

	s.dispatcher.Fire(ctx, hooks.UserPromptSubmit, hooks.UserPromptSubmitPayload{Input: prompt})
	s.frontend.HandleEvent(ctx, UserPromptEvent{Prompt: prompt})

	response, err := s.agent.Chat(ctx, prompt)
	if err != nil {
		s.frontend.HandleEvent(ctx, ErrorEvent{Err: err})
	}
	s.dispatcher.Fire(ctx, hooks.Stop, hooks.StopPayload{Response: response, Err: err})
	return response, err
}

func isReadOnlyTool(name string) bool {
	switch name {
	case "read_file", "compact_context", "code_search", "WebSearch", "WebFetch":
		return true
	default:
		return false
	}
}

func isChangeTool(name string) bool {
	return name == "write_file" || name == "edit_file"
}

package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/webforspeed/bono/hooks"
	"github.com/webforspeed/bono/internal/session"
)

// Update handles all incoming messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleResize(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinnerBar, cmd = m.spinnerBar.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		// Filter out garbage terminal response sequences
		// These come from terminal queries (like OSC 11 for background color)
		keyStr := msg.String()
		if isTerminalGarbage(keyStr) {
			return m, nil
		}

		// Model modal gets first chance at keys when active
		if m.modelModal.IsActive() {
			if cmd, handled := m.modelModal.HandleKey(msg); handled {
				m.recalculateLayout()
				if cmd != nil {
					return m, cmd
				}
				return m, nil
			}
		}

		// Reasoning modal gets keys when active
		if m.reasoningModal.IsActive() {
			if cmd, handled := m.reasoningModal.HandleKey(msg); handled {
				m.recalculateLayout()
				if cmd != nil {
					return m, cmd
				}
				return m, nil
			}
		}

		// Slash modal gets first chance at keys when active
		if m.slashModal.IsActive() {
			if selected, handled := m.slashModal.HandleKey(msg); handled {
				if selected != "" {
					m.input.SetValue(selected)
				}
				return m, nil
			}
		}

		if m.diffActive && msg.Type == tea.KeyTab {
			m.diffViewer.ToggleMode()
			m.rerenderDiffPreviews()
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			// If pending tool approval, approve it
			if m.pendingApproval != nil {
				m.pendingApproval.Approved <- true
				m.pendingApproval = nil
				m.spinnerBar.SetText("Thinking...")
				return m, nil
			}
			// If pending sandbox fallback approval, approve it
			if m.pendingSandboxFallback != nil {
				m.pendingSandboxFallback.Approved <- true
				m.pendingSandboxFallback = nil
				m.spinnerBar.SetText("Running unsandboxed...")
				return m, nil
			}
			// If a change batch is awaiting review, approve it.
			if m.pendingBatchApproval != nil {
				msg := m.pendingBatchApproval
				if len(m.messages) > 0 {
					m.messages[len(m.messages)-1] = m.renderBatchReviewLine(msg.Count, "approved")
					m.updateViewportContent()
				}
				msg.Approved <- true
				m.pendingBatchApproval = nil
				m.diffActive = false
				m.diffPreviews = nil
				m.spinnerBar.SetText("Thinking...")
				return m, nil
			}
			// If pending plan approval: empty input = approve, text = revise
			if m.pendingPlanApproval != nil {
				feedback := strings.TrimSpace(m.input.Value())
				msg := m.pendingPlanApproval
				m.pendingPlanApproval = nil
				m.input.Reset()
				if feedback == "" {
					// Approve
					if len(m.messages) > 0 {
						m.messages[len(m.messages)-1] = "  ↳ Plan approved — implementing..."
						m.updateViewportContent()
					}
					msg.Response <- planApprovalResponse{Action: 0}
					m.spinnerBar.SetText("Implementing plan...")
				} else {
					// Revise
					if len(m.messages) > 0 {
						m.messages[len(m.messages)-1] = fmt.Sprintf("  ↳ Revising plan: %s", feedback)
						m.updateViewportContent()
					}
					msg.Response <- planApprovalResponse{Action: 2, Feedback: feedback}
					m.spinnerBar.SetText("Revising plan...")
					m.spinnerBar.SetActive(true)
				}
				return m, nil
			}
			// If slash modal is active and Enter is pressed, select the command
			if m.slashModal.IsActive() {
				if cmd := m.slashModal.SelectedCommand(); cmd != nil {
					m.input.SetValue("/" + cmd.Name)
					m.slashModal.Update("") // Deactivate modal
					m.recalculateLayout()
					return m, nil
				}
			}
			// Otherwise submit input
			return m, m.submitInput()

		case tea.KeyCtrlC:
			// If pending approval, reject it before quitting
			if m.pendingApproval != nil {
				m.pendingApproval.Approved <- false
				m.pendingApproval = nil
			}
			if m.pendingSandboxFallback != nil {
				m.pendingSandboxFallback.Approved <- false
				m.pendingSandboxFallback = nil
			}
			if m.pendingBatchApproval != nil {
				m.pendingBatchApproval.Approved <- false
				m.pendingBatchApproval = nil
				m.diffActive = false
				m.diffPreviews = nil
			}
			if m.pendingPlanApproval != nil {
				m.pendingPlanApproval.Response <- planApprovalResponse{Action: 1}
				m.pendingPlanApproval = nil
			}
			return m, tea.Quit

		case tea.KeyEsc:
			// If pending tool approval, reject it
			if m.pendingApproval != nil {
				m.pendingApproval.Approved <- false
				m.pendingApproval = nil
				m.spinnerBar.SetText("Thinking...")
				// Update the message to show cancelled
				if len(m.messages) > 0 {
					m.messages[len(m.messages)-1] = strings.TrimSuffix(m.messages[len(m.messages)-1], " [Enter/Esc]") + " => cancelled"
					m.updateViewportContent()
				}
				return m, nil
			}
			// If pending sandbox fallback approval, reject it
			if m.pendingSandboxFallback != nil {
				m.pendingSandboxFallback.Approved <- false
				m.pendingSandboxFallback = nil
				m.spinnerBar.SetText("Thinking...")
				// Update the message to show cancelled
				if len(m.messages) > 0 {
					m.messages[len(m.messages)-1] = strings.TrimSuffix(m.messages[len(m.messages)-1], " [Enter/Esc]") + " => cancelled"
					m.updateViewportContent()
				}
				return m, nil
			}
			// If a change batch is awaiting review, undo it.
			if m.pendingBatchApproval != nil {
				msg := m.pendingBatchApproval
				if len(m.messages) > 0 {
					m.messages[len(m.messages)-1] = m.renderBatchReviewLine(msg.Count, "undone")
					m.updateViewportContent()
				}
				msg.Approved <- false
				m.pendingBatchApproval = nil
				m.diffActive = false
				m.diffPreviews = nil
				m.spinnerBar.SetText("Thinking...")
				return m, nil
			}
			// If pending plan approval, reject it
			if m.pendingPlanApproval != nil {
				if len(m.messages) > 0 {
					m.messages[len(m.messages)-1] = "  ↳ Plan skipped"
					m.updateViewportContent()
				}
				m.pendingPlanApproval.Response <- planApprovalResponse{Action: 1}
				m.pendingPlanApproval = nil
				m.spinnerBar.SetText("Thinking...")
				return m, nil
			}
			if m.slashModal.IsActive() {
				m.slashModal.Update("") // Deactivate modal
				m.recalculateLayout()
				return m, nil
			}
			return m, tea.Quit
		}

	// Streaming deltas
	case AgentContentDeltaMsg:
		m.streamingContent += string(msg)
		m.updateStreamingView()

	case AgentReasoningDeltaMsg:
		m.streamingReasoning += string(msg)
		m.updateStreamingView()

	// Agent messages
	case AgentMessageMsg:
		// If streaming was active, replace the raw placeholder with final markdown render.
		if m.isStreaming {
			m.isStreaming = false
			if len(m.messages) > 0 {
				m.messages = m.messages[:len(m.messages)-1]
			}
			// Preserve reasoning as a separate styled message above the response.
			if m.streamingReasoning != "" {
				m.messages = append(m.messages, m.renderReasoning("Thinking: "+m.streamingReasoning))
			}
			m.streamingContent = ""
			m.streamingReasoning = ""
		}
		m.AppendMessage(string(msg))

	case AgentToolCallMsg:
		// Finalize streaming if active (model emitted text before tool calls).
		if m.isStreaming {
			m.isStreaming = false
			if len(m.messages) > 0 {
				m.messages = m.messages[:len(m.messages)-1]
			}
			// Preserve reasoning as a separate styled message.
			if m.streamingReasoning != "" {
				m.messages = append(m.messages, m.renderReasoning("Thinking: "+m.streamingReasoning))
			}
			if content := m.streamingContent; content != "" {
				m.AppendMessage(content)
			}
			m.streamingContent = ""
			m.streamingReasoning = ""
		}
		prompt := session.FormatTool(msg.Name, msg.Args)
		// Soft wrap to viewport width using lipgloss
		wrapWidth := m.mainWidth() - 2
		if wrapWidth < 40 {
			wrapWidth = 40 // minimum width
		}
		wrapStyle := lipgloss.NewStyle().Width(wrapWidth)

		// Build display string with optional sandbox tag
		var displayStr string
		if msg.Sandboxed {
			// Sandboxed shell execution - auto-approved
			displayStr = fmt.Sprintf("● %s [Running in sandbox]", prompt)
			m.spinnerBar.SetText("Running in sandbox...")
		} else if msg.Approved != nil && msg.ExecutionReason != "" {
			displayStr = fmt.Sprintf("● %s [Outside sandbox: %s] [Enter/Esc]", prompt, msg.ExecutionReason)
			m.pendingApproval = &msg
			m.spinnerBar.SetText("Waiting for host execution approval...")
		} else if msg.Approved == nil {
			// Auto-approved (e.g., read_file) - just show it
			displayStr = fmt.Sprintf("● %s", prompt)
		} else {
			// Needs approval - show prompt and store for Enter/Esc handling
			displayStr = fmt.Sprintf("● %s [Enter/Esc]", prompt)
			m.pendingApproval = &msg
			m.spinnerBar.SetText("Waiting for approval...")
		}
		m.AppendRawMessage(wrapStyle.Render(displayStr))

	case AgentToolDoneMsg:
		// Update the last message to show result
		prompt := session.FormatTool(msg.Name, msg.Args)
		// Soft wrap to viewport width using lipgloss
		wrapWidth := m.mainWidth() - 2
		if wrapWidth < 40 {
			wrapWidth = 40
		}
		wrapStyle := lipgloss.NewStyle().Width(wrapWidth)

		// Add sandbox tag if applicable
		var sandboxTag string
		if msg.Sandboxed {
			sandboxTag = " [Ran in sandbox]"
		}

		if len(m.messages) > 0 {
			m.messages[len(m.messages)-1] = wrapStyle.Render(fmt.Sprintf("● %s%s => %s", prompt, sandboxTag, msg.Status))
			m.updateViewportContent()
		}

		// Refresh git status after tool calls (files may have changed)
		cmds = append(cmds, refreshGitStatus)

	case AgentDiffPreviewMsg:
		rendered := m.renderDiffPreview(msg)
		messageIndex := len(m.messages)
		m.AppendRawMessage(rendered)
		m.diffPreviews = append(m.diffPreviews, diffPreviewBlock{
			messageIndex: messageIndex,
			preview:      msg,
		})

	case AgentChangeBatchApprovalMsg:
		wrapWidth := m.mainWidth() - 2
		if wrapWidth < 40 {
			wrapWidth = 40
		}
		wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
		displayStr := fmt.Sprintf("● %s [Enter/Esc]", session.BatchReviewPrompt(msg.Count))
		m.AppendRawMessage(wrapStyle.Render(displayStr))
		m.pendingBatchApproval = &msg
		m.diffActive = true
		m.spinnerBar.SetText("Waiting for change approval...")

	case AgentPreTaskStartMsg:
		m.AppendRawMessage(fmt.Sprintf("● Running %s agent...", string(msg)))

	case AgentPreTaskEndMsg:
		m.AppendRawMessage(fmt.Sprintf("● Completed %s agent", string(msg)))

	case AgentPreTaskDoneMsg:
		m.processing = false
		m.spinnerBar.SetActive(false)
		if msg.Err != nil {
			m.AppendRawMessage(fmt.Sprintf("Error: %v", msg.Err))
		}

	case SubAgentStartMsg:
		m.sidebar.SetCurrentMode(string(msg))
		m.recalculateLayout()

	case SubAgentEndMsg:
		// Lifecycle event — completion handled by SubAgentDoneMsg.

	case SubAgentDoneMsg:
		m.sidebar.SetCurrentMode("")
		m.recalculateLayout()
		if msg.Err != nil {
			m.processing = false
			m.spinnerBar.SetActive(false)
			m.AppendRawMessage(fmt.Sprintf("  ↳ Failed: %v", msg.Err))
		} else if msg.Approved {
			// Plan approved — auto-trigger main agent to implement.
			m.spinnerBar.SetText("Implementing plan...")
			agent := m.agent
			ctx := m.ctx
			d := m.dispatcher
			return m, tea.Batch(m.spinnerBar.Tick(), func() tea.Msg {
				response, err := agent.Chat(ctx, "Implement the plan.")
				if d != nil {
					d.Fire(ctx, hooks.Stop, hooks.StopPayload{Response: response, Err: err})
				}
				return AgentResponseMsg{Response: response, Err: err}
			})
		} else {
			m.processing = false
			m.spinnerBar.SetActive(false)
		}

	case AgentPlanApprovalMsg:
		wrapWidth := m.mainWidth() - 2
		if wrapWidth < 40 {
			wrapWidth = 40
		}
		wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
		if msg.OutputPath != "" {
			m.AppendRawMessage(wrapStyle.Render(fmt.Sprintf("  ↳ Plan saved to %s", msg.OutputPath)))
		}
		m.AppendRawMessage(wrapStyle.Render("  ↳ Press Enter to implement, Esc to skip, or type feedback to revise [Enter/Esc]"))
		m.pendingPlanApproval = &msg
		m.spinnerBar.SetActive(false)
		m.spinnerBar.SetText("Review plan — Enter to implement, Esc to skip")

	case AgentSandboxFallbackMsg:
		// Sandbox blocked a command - request approval for unsandboxed execution
		wrapWidth := m.mainWidth() - 2
		if wrapWidth < 40 {
			wrapWidth = 40
		}
		wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
		reason := msg.Reason
		if reason == "" {
			reason = "policy violation"
		}
		displayCmd := session.DisplaySandboxCommand(msg.Command)
		displayStr := fmt.Sprintf("  ↳ %s [Sandbox blocked: %s] [Enter/Esc]", displayCmd, reason)
		m.AppendRawMessage(wrapStyle.Render(displayStr))
		m.pendingSandboxFallback = &msg
		m.spinnerBar.SetText("Sandbox blocked - approve unsandboxed?")

	case AgentContextUsageMsg:
		m.sidebar.SetContextUsage(msg.Pct)
		m.sidebar.SetTotalCost(msg.TotalCost)

	case AgentResponseModelMsg:
		if label := m.displayModelName(msg.ModelID); label != "" {
			m.sidebar.SetModelName(label)
		}
		// Async-warm model limits for the actual response model so context usage works.
		modelID := msg.ModelID
		cmds = append(cmds, func() tea.Msg {
			warmCtx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
			defer cancel()
			_ = m.agent.WarmModelUsageLimits(warmCtx, modelID)
			return ModelWarmDoneMsg{ModelID: modelID}
		})

	case ReasoningSelectedMsg:
		m.agent.SetReasoningEffort(msg.Level.Value)
		m.sidebar.SetReasoningEffort(msg.Level.Value)
		if msg.Level.Value == "" {
			m.AppendRawMessage("  ↳ Reasoning effort: disabled")
		} else {
			m.AppendRawMessage(fmt.Sprintf("  ↳ Reasoning effort: %s", msg.Level.Label))
		}
		m.recalculateLayout()

	case ModelSelectedMsg:
		if !msg.Model.IsLocal && m.agent.APIKey() == "" {
			m.AppendRawMessage(fmt.Sprintf("  ↳ Cannot use %s: OPENROUTER_API_KEY not set. Export it or add to .env file.", msg.Model.Name))
			m.recalculateLayout()
			break
		}
		m.agent.SetModel(msg.Model.ID)
		m.sidebar.SetModelName(msg.Model.Name)
		m.AppendRawMessage(fmt.Sprintf("  ↳ Switched to %s (%s)", msg.Model.Name, msg.Model.ID))
		m.agent.SetBaseURL(msg.Model.BaseURL)
		m.recalculateLayout()
		modelID := msg.Model.ID
		cmds = append(cmds, func() tea.Msg {
			warmCtx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
			defer cancel()
			err := m.agent.WarmModelUsageLimits(warmCtx, modelID)
			return ModelWarmDoneMsg{ModelID: modelID, Err: err}
		})

	case ModelWarmDoneMsg:
		// Ignore warm-up results for models that are no longer active.
		if m.agent.ModelName() != msg.ModelID {
			break
		}
		if msg.Err != nil {
			if shouldSuppressModelWarmWarning(msg.ModelID, msg.Err) {
				break
			}
			m.AppendRawMessage(fmt.Sprintf("  ↳ Warning: couldn't load usage limits for %s; context %% may be unavailable (%v)", msg.ModelID, msg.Err))
		}

	case AgentErrorMsg:
		m.AppendRawMessage(fmt.Sprintf("Error: %v", msg.Err))

	case IndexProgressMsg:
		m.spinnerBar.SetText(fmt.Sprintf("Indexing: %s (%d/%d files)", msg.Phase, msg.FilesDone, msg.FilesTotal))

	case IndexDoneMsg:
		m.processing = false
		m.spinnerBar.SetActive(false)
		if msg.Err != nil {
			m.AppendRawMessage(fmt.Sprintf("  ↳ Indexing failed: %v", msg.Err))
		} else {
			m.AppendRawMessage(fmt.Sprintf("  ↳ Index complete: %d chunks across %d files (%.1fs)",
				msg.TotalChunks, msg.TotalFiles, msg.Duration))
			m.sidebar.SetIndexStats(msg.TotalFiles)
		}
		// Reset watcher's changed file list since we just indexed
		if m.watcher != nil {
			m.watcher.Reset()
		}

	case WatcherNotifyMsg:
		if msg.ChangedCount > 0 {
			m.sidebar.SetChangedFiles(msg.ChangedCount)
		}
		// Refresh git status when files change
		cmds = append(cmds, refreshGitStatus)

	case GitStatusMsg:
		m.sidebar.SetGitStatus(msg.Status)
		// Schedule next periodic refresh
		cmds = append(cmds, scheduleGitStatusTick())

	case GitStatusTickMsg:
		cmds = append(cmds, refreshGitStatus)

	case UpdateBannerMsg:
		m.SetStatusBarBanner(msg.Text)

	case AgentResponseMsg:
		m.processing = false
		m.spinnerBar.SetActive(false)
		if msg.Err != nil {
			m.AppendRawMessage(fmt.Sprintf("Error: %v", msg.Err))
		}
		// Response content is already handled by OnMessage hook
		return m, nil

	case SubmitInputMsg:
		// This is handled by submitInput() returning a command
		return m, nil
	}

	// Only pass relevant messages to input (not system messages like WindowSizeMsg)
	switch msg.(type) {
	case tea.KeyMsg:
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)
		cmds = append(cmds, inputCmd)

		// Update slash modal based on current input value
		m.slashModal.Update(m.input.Value())
		m.recalculateLayout()
	}

	// Update viewport with all messages
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// isTerminalGarbage detects terminal response sequences that shouldn't be treated as input.
// These include OSC responses (like ]11;rgb:...), CSI responses (like [1;1R), etc.
func isTerminalGarbage(s string) bool {
	// OSC sequences (Operating System Command) - start with ] or contain rgb:
	if strings.HasPrefix(s, "]") || strings.Contains(s, "rgb:") {
		return true
	}
	// CSI cursor position reports - like [1;1R or similar
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "R") {
		return true
	}
	// Any string containing escape sequences
	if strings.Contains(s, "\x1b") || strings.Contains(s, "\033") {
		return true
	}
	// Sequences starting with ; (often partial responses)
	if strings.HasPrefix(s, ";") {
		return true
	}
	return false
}

func shouldSuppressModelWarmWarning(modelID string, err error) bool {
	if err == nil || modelID != "openrouter/free" {
		return false
	}
	return strings.Contains(err.Error(), `no endpoint limits for model "openrouter/free"`)
}

// refreshGitStatus fetches the current git status for the sidebar.
func refreshGitStatus() tea.Msg {
	return GitStatusMsg{Status: FetchGitStatus()}
}

// renderBatchReviewLine builds a formatted batch review line with the given status suffix.
func (m Model) renderBatchReviewLine(count int, status string) string {
	wrapWidth := m.mainWidth() - 2
	if wrapWidth < 40 {
		wrapWidth = 40
	}
	wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
	displayStr := fmt.Sprintf("● %s => %s", session.BatchReviewPrompt(count), status)
	return wrapStyle.Render(displayStr)
}

func (m *Model) rerenderDiffPreviews() {
	for _, block := range m.diffPreviews {
		if block.messageIndex >= 0 && block.messageIndex < len(m.messages) {
			m.messages[block.messageIndex] = m.renderDiffPreview(block.preview)
		}
	}
	m.updateViewportContent()
}

func (m Model) renderDiffPreview(preview AgentDiffPreviewMsg) string {
	viewer := NewDiffViewer()
	viewer.viewMode = m.diffViewer.viewMode

	width := m.mainWidth() - 2
	if width < 40 {
		width = 40
	}
	viewer.SetSize(width, 20)
	viewer.SetContent(
		preview.OldContent,
		preview.NewContent,
		preview.RelPath+" (before)",
		preview.RelPath+" (after)",
	)
	return viewer.RenderFull()
}

// scheduleGitStatusTick returns a command that fires a GitStatusTickMsg after a delay.
func scheduleGitStatusTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return GitStatusTickMsg{}
	})
}

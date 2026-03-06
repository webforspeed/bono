package tui

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
			// Re-render the inline diff message
			if m.diffMessageIndex >= 0 && m.diffMessageIndex < len(m.messages) {
				m.messages[m.diffMessageIndex] = m.diffViewer.RenderFull()
				m.updateViewportContent()
			}
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
			// If pending diff approval, approve it
			if m.pendingDiffApproval != nil {
				msg := m.pendingDiffApproval
				if len(m.messages) > 0 {
					m.messages[len(m.messages)-1] = m.renderReviewLine(msg.RelPath, msg.Index, msg.Total, "ok")
					m.updateViewportContent()
				}
				msg.Approved <- true
				m.pendingDiffApproval = nil
				m.diffActive = false
				m.spinnerBar.SetText("Thinking...")
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
			if m.pendingDiffApproval != nil {
				m.pendingDiffApproval.Approved <- false
				m.pendingDiffApproval = nil
				m.diffActive = false
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
			// If pending diff approval, reject it
			if m.pendingDiffApproval != nil {
				msg := m.pendingDiffApproval
				if len(m.messages) > 0 {
					m.messages[len(m.messages)-1] = m.renderReviewLine(msg.RelPath, msg.Index, msg.Total, "skipped")
					m.updateViewportContent()
				}
				msg.Approved <- false
				m.pendingDiffApproval = nil
				m.diffActive = false
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
				m.messages = append(m.messages, m.styles.Reasoning.Render("Thinking: "+m.streamingReasoning))
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
				m.messages = append(m.messages, m.styles.Reasoning.Render("Thinking: "+m.streamingReasoning))
			}
			if content := m.streamingContent; content != "" {
				m.AppendMessage(content)
			}
			m.streamingContent = ""
			m.streamingReasoning = ""
		}
		prompt := formatTool(msg.Name, msg.Args)
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
		prompt := formatTool(msg.Name, msg.Args)
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
		m.diffViewer.SetContent(msg.OldContent, msg.NewContent, msg.RelPath+" (before)", msg.RelPath+" (after)")
		// Render the diff inline in the viewport
		m.diffMessageIndex = len(m.messages)
		m.AppendRawMessage(m.diffViewer.RenderFull())

	case AgentDiffApprovalMsg:
		wrapWidth := m.mainWidth() - 2
		if wrapWidth < 40 {
			wrapWidth = 40
		}
		wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
		var displayStr string
		if msg.Total > 0 {
			displayStr = fmt.Sprintf("● Review('%s') (%d/%d) [Enter/Esc]", msg.RelPath, msg.Index, msg.Total)
		} else {
			displayStr = fmt.Sprintf("● Review('%s') [Enter/Esc]", msg.RelPath)
		}
		m.AppendRawMessage(wrapStyle.Render(displayStr))
		m.pendingDiffApproval = &msg
		m.diffActive = true
		m.spinnerBar.SetText("Waiting for diff approval...")

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
		displayCmd := fmt.Sprintf("Bash(%s)", msg.Command)
		if code, ok := pythonCodeFromCommand(msg.Command); ok {
			displayCmd = fmt.Sprintf("Python(%s)", code)
		}
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
		m.agent.SetModel(msg.Model.ID)
		m.sidebar.SetModelName(msg.Model.Name)
		m.AppendRawMessage(fmt.Sprintf("  ↳ Switched to %s (%s)", msg.Model.Name, msg.Model.ID))
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

// formatTool formats a tool call for display with friendly names.
func formatTool(name string, args map[string]any) string {
	switch name {
	case "read_file":
		path, _ := args["path"].(string)
		return fmt.Sprintf("Read('%s')", path)
	case "write_file":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		lines := len(strings.Split(content, "\n"))
		return fmt.Sprintf("Write('%s', %d lines)", path, lines)
	case "edit_file":
		path, _ := args["path"].(string)
		return fmt.Sprintf("Edit('%s')", path)
	case "run_shell":
		cmd, _ := args["command"].(string)
		desc, _ := args["description"].(string)
		safety, _ := args["safety"].(string)
		if desc == "" {
			desc = "(no description)"
		}
		if safety == "" {
			safety = "modify"
		}
		return fmt.Sprintf("Bash('%s') # %s, %s", cmd, desc, safety)
	case "python_runtime":
		code, _ := args["code"].(string)
		desc, _ := args["description"].(string)
		safety, _ := args["safety"].(string)
		if desc == "" {
			desc = "(no description)"
		}
		if safety == "" {
			safety = "modify"
		}
		if code == "" {
			code = "(empty code)"
		}
		return fmt.Sprintf("Python(%s) # %s, %s", code, desc, safety)
	case "compact_context":
		return "Compact(context)"
	case "code_search":
		query, _ := args["query"].(string)
		searchType, _ := args["search_type"].(string)
		if searchType == "" {
			searchType = "semantic"
		}
		return fmt.Sprintf("Search('%s', %s)", query, searchType)
	case "WebSearch":
		query, _ := args["query"].(string)
		mode, _ := args["mode"].(string)
		if mode != "" {
			return fmt.Sprintf("WebSearch('%s', %s)", query, mode)
		}
		return fmt.Sprintf("WebSearch('%s')", query)
	case "WebFetch":
		url, _ := args["url"].(string)
		question, _ := args["question"].(string)
		if question != "" {
			return fmt.Sprintf("WebFetch('%s', '%s')", url, question)
		}
		return fmt.Sprintf("WebFetch('%s')", url)
	default:
		return name
	}
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

// renderReviewLine builds a formatted review line with the given status suffix.
func (m Model) renderReviewLine(relPath string, index, total int, status string) string {
	wrapWidth := m.mainWidth() - 2
	if wrapWidth < 40 {
		wrapWidth = 40
	}
	wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
	var displayStr string
	if total > 0 {
		displayStr = fmt.Sprintf("● Review('%s') (%d/%d) => %s", relPath, index, total, status)
	} else {
		displayStr = fmt.Sprintf("● Review('%s') => %s", relPath, status)
	}
	return wrapStyle.Render(displayStr)
}

// scheduleGitStatusTick returns a command that fires a GitStatusTickMsg after a delay.
func scheduleGitStatusTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return GitStatusTickMsg{}
	})
}

func pythonCodeFromCommand(command string) (string, bool) {
	const marker = "base64.b64decode('"
	idx := strings.Index(command, marker)
	if idx == -1 {
		return "", false
	}
	start := idx + len(marker)
	end := strings.Index(command[start:], "')")
	if end == -1 {
		return "", false
	}
	encoded := command[start : start+end]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", false
	}
	return string(decoded), true
}

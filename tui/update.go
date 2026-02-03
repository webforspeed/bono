package tui

import (
	"fmt"
	"strings"

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

		// Slash modal gets first chance at keys when active
		if m.slashModal.IsActive() {
			if selected, handled := m.slashModal.HandleKey(msg); handled {
				if selected != "" {
					m.input.SetValue(selected)
				}
				return m, nil
			}
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
			if m.slashModal.IsActive() {
				m.slashModal.Update("") // Deactivate modal
				m.recalculateLayout()
				return m, nil
			}
			return m, tea.Quit
		}

	// Agent messages
	case AgentMessageMsg:
		m.AppendMessage(string(msg))

	case AgentToolCallMsg:
		prompt := formatTool(msg.Name, msg.Args)
		// Soft wrap to viewport width using lipgloss
		wrapWidth := m.width - 2
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
		wrapWidth := m.width - 2
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

	case AgentPreTaskStartMsg:
		m.AppendRawMessage(fmt.Sprintf("● Running %s agent...", string(msg)))

	case AgentPreTaskEndMsg:
		m.AppendRawMessage(fmt.Sprintf("● Completed %s agent", string(msg)))

	case AgentSandboxFallbackMsg:
		// Sandbox blocked a command - request approval for unsandboxed execution
		wrapWidth := m.width - 2
		if wrapWidth < 40 {
			wrapWidth = 40
		}
		wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
		reason := msg.Reason
		if reason == "" {
			reason = "policy violation"
		}
		displayStr := fmt.Sprintf("  ↳ Bash(%s) [Sandbox blocked: %s] [Enter/Esc]", msg.Command, reason)
		m.AppendRawMessage(wrapStyle.Render(displayStr))
		m.pendingSandboxFallback = &msg
		m.spinnerBar.SetText("Sandbox blocked - approve unsandboxed?")

	case AgentErrorMsg:
		m.AppendRawMessage(fmt.Sprintf("Error: %v", msg.Err))

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

package tui

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	core "github.com/webforspeed/bono-core"
)

// Model is the main Bubble Tea model that composes all TUI components.
type Model struct {
	// Composed components
	viewport   viewport.Model
	input      InputBox
	spinnerBar SpinnerBar
	statusBar  StatusBar
	slashModal SlashModal

	// Shared state
	messages []string
	styles   Styles

	// External dependencies
	agent    *core.Agent
	ctx      context.Context
	renderer *glamour.TermRenderer

	// For async agent calls
	program *tea.Program

	// Dimensions
	width, height int
	ready         bool
	processing    bool // true when agent is processing

	// Tool approval state
	pendingApproval *AgentToolCallMsg // current tool awaiting Enter/Esc
}

// New creates a new TUI Model with the given agent and context.
func New(agent *core.Agent, ctx context.Context) Model {
	return NewWithSpinnerType(agent, ctx, SpinnerDot)
}

// NewWithSpinnerType creates a new TUI Model with the given agent, context, and spinner type.
func NewWithSpinnerType(agent *core.Agent, ctx context.Context, spinnerType SpinnerType) Model {
	// Use DarkStyle instead of AutoStyle to avoid terminal queries that cause garbage input
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(80),
	)

	// Get current working directory for display
	cwd, _ := os.Getwd()

	spinnerBar := NewSpinnerBar(spinnerType)
	spinnerBar.SetIdleText(cwd)

	return Model{
		viewport:   viewport.New(80, 20),
		input:      NewInputBox(),
		spinnerBar: spinnerBar,
		statusBar:  NewStatusBar(),
		slashModal: NewSlashModal(),
		styles:     DefaultStyles(),
		agent:      agent,
		ctx:        ctx,
		renderer:   renderer,
		messages:   []string{},
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.spinnerBar.Tick())
}

// AppendMessage adds a message to the viewport.
func (m *Model) AppendMessage(content string) {
	// Render markdown if renderer is available
	if m.renderer != nil {
		rendered, err := m.renderer.Render(content)
		if err == nil {
			content = rendered
		}
	}
	m.messages = append(m.messages, content)
	m.updateViewportContent()
}

// AppendRawMessage adds a raw (non-markdown) message to the viewport.
func (m *Model) AppendRawMessage(content string) {
	m.messages = append(m.messages, content)
	m.updateViewportContent()
}

// updateViewportContent updates the viewport with the current messages.
func (m *Model) updateViewportContent() {
	content := strings.Join(m.messages, "\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// ClearMessages clears all messages from the viewport.
func (m *Model) ClearMessages() {
	m.messages = []string{}
	m.viewport.SetContent("")
}

// SetStatus updates the spinner bar text.
func (m *Model) SetStatus(text string) {
	m.spinnerBar.SetText(text)
}

// SetSpinnerType changes the spinner style.
func (m *Model) SetSpinnerType(t SpinnerType) {
	m.spinnerBar.SetSpinnerType(t)
}

// GetSpinnerType returns the current spinner type.
func (m *Model) GetSpinnerType() SpinnerType {
	return m.spinnerBar.GetSpinnerType()
}

// NextSpinnerType cycles to the next spinner type.
func (m *Model) NextSpinnerType() {
	m.spinnerBar.NextSpinnerType()
}

// GetAgent returns the agent for external configuration.
func (m *Model) GetAgent() *core.Agent {
	return m.agent
}

// SetProgram sets the tea.Program reference for async operations.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// IsProcessing returns whether the agent is currently processing.
func (m Model) IsProcessing() bool {
	return m.processing
}

// AgentResponseMsg is sent when the agent finishes processing.
type AgentResponseMsg struct {
	Response string
	Err      error
}

// handleResize handles terminal resize events.
func (m *Model) handleResize(msg tea.WindowSizeMsg) {
	// Skip if dimensions haven't actually changed
	if m.ready && m.width == msg.Width && m.height == msg.Height {
		return
	}

	m.width = msg.Width
	m.height = msg.Height
	m.ready = true

	// Calculate heights
	spinnerHeight := 1 // Spinner bar above input
	inputHeight := 3   // Input box with border
	statusHeight := 1  // Status bar
	slashHeight := m.slashModal.Height()

	// Set component widths
	m.spinnerBar.SetWidth(msg.Width)
	m.input.SetWidth(msg.Width, m.styles.InputBox)
	m.statusBar.SetWidth(msg.Width)
	m.slashModal.SetWidth(msg.Width)

	// Viewport gets remaining space
	m.viewport.Width = msg.Width
	m.viewport.Height = msg.Height - spinnerHeight - inputHeight - statusHeight - slashHeight

	// Note: We don't recreate the glamour renderer on resize because
	// glamour.WithAutoStyle() queries the terminal and causes garbage input.
	// The initial word wrap of 80 chars is sufficient for most cases.
}

// submitInput handles submitting the current input.
func (m *Model) submitInput() tea.Cmd {
	value := strings.TrimSpace(m.input.Value())
	if value == "" || m.processing {
		return nil
	}

	// Handle slash commands
	if strings.HasPrefix(value, "/") {
		return m.handleSlashCommand(value)
	}

	// Clear input and submit to agent
	m.input.Reset()

	// Add user message to viewport
	m.AppendRawMessage("> " + value)

	// Mark as processing and activate spinner
	m.processing = true
	m.spinnerBar.SetText("Thinking...")
	m.spinnerBar.SetActive(true)

	// Return a command that will call the agent asynchronously
	agent := m.agent
	ctx := m.ctx
	return tea.Batch(
		m.spinnerBar.Tick(),
		func() tea.Msg {
			response, err := agent.Chat(ctx, value)
			return AgentResponseMsg{Response: response, Err: err}
		},
	)
}

// handleSlashCommand processes slash commands.
func (m *Model) handleSlashCommand(cmd string) tea.Cmd {
	cmd = strings.TrimPrefix(cmd, "/")
	cmd = strings.TrimSpace(cmd)

	// Handle commands with arguments
	parts := strings.SplitN(cmd, " ", 2)
	cmdName := parts[0]
	var cmdArg string
	if len(parts) > 1 {
		cmdArg = strings.TrimSpace(parts[1])
	}

	switch cmdName {
	case "clear":
		m.ClearMessages()
		m.input.Reset()
		return nil

	case "exit":
		return tea.Quit

	case "help":
		m.AppendRawMessage("Available commands:\n" +
			"  /help           - Show this help\n" +
			"  /clear          - Clear chat history\n" +
			"  /model          - Show current model\n" +
			"  /context        - Show context info\n" +
			"  /spinner        - Cycle to next spinner style\n" +
			"  /spinner <type> - Set spinner (dot, line, minidot, jump, pulse, points, globe, moon, monkey, meter, hamburger, ellipsis)\n" +
			"  /exit           - Exit Bono")
		m.input.Reset()
		return nil

	case "model":
		m.AppendRawMessage("Model info: (dynamic info coming soon)")
		m.input.Reset()
		return nil

	case "context":
		m.AppendRawMessage("Context info: (dynamic info coming soon)")
		m.input.Reset()
		return nil

	case "spinner":
		if cmdArg == "" {
			// Cycle to next spinner
			m.NextSpinnerType()
			typeName := SpinnerTypeNames[m.GetSpinnerType()]
			m.AppendRawMessage("Spinner changed to: " + typeName)
		} else {
			// Set specific spinner type
			newType := ParseSpinnerType(cmdArg)
			m.SetSpinnerType(newType)
			typeName := SpinnerTypeNames[m.GetSpinnerType()]
			m.AppendRawMessage("Spinner set to: " + typeName)
		}
		m.input.Reset()
		return nil

	default:
		m.AppendRawMessage("Unknown command: /" + cmd)
		m.input.Reset()
		return nil
	}
}

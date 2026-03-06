package tui

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	core "github.com/webforspeed/bono-core"
	"github.com/webforspeed/bono/hooks"
)

// Model is the main Bubble Tea model that composes all TUI components.
type Model struct {
	// Composed components
	viewport       viewport.Model
	input          InputBox
	spinnerBar     SpinnerBar
	statusBar      StatusBar
	sidebar        Sidebar
	slashModal     SlashModal
	modelModal     ModelModal
	reasoningModal ReasoningModal

	// Shared state
	messages          []string
	styles            Styles
	slashCommands     []SlashCommandSpec
	slashCommandIndex map[string]SlashCommandSpec
	statusBarBaseText string
	statusBarBanner   string

	// External dependencies
	agent      *core.Agent
	ctx        context.Context
	renderer   *glamour.TermRenderer
	dispatcher *hooks.Dispatcher

	// For async agent calls
	program *tea.Program

	// Dimensions
	width, height int
	ready         bool
	processing    bool // true when agent is processing

	// Tool approval state
	pendingApproval        *AgentToolCallMsg        // current tool awaiting Enter/Esc
	pendingSandboxFallback *AgentSandboxFallbackMsg // sandbox fallback awaiting Enter/Esc
	pendingDiffApproval    *AgentDiffApprovalMsg    // diff approval awaiting Enter/Esc

	// Diff viewer state
	diffViewer       DiffViewer
	diffActive       bool // true when diff is pending approval (for Tab key handling)
	diffMessageIndex int  // index in m.messages where the inline diff is rendered

	// Code search watcher metadata
	watcher *FileWatcher

	// Streaming state
	streamingContent   string // accumulates content deltas
	streamingReasoning string // accumulates reasoning deltas
	isStreaming        bool   // true while streaming response in progress
}

// New creates a new TUI Model with the given agent and context.
func New(agent *core.Agent, ctx context.Context) Model {
	return NewWithOptions(agent, ctx, SpinnerDot, nil)
}

// NewWithOptions creates a new TUI Model with the given agent, context, spinner type, and model catalog.
func NewWithOptions(agent *core.Agent, ctx context.Context, spinnerType SpinnerType, models []ModelInfo) Model {
	// Use DarkStyle instead of AutoStyle to avoid terminal queries that cause garbage input
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(80),
	)

	// Get current working directory for display
	cwd, _ := os.Getwd()

	spinnerBar := NewSpinnerBar(spinnerType)
	statusBar := NewStatusBar()

	// Resolve display name for current model
	modelName := agent.ModelName()
	for _, m := range models {
		if m.ID == modelName {
			modelName = m.Name
			break
		}
	}

	sidebar := NewSidebar()
	sidebar.SetCwd(cwd)
	sidebar.SetModelName(modelName)
	sidebar.SetGitStatus(FetchGitStatus())

	slashCommands := DefaultSlashCommandSpecs()

	return Model{
		viewport:          viewport.New(80, 20),
		input:             NewInputBox(),
		spinnerBar:        spinnerBar,
		statusBar:         statusBar,
		sidebar:           sidebar,
		slashModal:        NewSlashModal(),
		modelModal:        NewModelModal(models),
		reasoningModal:    NewReasoningModal(),
		diffViewer:        NewDiffViewer(),
		styles:            DefaultStyles(),
		slashCommands:     slashCommands,
		slashCommandIndex: slashCommandIndex(slashCommands),
		statusBarBaseText: statusBar.Text(),
		agent:             agent,
		ctx:               ctx,
		renderer:          renderer,
		messages:          []string{},
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.spinnerBar.Tick(), scheduleGitStatusTick())
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

// updateStreamingView updates the viewport with the current streaming content.
// Shows raw text during streaming (no markdown) for speed.
func (m *Model) updateStreamingView() {
	content := m.streamingContent

	// Show reasoning dimmed above content if present.
	var display string
	if reasoning := m.streamingReasoning; reasoning != "" {
		dimStyle := m.styles.Reasoning
		display = dimStyle.Render("Thinking: "+reasoning) + "\n\n"
	}
	display += content

	// Replace or append the streaming placeholder in messages.
	if len(m.messages) > 0 && m.isStreaming {
		m.messages[len(m.messages)-1] = display
	} else {
		m.messages = append(m.messages, display)
		m.isStreaming = true
	}
	m.updateViewportContent()
}

// recalculateLayout recomputes component sizes based on current dimensions.
func (m *Model) recalculateLayout() {
	if !m.ready {
		return
	}

	// Two-column width split
	sidebarW := sidebarWidth
	if m.width < 80 {
		sidebarW = 0
	}
	mainW := m.width - sidebarW

	// Calculate heights
	spinnerHeight := 1 // Spinner bar above input
	inputHeight := 3   // Input box with border
	statusHeight := 1  // Status bar
	slashHeight := m.slashModal.Height()
	modelHeight := m.modelModal.Height()
	reasoningHeight := m.reasoningModal.Height()

	// Set component widths to main column width
	m.spinnerBar.SetWidth(mainW)
	m.input.SetWidth(mainW, m.styles.InputBox)
	m.statusBar.SetWidth(mainW)
	m.slashModal.SetWidth(mainW)
	m.modelModal.SetWidth(mainW)
	m.reasoningModal.SetWidth(mainW)

	// Viewport gets remaining height, using main column width
	m.viewport.Width = mainW
	height := m.height - spinnerHeight - inputHeight - statusHeight - slashHeight - modelHeight - reasoningHeight
	if height < 1 {
		height = 1
	}
	m.viewport.Height = height

	// Sidebar gets full height and fixed width
	m.sidebar.SetWidth(sidebarW)
	m.sidebar.SetHeight(m.height)
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

// SetIndexStats updates the sidebar with index file count.
func (m *Model) SetIndexStats(files int) {
	m.sidebar.SetIndexStats(files)
}

// SetStatusBarText updates the bottom status bar text.
func (m *Model) SetStatusBarText(text string) {
	m.statusBarBaseText = text
	m.refreshStatusBarText()
}

// SetStatusBarBanner updates the optional footer banner segment.
func (m *Model) SetStatusBarBanner(text string) {
	m.statusBarBanner = strings.TrimSpace(text)
	m.refreshStatusBarText()
}

func (m *Model) refreshStatusBarText() {
	text := m.statusBarBaseText
	if text == "" {
		text = m.statusBar.Text()
	}
	if m.statusBarBanner != "" {
		text += " • " + m.statusBarBanner
	}
	m.statusBar.SetText(text)
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

// mainWidth returns the width of the main content column (excluding sidebar).
func (m Model) mainWidth() int {
	if m.width < 80 {
		return m.width
	}
	return m.width - sidebarWidth
}

// SetWatcher sets the file watcher for change notifications.
func (m *Model) SetWatcher(w *FileWatcher) {
	m.watcher = w
}

// SetDispatcher sets the hook dispatcher for lifecycle events.
func (m *Model) SetDispatcher(d *hooks.Dispatcher) {
	m.dispatcher = d
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

	m.recalculateLayout()

	// Note: We don't recreate the glamour renderer on resize because
	// glamour.WithAutoStyle() queries the terminal and causes garbage input.
	// The initial word wrap of 80 chars is sufficient for most cases.
}

func (m *Model) displayModelName(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	for _, info := range m.modelModal.models {
		if info.ID == modelID {
			return info.Name
		}
	}
	return modelID
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

	// Fire UserPromptSubmit hook
	if m.dispatcher != nil {
		m.dispatcher.Fire(m.ctx, hooks.UserPromptSubmit, hooks.UserPromptSubmitPayload{Input: value})
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
	d := m.dispatcher
	return tea.Batch(
		m.spinnerBar.Tick(),
		func() tea.Msg {
			response, err := agent.Chat(ctx, value)
			if d != nil {
				d.Fire(ctx, hooks.Stop, hooks.StopPayload{Response: response, Err: err})
			}
			return AgentResponseMsg{Response: response, Err: err}
		},
	)
}

// handleSlashCommand processes slash commands.
func (m *Model) handleSlashCommand(cmd string) tea.Cmd {
	cmd = strings.TrimPrefix(cmd, "/")
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		m.input.Reset()
		return nil
	}

	// Handle commands with arguments
	parts := strings.SplitN(cmd, " ", 2)
	cmdName := strings.ToLower(parts[0])
	var cmdArg string
	if len(parts) > 1 {
		cmdArg = strings.TrimSpace(parts[1])
	}

	if spec, ok := m.slashCommandIndex[cmdName]; ok {
		return spec.Handler(m, cmdArg)
	}

	m.AppendRawMessage("Unknown command: /" + cmd)
	m.input.Reset()
	return nil
}

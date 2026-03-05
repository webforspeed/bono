package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	core "github.com/webforspeed/bono-core"
)

type SlashCommandSpec struct {
	Name        string
	Description string
	Handler     func(*Model, string) tea.Cmd
}

const helpText = `Available commands:
  /init              - Run exploring agent
  /index             - Index codebase for semantic code search
  /help              - Show this help
  /clear             - Clear chat history
  /model             - Show current model
  /reasoning <level> - Set reasoning effort (xhigh/high/medium/low/minimal/none)
  /spinner           - Cycle to next spinner style
  /spinner <type>    - Set spinner (dot, line, minidot, jump, pulse, points, globe, moon, monkey, meter, hamburger, ellipsis)
  /exit              - Exit Bono`

func DefaultSlashCommandSpecs() []SlashCommandSpec {
	return []SlashCommandSpec{
		{Name: "init", Description: "Run exploring agent", Handler: handleInit},
		{Name: "index", Description: "Index codebase for semantic search", Handler: handleIndex},
		{Name: "help", Description: "Show available commands", Handler: handleHelp},
		{Name: "clear", Description: "Clear the chat history", Handler: handleClear},
		{Name: "model", Description: "Switch AI model", Handler: handleModel},
		{Name: "reasoning", Description: "Set reasoning effort level", Handler: handleReasoning},
		{Name: "spinner", Description: "Change spinner style", Handler: handleSpinner},
		{Name: "exit", Description: "Exit Bono", Handler: handleExit},
	}
}

func slashCommandList(specs []SlashCommandSpec) []SlashCommand {
	commands := make([]SlashCommand, 0, len(specs))
	for _, spec := range specs {
		commands = append(commands, SlashCommand{Name: spec.Name, Description: spec.Description})
	}
	return commands
}

func slashCommandIndex(specs []SlashCommandSpec) map[string]SlashCommandSpec {
	index := make(map[string]SlashCommandSpec, len(specs))
	for _, spec := range specs {
		index[strings.ToLower(spec.Name)] = spec
	}
	return index
}

func handleInit(m *Model, arg string) tea.Cmd {
	return m.runExploringPreTask()
}

func handleHelp(m *Model, arg string) tea.Cmd {
	m.AppendRawMessage(helpText)
	m.input.Reset()
	return nil
}

func handleClear(m *Model, arg string) tea.Cmd {
	m.ClearMessages()
	m.agent.Reset()
	m.agent.ResetCost()
	m.sidebar.SetContextUsage(0)
	m.sidebar.SetTotalCost(0)
	m.input.Reset()
	return nil
}

func handleModel(m *Model, arg string) tea.Cmd {
	if len(m.modelModal.models) == 0 {
		m.AppendRawMessage("No models available. Edit tui/model_catalog.go to configure models.")
		m.input.Reset()
		return nil
	}
	m.AppendRawMessage("● /model")
	m.input.Reset()
	m.modelModal.Show()
	m.recalculateLayout()
	return nil
}

func handleReasoning(m *Model, arg string) tea.Cmd {
	arg = strings.TrimSpace(strings.ToLower(arg))

	// With argument: set directly without modal.
	if arg != "" {
		valid := map[string]bool{"xhigh": true, "high": true, "medium": true, "low": true, "minimal": true, "none": true}
		if !valid[arg] {
			m.AppendRawMessage("  Invalid reasoning effort. Use: xhigh, high, medium, low, minimal, none")
			m.input.Reset()
			return nil
		}
		if arg == "none" {
			m.agent.SetReasoningEffort("")
			m.sidebar.SetReasoningEffort("")
			m.AppendRawMessage("  Reasoning effort: disabled")
		} else {
			m.agent.SetReasoningEffort(arg)
			m.sidebar.SetReasoningEffort(arg)
			m.AppendRawMessage(fmt.Sprintf("  Reasoning effort: %s", arg))
		}
		m.input.Reset()
		return nil
	}

	// No argument: show modal picker.
	m.AppendRawMessage("● /reasoning")
	m.input.Reset()
	m.reasoningModal.Show(m.agent.ReasoningEffort())
	m.recalculateLayout()
	return nil
}

func handleSpinner(m *Model, arg string) tea.Cmd {
	if strings.TrimSpace(arg) == "" {
		// Cycle to next spinner
		m.NextSpinnerType()
		typeName := SpinnerTypeNames[m.GetSpinnerType()]
		m.AppendRawMessage("Spinner changed to: " + typeName)
		m.input.Reset()
		return nil
	}

	// Set specific spinner type
	newType := ParseSpinnerType(arg)
	m.SetSpinnerType(newType)
	typeName := SpinnerTypeNames[m.GetSpinnerType()]
	m.AppendRawMessage("Spinner set to: " + typeName)
	m.input.Reset()
	return nil
}

func handleExit(m *Model, arg string) tea.Cmd {
	m.input.Reset()
	return tea.Quit
}

func handleIndex(m *Model, arg string) tea.Cmd {
	if m.processing {
		return nil
	}

	m.AppendRawMessage("● /index")

	codeSearchService := m.agent.CodeSearchService()
	if codeSearchService == nil {
		m.AppendRawMessage("  ↳ Code search engine not initialized. Check configuration.")
		m.input.Reset()
		return nil
	}

	m.input.Reset()
	m.processing = true
	m.spinnerBar.SetText("Indexing codebase...")
	m.spinnerBar.SetActive(true)

	ctx := m.ctx
	prog := m.program

	return tea.Batch(
		m.spinnerBar.Tick(),
		func() tea.Msg {
			stats, err := codeSearchService.CodeSearchIndex(ctx, ".", core.CodeSearchIndexOptions{},
				func(p core.CodeSearchIndexProgress) {
					if prog != nil {
						prog.Send(IndexProgressMsg{
							Phase:      p.Phase,
							FilesDone:  p.FilesDone,
							FilesTotal: p.FilesTotal,
						})
					}
				},
			)
			return IndexDoneMsg{
				Err:         err,
				TotalFiles:  stats.TotalFiles,
				TotalChunks: stats.TotalChunks,
				Duration:    stats.Duration.Seconds(),
			}
		},
	)
}

func (m *Model) runExploringPreTask() tea.Cmd {
	if m.processing {
		return nil
	}

	m.input.Reset()
	m.processing = true
	m.spinnerBar.SetText("Running exploring agent...")
	m.spinnerBar.SetActive(true)

	agent := m.agent
	ctx := m.ctx
	return tea.Batch(
		m.spinnerBar.Tick(),
		func() tea.Msg {
			err := agent.RunPreTask(ctx, core.DefaultExploringTask())
			return AgentPreTaskDoneMsg{Err: err}
		},
	)
}

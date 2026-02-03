package tui

import (
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
  /init           - Run exploring agent
  /help           - Show this help
  /clear          - Clear chat history
  /model          - Show current model
  /context        - Show context info
  /spinner        - Cycle to next spinner style
  /spinner <type> - Set spinner (dot, line, minidot, jump, pulse, points, globe, moon, monkey, meter, hamburger, ellipsis)
  /exit           - Exit Bono`

func DefaultSlashCommandSpecs() []SlashCommandSpec {
	return []SlashCommandSpec{
		{Name: "init", Description: "Run exploring agent", Handler: handleInit},
		{Name: "help", Description: "Show available commands", Handler: handleHelp},
		{Name: "clear", Description: "Clear the chat history", Handler: handleClear},
		{Name: "model", Description: "Switch AI model", Handler: handleModel},
		{Name: "context", Description: "Show context window info", Handler: handleContext},
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
	m.input.Reset()
	return nil
}

func handleModel(m *Model, arg string) tea.Cmd {
	m.AppendRawMessage("Model info: (dynamic info coming soon)")
	m.input.Reset()
	return nil
}

func handleContext(m *Model, arg string) tea.Cmd {
	m.AppendRawMessage("Context info: (dynamic info coming soon)")
	m.input.Reset()
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

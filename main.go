package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	core "github.com/webforspeed/bono-core"
	"github.com/webforspeed/bono/prompts"
	"github.com/webforspeed/bono/tui"
)

func main() {
	loadEnv()

	config := core.Config{
		APIKey:       os.Getenv("OPENROUTER_API_KEY"),
		BaseURL:      os.Getenv("BASE_URL"),
		Model:        os.Getenv("MODEL"),
		SystemPrompt: prompts.System,
		HTTPTimeout:  30 * time.Second,
		PreTasks:     []core.PreTaskConfig{core.DefaultExploringTask()},
	}

	// Validate API key early
	if config.APIKey == "" {
		fmt.Println("Error: OPENROUTER_API_KEY required")
		os.Exit(1)
	}

	// Load tools
	toolsData, err := os.ReadFile("tools.json")
	if err != nil {
		fmt.Printf("Error loading tools.json: %v\n", err)
		os.Exit(1)
	}
	json.Unmarshal(toolsData, &config.Tools)

	// Create agent
	agent, err := core.NewAgent(config)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Create context
	ctx := context.Background()

	// Create TUI model
	model := tui.New(agent, ctx)

	// Create Bubble Tea program (use alt screen for full viewport)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Set up agent hooks to send messages to TUI
	agent.OnToolCall = func(name string, args map[string]any) bool {
		// Skip display for run_shell - subagent hooks handle it
		if name == "run_shell" {
			return true // auto-approve, subagent will ask for each command
		}

		if name == "read_file" {
			// Auto-approve reads
			p.Send(tui.AgentToolCallMsg{Name: name, Args: args, Approved: nil})
			return true
		}

		// Create channel and wait for TUI approval
		approved := make(chan bool, 1)
		p.Send(tui.AgentToolCallMsg{Name: name, Args: args, Approved: approved})

		// Block until user approves (Enter) or rejects (Esc), or context cancelled
		select {
		case result := <-approved:
			return result
		case <-ctx.Done():
			return false
		}
	}

	agent.OnToolDone = func(name string, args map[string]any, result core.ToolResult) {
		// Skip for run_shell - subagent hooks handle it
		if name == "run_shell" {
			return
		}
		p.Send(tui.AgentToolDoneMsg{Name: name, Args: args, Status: result.Status})
	}

	agent.OnMessage = func(content string) {
		p.Send(tui.AgentMessageMsg(content))
	}

	agent.OnPreTaskStart = func(name string) {
		p.Send(tui.AgentPreTaskStartMsg(name))
	}

	agent.OnPreTaskEnd = func(name string) {
		p.Send(tui.AgentPreTaskEndMsg(name))
	}

	agent.OnShellSubagentStart = func(systemPrompt string) {
		p.Send(tui.AgentShellSubagentStartMsg(systemPrompt))
	}

	agent.OnShellSubagentEnd = func(result core.ToolResult) {
		p.Send(tui.AgentShellSubagentEndMsg{Status: result.Status})
	}

	agent.OnSubagentToolCall = func(name string, args map[string]any, meta *core.ExecMeta) bool {
		// Check if this is a sandboxed execution notification (no approval needed)
		if meta != nil && meta.Sandboxed && !meta.SandboxError {
			// Sandboxed execution - just notify TUI, auto-approve
			p.Send(tui.AgentSubagentToolCallMsg{
				Name:      name,
				Args:      args,
				Sandboxed: true,
			})
			return true
		}

		// Check if sandbox blocked and needs fallback approval
		if meta != nil && meta.SandboxError {
			approved := make(chan bool, 1)
			p.Send(tui.AgentSubagentToolCallMsg{
				Name:          name,
				Args:          args,
				Approved:      approved,
				SandboxErr:    true,
				SandboxReason: meta.SandboxReason,
			})
			select {
			case result := <-approved:
				return result
			case <-ctx.Done():
				return false
			}
		}

		// Non-sandboxed execution - require approval
		approved := make(chan bool, 1)
		p.Send(tui.AgentSubagentToolCallMsg{Name: name, Args: args, Approved: approved})

		// Block until user approves (Enter) or rejects (Esc), or context cancelled
		select {
		case result := <-approved:
			return result
		case <-ctx.Done():
			return false
		}
	}

	agent.OnSubagentToolDone = func(name string, args map[string]any, result core.ToolResult) {
		sandboxed := false
		if result.ExecMeta != nil {
			sandboxed = result.ExecMeta.Sandboxed
		}
		p.Send(tui.AgentSubagentToolDoneMsg{
			Name:      name,
			Args:      args,
			Status:    result.Status,
			Sandboxed: sandboxed,
		})
	}

	agent.OnSandboxFallback = func(command string, reason string) bool {
		approved := make(chan bool, 1)
		p.Send(tui.AgentSandboxFallbackMsg{
			Command:  command,
			Reason:   reason,
			Approved: approved,
		})

		// Block until user approves (Enter) or rejects (Esc), or context cancelled
		select {
		case result := <-approved:
			return result
		case <-ctx.Done():
			return false
		}
	}

	// Run the TUI
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func loadEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			os.Setenv(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
}

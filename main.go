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
		if name == "read_file" {
			// Auto-approve reads
			p.Send(tui.AgentToolCallMsg{Name: name, Args: args})
			return true
		}

		if name == "run_shell" || name == "python_runtime" {
			// Sandboxed commands auto-approve
			if core.IsSandboxEnabled() {
				p.Send(tui.AgentToolCallMsg{Name: name, Args: args, Sandboxed: true})
				return true
			}
			// Non-sandboxed requires approval
			approved := make(chan bool, 1)
			p.Send(tui.AgentToolCallMsg{Name: name, Args: args, Approved: approved})
			select {
			case result := <-approved:
				return result
			case <-ctx.Done():
				return false
			}
		}

		// Other tools (write_file, edit_file) require approval
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
		sandboxed := false
		if result.ExecMeta != nil {
			sandboxed = result.ExecMeta.Sandboxed
		}
		p.Send(tui.AgentToolDoneMsg{Name: name, Args: args, Status: result.Status, Sandboxed: sandboxed})
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

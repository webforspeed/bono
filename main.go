package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	core "github.com/webforspeed/bono-core"
	"github.com/webforspeed/bono/prompts"
	"github.com/webforspeed/bono/tui"
)

const systemPromptVersion = "v1.0.5"

func main() {
	loadEnv()

	cwd, _ := os.Getwd()
	if cwd == "" {
		cwd = "."
	}
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("LOGNAME")
	}

	promptCtx := prompts.PromptContext{
		HostContext: prompts.HostContext{
			CWD:      cwd,
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Username: username,
			DateTime: time.Now().Format("2006-01-02 15:04:05 MST"),
		},
		Identity: prompts.AgentIdentity{
			Role:     "coding",
			Platform: "terminal",
		},
	}

	systemPrompt, err := prompts.LoadSystemPromptVersion(promptCtx, systemPromptVersion)
	if err != nil {
		fmt.Printf("Error loading system prompt version %q: %v\n", systemPromptVersion, err)
		os.Exit(1)
	}

	// Load model catalog
	models, err := tui.LoadModelCatalog("models.json")
	if err != nil {
		// Non-fatal: catalog is optional, fall back to config default
		models = nil
	}

	// Model priority: MODEL env var (deprecated) > first model in catalog > bono-core default
	model := os.Getenv("MODEL")
	if model == "" && len(models) > 0 {
		model = models[0].ID
	}

	config := core.Config{
		APIKey:       os.Getenv("OPENROUTER_API_KEY"),
		BaseURL:      os.Getenv("BASE_URL"),
		Model:        model,
		SystemPrompt: systemPrompt,
		HTTPTimeout:  120 * time.Second,
	}
	if n := os.Getenv("API_TIMEOUT_SEC"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			config.HTTPTimeout = time.Duration(v) * time.Second
		}
	}
	if n := os.Getenv("MAX_TOOL_CALLS_PER_TURN"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v >= 0 {
			config.MaxToolCallsPerTurn = v
		}
	} else {
		config.MaxToolCallsPerTurn = 50
	}

	// Validate API key early
	if config.APIKey == "" {
		fmt.Println("Error: OPENROUTER_API_KEY required")
		os.Exit(1)
	}

	// Create agent
	agent, err := core.NewAgent(config)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Create context
	ctx := context.Background()

	// Create TUI model
	tuiModel := tui.NewWithOptions(agent, ctx, tui.SpinnerDot, models)

	// Create Bubble Tea program (use alt screen for full viewport)
	p := tea.NewProgram(tuiModel, tea.WithAltScreen())

	// Set up agent hooks to send messages to TUI
	agent.OnToolCall = func(name string, args map[string]any) bool {
		if name == "read_file" || name == "compact_context" {
			// Auto-approve reads and context compaction
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

	agent.OnContextUsage = func(pct float64, totalCost float64) {
		p.Send(tui.AgentContextUsageMsg{Pct: pct, TotalCost: totalCost})
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

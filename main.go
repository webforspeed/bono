package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/glamour"
	core "github.com/webforspeed/bono-core"
	"github.com/webforspeed/bono/prompts"
	"golang.org/x/term"
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
		panic("OPENROUTER_API_KEY required")
	}

	// Load tools
	toolsData, err := os.ReadFile("tools.json")
	if err != nil {
		panic(err)
	}
	json.Unmarshal(toolsData, &config.Tools)

	// Create agent
	agent, err := core.NewAgent(config)
	if err != nil {
		panic(err)
	}

	// Set up TUI hooks
	renderer, _ := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(0))

	agent.OnToolCall = func(name string, args map[string]any) bool {
		if name == "read_file" {
			fmt.Printf("● Read('%s') ", args["path"])
			return true // auto-approve reads
		}
		fmt.Printf("● %s [Enter/Esc] ", formatTool(name, args))
		return getch() != 0x1b
	}

	agent.OnToolDone = func(name string, args map[string]any, result core.ToolResult) {
		prompt := formatToolDone(name, args)
		fmt.Printf("\r● %s => %s%s\n", prompt, result.Status, strings.Repeat(" ", 40))
	}

	agent.OnMessage = func(content string) {
		out, _ := renderer.Render(content)
		fmt.Print(out)
	}

	// Signal handling
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	go func() {
		<-sigChan
		cancel()
		fmt.Println("\nSee you later, alligator!")
		os.Exit(0)
	}()

	// Main loop (TUI controls the loop, not core)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if input == "" {
			continue
		}
		if _, err := agent.Chat(ctx, input); err != nil {
			fmt.Println("Error:", err)
		}
	}
}

func formatTool(name string, args map[string]any) string {
	switch name {
	case "read_file":
		return fmt.Sprintf("Read('%s')", args["path"])
	case "write_file":
		content, _ := args["content"].(string)
		lines := len(strings.Split(content, "\n"))
		return fmt.Sprintf("Write('%s', %d lines)", args["path"], lines)
	case "edit_file":
		return fmt.Sprintf("Edit('%s')", args["path"])
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

func formatToolDone(name string, args map[string]any) string {
	// Reuse formatTool to keep the full prompt (including description)
	return formatTool(name, args)
}

func getch() byte {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	b := make([]byte, 1)
	os.Stdin.Read(b)
	return b[0]
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

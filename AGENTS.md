# AGENTS.md

## Project Map

**Entry:** `main.go` - CLI entry point, config setup, agent hooks  
**TUI:** `tui/` - Bubble Tea terminal UI components  
**Core:** `github.com/webforspeed/bono-core` - external library providing agent loop, API client, and tool execution

### Structure (grouped)

**Entry & Config**
- `main.go` - entry point, config loading, agent hooks (OnToolCall, OnToolDone, OnMessage, OnPreTask*)
- `prompts/prompts.go` - system prompt configuration
- `tools.json` - tool definitions (read_file, write_file, edit_file, run_shell)
- `.env` / `.env.example` - environment configuration (API keys, model selection)

**TUI Package** (`tui/`)
- `model.go` - main Model struct, initialization, slash command handling
- `update.go` - Update function, key handling, tool formatting, terminal garbage filtering
- `view.go` - View rendering, layout composition
- `messages.go` - message types (AgentMessageMsg, AgentToolCallMsg, AgentToolDoneMsg, etc.)
- `input.go` - InputBox component
- `spinner.go` - SpinnerBar component with multiple spinner styles
- `statusbar.go` - StatusBar component
- `slashmodal.go` - SlashModal component for command autocomplete
- `styles.go` - centralized styles (DefaultStyles)

**Build Artifacts**
- `go.mod` / `go.sum` - Go module dependencies
- `bono` - compiled binary

### Conventions

- System prompts → `prompts/prompts.go` as exported constants
- Tool definitions → `tools.json` (JSON schema format)
- Configuration → environment variables via `.env` file
- Core agent logic → external `bono-core` library (not in this repo)
- TUI follows Bubble Tea patterns: Model, Update, View
- TUI components are composable (InputBox, SpinnerBar, StatusBar, SlashModal)

### Finding things

- Agent behavior/personality → `prompts/prompts.go`
- Available tools → `tools.json`
- Agent hooks setup → `main.go` (OnToolCall, OnToolDone, OnMessage, OnPreTaskStart, OnPreTaskEnd)
- TUI model & slash commands → `tui/model.go`
- Key handling & tool approval → `tui/update.go`
- Message types for TUI ↔ agent → `tui/messages.go`
- Visual rendering → `tui/view.go`
- Component styling → `tui/styles.go`
- Configuration → `.env` or `.env.example`
- Core agent loop → external `bono-core` package

## Rules

### Always

- Run `go build .` after changes to test compilation
- Set `OPENROUTER_API_KEY` environment variable before running
- Use `.env` file for local configuration
- For local bono-core development, use `replace` directive in `go.mod`
- Follow Bubble Tea patterns in `tui/` package (Model, Update, View)

### Never

- Don't commit `.env` file with real API keys
- Don't modify `bono` binary directly (it's compiled output)
- Don't change tool schemas in `tools.json` without updating corresponding handlers in `bono-core`
- Don't use magic constants for UI sizing - derive dimensions dynamically from styles and content
- Don't put TUI logic in `main.go` - keep it in the `tui/` package

### Style

- Go standard formatting (`go fmt`)
- System prompts use backtick-delimited multi-line strings
- Tool definitions follow OpenAI function calling JSON schema
- Keep `main.go` minimal - just config, hooks setup, and program launch
- TUI components should be self-contained with their own state and update methods
- Use `tui/messages.go` for all custom message types
- Use `tui/styles.go` for centralized styling

### When unsure

- Check `README.md` for usage and installation instructions
- Review `bono-core` library for agent loop implementation details
- Check `.env.example` for required environment variables
- Ask before modifying system prompts or tool definitions
- Check existing TUI components in `tui/` for patterns to follow

# AGENT.md

## Project Map

**Entry:** `main.go` - CLI entry point, TUI setup, and main loop  
**Core:** `github.com/webforspeed/bono-core` - external library providing agent loop, API client, and tool execution

### Structure (grouped)

- `main.go` - entry point, TUI hooks, tool confirmations, signal handling
- `prompts/prompts.go` - system prompt configuration
- `tools.json` - tool definitions (read_file, write_file, edit_file, run_shell)
- `.env` / `.env.example` - environment configuration (API keys, model selection)
- `go.mod` / `go.sum` - Go module dependencies
- `bono` - compiled binary

### Conventions

- System prompts → `prompts/prompts.go` as exported constants
- Tool definitions → `tools.json` (JSON schema format)
- Configuration → environment variables via `.env` file
- Core agent logic → external `bono-core` library (not in this repo)

### Finding things

- Agent behavior/personality → `prompts/prompts.go`
- Available tools → `tools.json`
- TUI formatting/hooks → `main.go` (OnToolCall, OnToolDone, OnMessage)
- Configuration → `.env` or `.env.example`
- Core agent loop → external `bono-core` package

## Rules

### Always

- Run `go build .` after changes to test compilation
- Set `OPENROUTER_API_KEY` environment variable before running
- Use `.env` file for local configuration
- For local bono-core development, use `replace` directive in `go.mod`

### Never

- Don't commit `.env` file with real API keys
- Don't modify `bono` binary directly (it's compiled output)
- Don't change tool schemas in `tools.json` without updating corresponding handlers in `bono-core`

### Style

- Go standard formatting (`go fmt`)
- System prompts use backtick-delimited multi-line strings
- Tool definitions follow OpenAI function calling JSON schema
- Keep TUI hooks simple - core logic lives in `bono-core`

### When unsure

- Check `README.md` for usage and installation instructions
- Review `bono-core` library for agent loop implementation details
- Check `.env.example` for required environment variables
- Ask before modifying system prompts or tool definitions

# Bono

Terminal-based autonomous coding agent for reading, writing, and editing files directly from your command line.

## Features

- [x] Autonomous agent loop with tool-call execution
- [x] Built‑in file tools (read/write/edit)
- [x] Shell execution with user approvals
- [ ] Worktree‑isolated runs (disposable workspaces)
- [ ] Sandboxed execution (FS/command restrictions)
- [ ] Agent config files (AGENT.md, CLAUDE.md)
- [ ] Skills system (prompt/tool bundles)
- [ ] MCP tool integrations
- [ ] Slash commands (built-in + user-defined)
- [ ] Plan mode (preview and approve steps before execution)
- [ ] Sub-agents (delegate subtasks to child agents)
- [ ] Parallel tool execution (concurrent tool calls)

## Dependencies

This CLI is built on [bono-core](https://github.com/webforspeed/bono-core), which provides the autonomous agent loop, API client, and tool execution.

## Installation

```bash
go install github.com/webforspeed/bono@latest
```

Or build from source:

```bash
git clone https://github.com/webforspeed/bono.git
cd bono
go build .
```

## Configuration

Create a `.env` file in your working directory:

```env
# .env.example
OPENROUTER_API_KEY=your-api-key-here
BASE_URL=https://openrouter.ai/api/v1
MODEL=anthropic/claude-opus-4.5
```

Or set environment variables directly in your shell.

## Usage

```bash
./bono
```

Then type your prompts at the `>` prompt.

### Tool Confirmations

| Tool | Confirmation |
|------|--------------|
| `read_file` | Auto-approved |
| `write_file` | Enter to confirm, Esc to cancel |
| `edit_file` | Enter to confirm, Esc to cancel |
| `run_shell` | Enter to confirm, Esc to cancel |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Approve tool execution |
| `Esc` | Cancel tool execution |
| `Ctrl+C` | Exit the agent |

## Local Development

To develop bono and bono-core together, add a replace directive to `go.mod`:

```go
replace github.com/webforspeed/bono-core => ../bono-core
```

Then run:

```bash
go mod tidy
go build .
```

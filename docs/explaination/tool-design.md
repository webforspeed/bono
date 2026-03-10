# Tool Design in Bono

## Architecture

Tools are defined in Go code in **bono-core**, not in JSON files. Each tool is a `*ToolDef` — a single struct that co-locates schema, execution, and approval policy. There are no special tool types or sub-interfaces.

```
ToolDef {
    Name        — stable identifier
    Description — model-facing policy
    Parameters  — JSON Schema for the API
    Execute     — func(args) ToolResult
    AutoApprove — func(sandboxed) bool
}
```

## Registry

Tools are held in a `Registry` — an instance-based map, not a global. Each agent gets its own registry in `NewAgent()`.

The registry provides:
- `Register(t)` — add a tool
- `Get(name)` — look up by name for dispatch
- `Tools(names...)` — return API-ready `[]Tool` for the LLM. Empty = all tools.

## Lifecycle

```
NewAgent(config)
  ├─ create Registry
  ├─ register all tools (injecting dependencies via closures)
  └─ resolve apiTools = registry.Tools(config.AllowedTools...)

Chat(input)
  ├─ send apiTools to LLM via ChatCompletionWithTools
  ├─ LLM responds with ToolCalls
  └─ for each tool call:
       ├─ OnToolCall hook (bono session decides approval)
       ├─ registry.Get(name).Execute(args)   ← uniform, one path
       ├─ OnToolDone hook (bono emits session events)
       └─ append ToolResult as "tool" message
```

Every tool goes through the same `tool.Execute(args)` dispatch. No name-based switches.

## AllowedTools Filtering

`Config.AllowedTools` controls which tools the LLM sees. The registry always holds all registered tools — filtering only affects what gets sent to the API.

```go
// All tools (default)
config.AllowedTools = nil

// Restricted set (for a specific frontend or deployment mode)
config.AllowedTools = []string{"read_file", "edit_file", "run_shell"}
```

## Dependency Injection

Tools that need nothing (read_file, write_file, edit_file) take no constructor parameters.

Tools that need agent capabilities take function parameters — the agent provides closures:

- `RunShellTool(exec)` and `PythonRuntimeTool(exec)` receive a shell executor that wraps sandbox + fallback
- `CompactContextTool(compact)` receives the agent's `compactMessages` method

This keeps all tools uniform from the registry's perspective while allowing per-tool dependencies.

## Approval Model

Each tool declares its default policy via `AutoApprove(sandboxed bool)`:

| Tool | AutoApprove |
|------|-------------|
| `read_file`, `compact_context`, `enter_plan_mode` | always true |
| `run_shell`, `python_runtime` | true when sandboxed |
| `write_file`, `edit_file` | always false |

Bono's session layer in `internal/session/session.go` makes the final decision — it can override AutoApprove for frontend-specific behavior such as TUI `Enter/Esc` approval or headless `[y/N]` prompts.

## Design Intent

- Prefer a small set of composable primitives over many specialized tools.
- Keep schemas strict and explicit so failures are obvious.
- Push guardrails into runtime nudges and approval boundaries rather than long static descriptions.
- Fail fast on invalid config or unknown tools; avoid silent fallback behavior.

## Practical Reading Order

1. `bono-core/types.go` (`ToolDef` struct)
2. `bono-core/registry.go`
3. `bono-core/tool_*.go` (any one tool file for the pattern)
4. `bono-core/agent.go` (`NewAgent` registry setup + `Chat` dispatch)
5. `bono/internal/session/session.go` (`OnToolCall`, `OnToolDone`, `OnSandboxFallback`)
6. `bono/internal/session/display.go` (shared tool formatting)

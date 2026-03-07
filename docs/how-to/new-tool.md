# How to Add a New Tool

## Where Things Live

All tool definitions live in **bono-core**. Each tool is a `*ToolDef` — same struct, same interface, no exceptions.

| What | Where |
|------|-------|
| Tool definition + execution | `bono-core/tool_<name>.go` |
| Tool registration | `bono-core/agent.go` → `NewAgent()` |
| Approval policy | `bono/internal/session/session.go` |
| Display formatting | `bono/internal/session/display.go` → `FormatTool()` |

## Steps

### 1. Create the tool file in bono-core

Create `tool_<name>.go` with a constructor that returns `*ToolDef`:

```go
func MyTool() *ToolDef {
    return &ToolDef{
        Name:        "my_tool",
        Description: "...",
        Parameters:  map[string]any{ ... },
        Execute: func(args map[string]any) ToolResult {
            // implementation
        },
        AutoApprove: func(sandboxed bool) bool {
            return false // or true for safe read-only tools
        },
    }
}
```

Co-locate the execution logic (e.g. `ExecuteMyTool()`) in the same file.

### 2. Register in `NewAgent()`

In `bono-core/agent.go`, add the registration alongside the others:

```go
a.registry.Register(MyTool())
```

### 3. Add display formatting in bono

In `bono/internal/session/display.go`, add a case to `FormatTool()` so all frontends share the same readable one-liner for this tool.

### 4. Verify

Build both repos. Run bono and confirm one successful call and one failure path.

## Tools That Need Dependencies

If the tool needs agent capabilities (shell execution, message history, etc.), the constructor takes a function parameter:

```go
func MyTool(doSomething func(input string) ToolResult) *ToolDef {
    return &ToolDef{
        ...
        Execute: func(args map[string]any) ToolResult {
            return doSomething(args["input"].(string))
        },
    }
}
```

The agent wires the dependency in `NewAgent()`:

```go
a.registry.Register(MyTool(func(input string) ToolResult {
    // has access to agent state via closure over `a`
    return ...
}))
```

Closures capture the agent pointer — hooks set after `NewAgent()` returns (like `OnSandboxFallback`) are evaluated at call time, so they pick up final values.

See `RunShellTool(exec)` and `CompactContextTool(compact)` for working examples.

## Constraints

- Tool names are stable identifiers — changing a name is a breaking change for conversation history.
- `AutoApprove` controls the default policy. Bono's session layer can override per frontend or runtime mode.
- `Description` is policy for the model — keep it about intent and constraints, not implementation details.
- Parameters use JSON Schema as `map[string]any`. Match existing tools for consistency.
- Prefer composable primitives over specialized tools. A new tool should earn its place.

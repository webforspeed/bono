## Context

bono-core currently uses `PreTaskConfig` (a flat struct) and `RunPreTask` (a hardcoded loop in `agent.go`) for subagent-like work. The exploring agent (`/init`) is the only user. This approach doesn't scale: no interface abstraction, no tool filtering, no prompt versioning, and the agent loop logic is tangled with the main agent.

bono's prompt system (`prompts/versions/*.tmpl` with `embed.FS`) already demonstrates the versioned template pattern. Subagent prompts should follow the same pattern but scoped per subagent in bono-core.

The TUI already has a middleware-based `SessionFrontend` chain and event-driven architecture. Subagent events should flow through the same system — the TUI doesn't need to know what kind of subagent is running.

## Goals / Non-Goals

**Goals:**
- Define a `SubAgent` interface in bono-core that any subagent implements
- Registry pattern for registering/looking up subagents by name
- Tool filtering so subagents only access their declared tools
- Versioned prompt templates per subagent (`subagent_prompts/<name>/*.tmpl`)
- Built-in subagents auto-register in `NewAgent()` — consumers of bono-core get everything
- `/plan <task>` as the first subagent built on this system
- TUI stays a dumb frontend — dispatches command, renders events
- Same subagent system works for future frontends (web, IDE)

**Non-Goals:**
- Tab-to-switch-mode UX (future work)
- Migrating the existing `/init` exploring agent to the new system (can be a follow-up)
- Interactive plan editing or plan persistence
- Subagent chaining or composition

## Decisions

### 1. SubAgent interface in bono-core

```go
type SubAgent interface {
    Name() string
    AllowedTools() []string  // empty = all tools
    SystemPrompt() string
}
```

**Rationale**: Minimal interface — a subagent is a name, a prompt, and a tool allowlist. The runner handles execution mechanics (loop, tool dispatch, callbacks). This keeps subagent implementations trivial: define prompt + tools, register, done.

**Alternative considered**: Adding `Run(ctx, input) error` to the interface so each subagent owns its loop — rejected because the execution mechanics are identical across subagents. Only the prompt and tool set vary.

### 2. Runner in bono-core executes subagents

```go
func (a *Agent) RunSubAgent(ctx context.Context, sa SubAgent, input string) error
```

`RunSubAgent` is a subagent-aware execution loop:
- Builds isolated message history with `sa.SystemPrompt()`
- Filters `a.registry.Tools()` to only `sa.AllowedTools()` before each LLM call
- Rejects disallowed tool calls at runtime as a second check
- Uses existing callbacks (`OnToolCall`, `OnToolDone`, `OnMessage`, etc.)
- Fires `OnSubAgentStart(name)` / `OnSubAgentEnd(name)` callbacks

**Rationale**: The runner is the only place that knows about tool filtering and the agent loop. Subagent implementations don't need to reimplement this.

### 3. Versioned prompt templates in bono-core

```
bono-core/
├── subagent.go              // SubAgent interface
├── subagents.go             // Built-in implementations + registerBuiltinSubAgents()
└── subagent_prompts/
    └── plan/
        └── v1.0.0.tmpl      // planning system prompt
```

All subagent implementations live in the core package (`subagents.go`) to avoid circular imports. Prompts are embedded via `//go:embed subagent_prompts/<name>/*.tmpl`.

**Rationale**: Follows bono's existing `prompts/versions/*.tmpl` pattern. Keeping implementations in the core package avoids sub-package import cycles while still supporting versioned prompt files. Prompts are versioned in git and can be revised independently of code.

**Alternative considered**: Separate `subagents/<name>/` sub-packages — rejected due to circular import (sub-package imports `core` for the interface, `core` imports sub-package for registration).

### 4. Auto-registration in NewAgent()

```go
func (a *Agent) registerBuiltinSubAgents() {
    a.RegisterSubAgent(newPlanAgent())
}
```

Called at the end of `NewAgent()`. Any consumer of bono-core gets all built-in subagents automatically.

**Rationale**: The frontend (bono) should never import subagent-specific code. bono-core is the agent harness — it owns subagent logic. Future frontends (web, IDE) get the same subagents without any registration code.

**Alternative considered**: Frontend-side registration (`main.go` imports and registers each subagent) — rejected because it couples frontends to subagent internals.

### 5. Tool filtering at the runner level

When `RunSubAgent` calls the LLM, it passes only the tools in `sa.AllowedTools()` via `a.registry.Tools(allowed...)`. If the LLM somehow returns a tool call for a disallowed tool, the runner returns an error result for that call.

**Rationale**: Enforcement at the API call level (don't send disallowed tool schemas) is the strongest guarantee. Belt-and-suspenders with runtime rejection.

### 6. TUI handler appends parent/child lines synchronously

```go
func handlePlan(m *Model, arg string) tea.Cmd {
    m.AppendRawMessage("● /plan")
    if strings.TrimSpace(arg) == "" {
        m.AppendRawMessage("  ↳ Usage: /plan <task description>")
        m.input.Reset()
        return nil
    }
    m.AppendRawMessage("  ↳ Starting planning subagent...")
    return m.runSubAgent("plan", arg)
}
```

Both the parent line (`● /plan`) and child line (`↳ Starting planning subagent...`) are appended synchronously in the handler before the async subagent work begins. The `SubAgentStartMsg` from the async event is a no-op in update.go.

`runSubAgent` is a generic method that looks up the subagent by name and dispatches — reusable for any future `/review`, `/docs`, etc.

**Rationale**: Synchronous rendering ensures the parent/child lines are always visible before async work starts. Async events can arrive unpredictably relative to streaming content.

### 7. Event flow uses existing SessionFrontend

SubAgent lifecycle events (`SubAgentStartEvent`, `SubAgentEndEvent`) flow through the same `SessionFrontend.HandleEvent()` path. The headless frontend prints them. The TUI frontend maps them to Bubble Tea messages (though the start message is currently a no-op since the handler renders synchronously).

**Rationale**: No new communication channels needed. The middleware chain (`SynchronizedMiddleware`, etc.) applies automatically.

## Risks / Trade-offs

- **[Risk]** Adding `RunSubAgent` alongside `RunPreTask` creates two paths → **Mitigation**: Keep both for now. `RunPreTask` continues to work. Migrate `/init` to subagent system in a follow-up. Eventually deprecate `RunPreTask`.
- **[Risk]** Tool filtering may miss edge cases (e.g., tool name mismatches) → **Mitigation**: Filter at API schema level (don't send tool definitions). Runner validates tool calls against allowlist as a second check.
- **[Trade-off]** Subagent prompts live in bono-core, not bono → This is intentional. Subagent logic is core behavior, not frontend concern. All frontends share the same prompts.
- **[Trade-off]** All subagent implementations in one file (`subagents.go`) → Avoids circular imports. If the file grows too large, individual subagents can be split into separate files in the core package (e.g., `subagent_plan.go`).
- **[Trade-off]** No `Run()` method on interface means subagents can't customize execution → Keeps the system simple. If a subagent needs custom loop behavior in the future, the interface can be extended with an optional `Runner` interface.

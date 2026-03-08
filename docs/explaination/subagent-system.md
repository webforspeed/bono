# How Subagents Work in Bono

## Overview

Subagents are self-contained agent modes in bono-core, each with its own system prompt and tool constraints. The runner in bono-core handles execution; frontends (TUI, headless, future web/IDE) just render events.

## Architecture

```
bono (frontend)                    bono-core (agent harness)
┌──────────────┐                   ┌─────────────────────────┐
│ /plan <task>  │───dispatch───────▶│ Agent.RunSubAgent()     │
│              │                   │   ├─ SubAgent.SystemPrompt()
│ TUI renders  │◀──events──────────│   ├─ tool filtering      │
│ via frontend │                   │   └─ isolated msg loop   │
└──────────────┘                   └─────────────────────────┘
```

## Key Components

### SubAgent Interface (bono-core)

Minimal contract — a subagent is a name, a prompt, and a tool allowlist:

```go
type SubAgent interface {
    Name() string
    AllowedTools() []string  // empty = all tools
    SystemPrompt() string
}
```

### Registry on Agent

Subagents register on the `Agent` at startup via `agent.RegisterSubAgent(sa)`. Lookup is by name: `agent.SubAgent("plan")`.

### Runner: `Agent.RunSubAgent(ctx, sa, input)`

- Creates an **isolated message history** (does not touch the main conversation)
- Filters tool schemas to `sa.AllowedTools()` before sending to the LLM
- Rejects disallowed tool calls at runtime as a second check
- Uses existing callbacks (`OnToolCall`, `OnToolDone`, `OnMessage`, `OnContentDelta`, etc.)
- Fires `OnSubAgentStart(name)` / `OnSubAgentEnd(name)` for lifecycle events

### Versioned Prompt Templates

Subagent prompts live in bono-core alongside their implementations:

```
bono-core/
├── subagent.go              // SubAgent interface
├── subagents.go             // Built-in implementations + registerBuiltinSubAgents()
└── subagent_prompts/
    └── plan/
        └── v1.0.0.tmpl      // system prompt, embedded via //go:embed
```

This mirrors bono's `prompts/versions/*.tmpl` pattern. Prompts are versioned in git and can be revised independently of code changes.

## Event Flow

```
1. TUI: handlePlan() appends "● /plan" and "↳ Starting planning subagent..." synchronously
2. TUI: handlePlan() → m.runSubAgent("plan", task) — sets processing, starts spinner
3. Async: agent.RunSubAgent() fires OnSubAgentStart("plan")
4. session.Bind(): OnSubAgentStart → frontend.HandleEvent(SubAgentStartEvent)
5. TUI: SubAgentStartMsg → no-op (lines already rendered by handler)
6. [agent loop runs with tool calls and streaming]
7. bono-core: fires OnSubAgentEnd("plan")
8. TUI: SubAgentDoneMsg → deactivate spinner, reset processing
```

Parent/child lines are rendered synchronously in the slash command handler to guarantee they appear before any async streaming content.

## Design Decisions

- **Subagent logic lives in bono-core** — all frontends share the same subagents and prompts. Built-in subagents auto-register in `NewAgent()`, so any consumer of bono-core gets them for free
- **TUI is a dumb dispatcher** — `runSubAgent(name, input)` is generic, reusable for any subagent. Bono never imports subagent-specific packages
- **Tool filtering is enforced at two levels** — API schema filtering (don't send tools) + runtime rejection (belt-and-suspenders)
- **Runs alongside existing `RunPreTask`** — no breaking changes, migration is a follow-up

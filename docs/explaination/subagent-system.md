# How Subagents Work in Bono

## Overview

Subagents are self-contained agent modes in bono-core, each with its own system prompt and tool constraints. The runner in bono-core handles execution; frontends (TUI, headless, future web/IDE) just render events. Subagents support composable post-completion hooks for persistence, approval gates, and other cross-cutting behaviors.

## Architecture

```
bono (frontend)                    bono-core (agent harness)
┌──────────────┐                   ┌─────────────────────────┐
│ /plan <task>  │───dispatch───────▶│ Agent.RunSubAgent()     │
│              │                   │   ├─ SubAgent.SystemPrompt()
│ TUI renders  │◀──events──────────│   ├─ tool filtering      │
│ via frontend │                   │   ├─ isolated msg loop   │
└──────────────┘                   │   └─ hooks (persist, approve)
                                   └─────────────────────────┘
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

### SubAgentHook Interface (bono-core)

Cross-cutting post-completion behaviors, composable via registration:

```go
type SubAgentHook interface {
    AfterComplete(ctx context.Context, result *SubAgentResult) error
}
```

Hooks annotate `SubAgentResult.Meta` (e.g., `"output_path"`, `"approval"`). Built-in hooks:

- **`PersistHook(dirTemplate)`** — writes subagent output to `<dir>/<timestamp>-<slug>.md`. Supports `{cwd}` expansion. Overwrites the same file on revision.
- **`ApprovalHook(getCallback)`** — prompts user to approve, reject, or revise. Auto-approves when callback is nil (headless mode).

### Registry on Agent

Subagents register on the `Agent` at startup via `agent.RegisterSubAgent(sa, hooks...)`. Hooks are variadic — zero means no post-completion behaviors. Lookup is by name: `agent.SubAgent("plan")`.

### Runner: `Agent.RunSubAgent(ctx, sa, input) (*SubAgentResult, error)`

- Creates an **isolated message history** (does not touch the main conversation)
- Filters tool schemas to `sa.AllowedTools()` before sending to the LLM
- Rejects disallowed tool calls at runtime as a second check
- Uses existing callbacks (`OnToolCall`, `OnToolDone`, `OnMessage`, `OnContentDelta`, etc.)
- Fires `OnSubAgentStart(name)` / `OnSubAgentEnd(name)` for lifecycle events
- After the final response, runs registered hooks in order
- If a hook sets `Meta["approval"]="revise"`, appends user feedback to the isolated history and continues the LLM loop (revision cycle)
- Builds a handoff message injected into the main conversation, varying by `Meta["approval"]` and `Meta["output_path"]`

### Versioned Prompt Templates

Subagent prompts live in bono-core alongside their implementations:

```
bono-core/
├── subagent.go              // SubAgent interface, SubAgentHook, SubAgentResult
├── subagent_hooks.go        // PersistHook, ApprovalHook, slugify
├── subagents.go             // Built-in implementations + registerBuiltinSubAgents()
└── subagent_prompts/
    └── plan/
        └── v1.0.0.tmpl      // system prompt, embedded via //go:embed
```

This mirrors bono's `prompts/versions/*.tmpl` pattern. Prompts are versioned in git and can be revised independently of code changes.

## Event Flow

### Basic subagent (no hooks)

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

### Plan subagent (with PersistHook + ApprovalHook)

```
1–6.  Same as above
7.    bono-core: final response — PersistHook writes plan to ~/.bono/<cwd>/plans/<timestamp>-<slug>.md
8.    bono-core: ApprovalHook calls OnSubAgentApproval callback
9.    session.Bind(): OnSubAgentApproval → frontend.RequestSubAgentApproval()
10.   TUI: AgentPlanApprovalMsg → shows file path + "Press Enter to implement, Esc to skip" prompt
11a.  [Enter] → approve: Meta["approval"]="approved", RunSubAgent returns
11b.  [Esc] → reject: Meta["approval"]="rejected", RunSubAgent returns
11c.  [typed feedback] → revise: feedback appended to isolated history, LLM revises, hooks re-run (go to 7)
12.   bono-core: builds handoff message, fires OnSubAgentEnd("plan")
13.   TUI: SubAgentDoneMsg with Approved=true → auto-triggers agent.Chat("Implement the plan.")
```

Parent/child lines are rendered synchronously in the slash command handler to guarantee they appear before any async streaming content.

## Design Decisions

- **Subagent logic lives in bono-core** — all frontends share the same subagents and prompts. Built-in subagents auto-register in `NewAgent()`, so any consumer of bono-core gets them for free
- **TUI is a dumb dispatcher** — `runSubAgent(name, input)` is generic, reusable for any subagent. Bono never imports subagent-specific packages
- **Tool filtering is enforced at two levels** — API schema filtering (don't send tools) + runtime rejection (belt-and-suspenders)
- **Hooks are orthogonal to identity** — `SubAgent` stays minimal (name, prompt, tools). Persistence, approval, and future behaviors compose via `SubAgentHook` without touching the interface
- **Hook order matters** — PersistHook runs before ApprovalHook so the file path is available in the approval prompt
- **Approval auto-triggers implementation** — when the plan is approved, the TUI auto-calls `agent.Chat("Implement the plan.")` so the handoff flows directly into the main agent loop without requiring user input

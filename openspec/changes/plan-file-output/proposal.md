## Why

The plan mode subagent produces a plan and hands it off as a summary injected into the main conversation. This summary is ephemeral — lost on compaction or new sessions. Persisting plans to `~/.bono/<cwd>/plans/` makes them reviewable, shareable, and referenceable across sessions, scoped per project.

Today the plan subagent auto-returns to the main agent on completion. The user wants an approval gate: review the plan, then decide to implement, discard, or revise before handing off.

## What Changes

- `RunSubAgent` gains a composable `SubAgentHook` middleware pattern: hooks run after the subagent's final response, before the handoff message is built.
- A reusable `PersistHook(dirTemplate)` writes subagent output to `~/.bono/{cwd}/plans/<timestamp>-<slug>.md` using a deterministic harness-generated path.
- `RegisterSubAgent` accepts optional hooks — the plan subagent is registered with `PersistHook("~/.bono/{cwd}/plans")`.
- After the plan file is written, `RunSubAgent` enters an **approval loop** via a new `RequestSubAgentApproval` frontend method. Three outcomes:
  - **Enter** → approve: TUI auto-triggers `agent.Chat("Implement the plan.")` — handoff is already in context, main agent starts immediately
  - **Escape** → reject: plan file left on disk, return to main agent with no implementation instruction
  - **Type feedback** → revise: feedback sent back to the subagent, plan is revised, same file updated, approval re-prompted
- `RunSubAgent` returns `(*SubAgentResult, error)` so callers can inspect `Meta["approval"]` and act accordingly.
- No new tools are given to the plan subagent — the file write is a harness-level hook, not a tool call.

## Capabilities

### New Capabilities
- `plan-file-persist`: Composable `SubAgentHook` middleware for post-completion behaviors, with a `PersistHook` implementation that writes subagent output to project-scoped paths under `~/.bono/<cwd>/`.
- `plan-approval`: Interactive approval gate after plan creation with approve/reject/revise semantics, using the existing frontend approval pattern extended to support free-text feedback.

### Modified Capabilities

## Impact

- `bono-core/subagent.go` — new `SubAgentHook` interface, `SubAgentResult` struct
- `bono-core/subagent_hooks.go` — `PersistHook` implementation and slug helper
- `bono-core/agent.go` — `RunSubAgent` executes hooks after completion, supports approval loop via callback; `RegisterSubAgent` accepts variadic hooks
- `bono-core/subagents.go` — plan subagent registered with `PersistHook`
- `bono/internal/session/frontend.go` — new `ApprovalSubAgentPlan` kind with three-outcome response type
- `bono/tui/update.go` — TUI handling for plan approval (Enter/Esc/type feedback)
- `bono/tui/messages.go` — new message type for plan approval prompt

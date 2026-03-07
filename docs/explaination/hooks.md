# How Hooks Work in Bono

## Two Hook Layers

Bono has two distinct layers that work together:

**bono-core callbacks** — Low-level function fields on the `Agent` struct (`OnToolCall`, `OnToolDone`, `OnMessage`, etc.). These are the plumbing between the agent loop and Bono's session layer. They carry operational data and can influence control flow (e.g. `OnToolCall` returns `bool` to approve/reject).

**bono lifecycle hooks** — Higher-level event system in the `hooks/` package. These are observability primitives modeled after Claude Code's hook events. They fire at lifecycle boundaries and are fire-and-forget — they cannot block or alter execution.

```text
bono-core (agent loop)          bono (session + hooks + frontends)
─────────────────────           ───────────────────────────────────
Agent.OnToolCall ──────────────► internal/session.Session
                                  ├─ dispatcher.Fire(PreToolUse)
                                  ├─ dispatcher.Fire(PermissionRequest)  [if approval needed]
                                  ├─ frontend.HandleEvent(...)
                                  └─ frontend.RequestApproval(...)

Agent.OnToolDone ──────────────► internal/session.Session
                                  ├─ dispatcher.Fire(PostToolUse)        [if success]
                                  ├─ dispatcher.Fire(PostToolUseFailure) [if failure]
                                  └─ frontend.HandleEvent(...)
```

The lifecycle hooks ride on top of the bono-core callbacks. bono-core stays unaware of Bono's hook system and frontend implementations.

## Architecture

Three types in the `hooks/` package:

| Type | Role |
|------|------|
| `Event` | String constant identifying a lifecycle point |
| `Handler` | Interface with `Handle(ctx, event, payload)` — the primitive |
| `Dispatcher` | Registry that maps events to handlers and dispatches them |

`Handler` follows the `http.Handler` pattern — one method, easy to implement. `HandlerFunc` adapts plain functions.

Payloads are typed structs passed as `any`. Handlers type-assert to access event-specific fields.

## Event Lifecycle

```
Program start
  │
  ├─ SessionStart
  │
  ├─ [user types and presses Enter]
  │   └─ UserPromptSubmit
  │
  ├─ [agent.Chat runs]
  │   ├─ [for each tool call]
  │   │   ├─ PreToolUse
  │   │   ├─ PermissionRequest          [only if tool needs approval]
  │   │   └─ PostToolUse / PostToolUseFailure
  │   │
  │   └─ Stop                           [agent.Chat returns]
  │
  ├─ [repeat per prompt]
  │
  └─ SessionEnd
```

Events that fire once per session: `SessionStart`, `SessionEnd`.
Events that fire per prompt: `UserPromptSubmit`, `Stop`.
Events that fire per tool call: `PreToolUse`, `PostToolUse`/`PostToolUseFailure`, `PermissionRequest`.

## Flow of Control

Lifecycle hooks are fire-and-forget. They do not return values and cannot block the agent. Decision-making (approve/reject tool calls) stays in bono-core's `OnToolCall` bool return.

`Dispatcher.Fire` calls handlers sequentially. Panics are recovered and logged to stderr. A misbehaving handler cannot crash the agent.

## Structured Logging

`internal/logging` provides a `slog.Logger` factory writing JSON to `logs/bono.jsonl`. The default `LogHandler` in `hooks/` uses this logger to record every event.

To swap the logging backend, pass a different `slog.Handler` (e.g. `slog.NewTextHandler`, a third-party handler) to `slog.New()`.

## Practical Reading Order

1. `hooks/event.go` (event constants + payload structs)
2. `hooks/hooks.go` (Handler, Dispatcher)
3. `hooks/log_handler.go` (default handler)
4. `internal/logging/logging.go` (slog factory)
5. `main.go` (dispatcher setup + mode selection)
6. `internal/session/session.go` (callback wiring + prompt lifecycle)
7. `tui/model.go` (`UserPromptSubmit` for the interactive TUI path)

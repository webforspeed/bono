# How to Add a New Hook

## Where Things Live

All hook code lives in **bono** (not bono-core). The hook system is an observability layer on top of bono-core's callbacks.

| What | Where |
|------|-------|
| Event constants + payloads | `hooks/event.go` |
| Handler interface + Dispatcher | `hooks/hooks.go` |
| Default log handler | `hooks/log_handler.go` |
| Event firing (most events) | `main.go` |
| Event firing (TUI events) | `tui/model.go` |
| Handler registration | `main.go` (dispatcher setup block) |

## Steps

### 1. Define the event and payload

In `hooks/event.go`, add a constant and a payload struct:

```go
const MyNewEvent Event = "MyNewEvent"

type MyNewEventPayload struct {
    Relevant string
    Fields   int
}
```

### 2. Fire the event

Decide where the event should fire. Two options:

**From `main.go`** — if it corresponds to an agent callback or program lifecycle point:
```go
dispatcher.Fire(ctx, hooks.MyNewEvent, hooks.MyNewEventPayload{...})
```

**From `tui/model.go`** — if it originates from user interaction:
```go
if m.dispatcher != nil {
    m.dispatcher.Fire(m.ctx, hooks.MyNewEvent, hooks.MyNewEventPayload{...})
}
```

If the event needs to fire from a point in bono-core where no callback exists yet, add a new callback field to bono-core's `Agent` struct first, then fire the hook from bono's callback.

### 3. Register the log handler

In `main.go`, add the event to the dispatcher setup block:

```go
dispatcher.On(hooks.MyNewEvent, logHandler)
```

### 4. Verify

Build both repos. Run bono, trigger the event, and check `logs/bono.jsonl` for the entry.

## Deciding Where an Event Fires

| Origin | Fire from | Example |
|--------|-----------|---------|
| Agent loop (tool calls, responses) | `main.go` via bono-core callback | `PreToolUse`, `PostToolUse` |
| User interaction (input, navigation) | `tui/model.go` | `UserPromptSubmit` |
| Program lifecycle (start, exit) | `main.go` | `SessionStart`, `SessionEnd` |
| Infrastructure (indexing, batch review) | `main.go` or relevant manager | `Stop` |

## Constraints

- Hook events are fire-and-forget. Do not use them for control flow decisions.
- Payload structs use exported fields so `slog` can serialize them as JSON.
- If an event needs a bono-core callback that doesn't exist, add the callback to bono-core's `Agent` struct and wire it in `main.go`. The hook system stays in bono.

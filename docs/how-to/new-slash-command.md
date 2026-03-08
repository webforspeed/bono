# How to Create a New Slash Command

**Note:** If your slash command runs a subagent (a separate agent with its own system prompt and tool restrictions), see [How to add a new subagent slash command](./new-subagent-slash-command.md) instead.

## Where Things Live

Slash commands are a TUI concern in this repo.

| What | Where |
|------|-------|
| Command list + handlers | `tui/slash_commands.go` |
| Input routing (`/foo` -> handler) | `tui/model.go` -> `handleSlashCommand()` |
| Slash picker modal (`/` suggestions) | `tui/slashmodal.go` |
| Async command messages | `tui/messages.go` |
| Async UI updates | `tui/update.go` |

## Runbook

### 1. Register the command in `DefaultSlashCommandSpecs()`

Add a new `SlashCommandSpec` entry with `Name`, `Description`, and `Handler`.

Use a stable lowercase name. The name is both:
- the slash picker key (`/mycmd`)
- the parser key in `handleSlashCommand()`

### 2. Add the help text line

Update `helpText` in `tui/slash_commands.go` so `/help` stays accurate.

### 3. Implement the handler in `tui/slash_commands.go`

Keep handlers simple and fail fast. Prefer returning early on invalid state.

If the command is user-visible in chat history, emit the command trace first:

```go
m.AppendRawMessage("● /mycmd")
```

Then emit result/error child lines:

```go
m.AppendRawMessage("  ↳ Did thing")
// or
m.AppendRawMessage("  ↳ Failed: <reason>")
```

Reset input (`m.input.Reset()`) before returning.

### 4. If command is async, add message types and update handling

For background work:
- add typed messages in `tui/messages.go`
- return a `tea.Cmd` from handler
- handle those messages in `tui/update.go`

Pattern:
- parent command line from handler (`● /mycmd`)
- completion/error child line from `Update()` (`↳ ...`)

### 5. Verify behavior in TUI

Check these flows:
- `/` picker shows the new command and description
- Enter and tab-complete paths both work
- `/help` includes the command
- output uses parent/child format

Expected output shape:

```text
● /mycmd
  ↳ Did thing
```

## UX Contract For Slash Commands

Use this output style for slash commands:
- parent line: `● /command` (once per invocation)
- child lines: `  ↳ ...` for result, progress milestones, or errors

Avoid flat standalone sentences like `Something changed to X` for slash command outcomes.
Keep slash command output visually grouped so chat history is easy to scan.

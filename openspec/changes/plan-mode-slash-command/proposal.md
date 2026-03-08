## Why

Bono needs a `/plan` slash command, but more importantly it needs a proper subagent architecture. The current `PreTaskConfig` + `RunPreTask` approach is a flat struct with a hardcoded loop — not extensible for the growing list of planned subagents (plan, review, docs, etc.). Each new subagent would require duplicating patterns and adding bespoke code paths.

A clean subagent system in bono-core with interfaces, a registry, versioned prompt templates, and read-only tool filtering will make adding future subagents trivial — define a prompt template, register it, done. The TUI (and future web/IDE frontends) stays a dumb event consumer.

## What Changes

- **New subagent interface and registry in bono-core**: `SubAgent` interface with `Name()`, `AllowedTools()`, `SystemPrompt()`, a registry on Agent for registration/lookup, and `RunSubAgent` runner with tool filtering
- **Built-in subagent auto-registration**: All subagents register inside `NewAgent()` via `registerBuiltinSubAgents()` — any consumer of bono-core gets all subagents automatically without importing sub-packages
- **Versioned prompt templates for subagents**: Following bono's existing `prompts/versions/*.tmpl` pattern, subagent prompts live in `subagent_prompts/<name>/*.tmpl` in bono-core with embedded FS
- **Tool filtering**: Subagents declare allowed tools; the runner enforces this at API schema level and rejects disallowed calls at runtime
- **`/plan <task>` slash command in TUI**: Thin handler that appends parent/child UX lines synchronously, then dispatches to bono-core's subagent runner
- **New explanation doc** (`docs/explaination/subagent-system.md`): How the subagent system works — interface, registry, runner, versioned prompts, event flow
- **New how-to doc** (`docs/how-to/new-subagent-slash-command.md`): Runbook for adding a new subagent-backed slash command
- **Update existing docs**: `docs/how-to/new-slash-command.md` to cross-reference subagent variant, `docs/explaination/system-prompt-lifecycle.md` to cover subagent prompt versioning, and CLAUDE.md to link new docs

## Capabilities

### New Capabilities
- `subagent-system`: The SubAgent interface, registry, runner, tool filtering, and versioned prompt loading in bono-core
- `plan-slash-command`: The `/plan` TUI slash command and its plan subagent implementation

### Modified Capabilities
<!-- No existing spec-level requirements are changing -->

## Impact

- **bono-core**: `subagent.go` (interface), `subagents.go` (built-in implementations + auto-registration), `agent.go` (runner + callbacks)
- **bono-core**: `subagent_prompts/plan/v1.0.0.tmpl` for plan mode system prompt
- **bono**: `tui/slash_commands.go` — new `/plan` handler + generic `runSubAgent` method
- **bono**: `tui/messages.go`, `tui/update.go` — subagent lifecycle messages
- **bono**: `internal/session/events.go`, `session.go` — subagent events wired to frontend
- No breaking changes to existing APIs — new system runs alongside `RunPreTask`
- `docs/explaination/subagent-system.md` — new explanation doc
- `docs/how-to/new-subagent-slash-command.md` — new how-to runbook
- `docs/how-to/new-slash-command.md` — updated to reference subagent pattern
- `docs/explaination/system-prompt-lifecycle.md` — updated to cover subagent prompt versioning
- `CLAUDE.md` — updated with links to new docs

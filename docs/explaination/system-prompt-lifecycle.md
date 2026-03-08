# System Prompt Lifecycle

## Data Flow

1. `main.go` builds `hostCtx`.
2. `systemPromptVersion` picks a template.
3. `prompts.LoadSystemPromptVersion` reads and renders `prompts/versions/<version>.tmpl`.
4. Rendered text is assigned to `core.Config.SystemPrompt`.
5. Agent is created and uses that system prompt for turns.

## Design Intent

- Revision history lives in files, not commits.
- Active prompt choice is explicit in code.
- Template rendering is strict; bad keys fail fast.

## Contract Surface

- Runtime inputs to templates are defined by `HostContext`.
- Template files should only depend on that struct.
- If template scope changes, update `HostContext` and `main.go` together.

## Subagent Prompt Versioning

Subagents in bono-core follow the same versioned template pattern, but scoped per subagent package:

- **Main agent prompts**: `bono/prompts/versions/*.tmpl` (embedded in bono)
- **Subagent prompts**: `bono-core/subagent_prompts/<name>/*.tmpl` (embedded in bono-core's `subagents.go`)

Each subagent embeds its own `prompts/*.tmpl` via `//go:embed` and loads the active version in its `SystemPrompt()` method. See [How subagents work](./subagent-system.md) for the full architecture.

## Practical Reading Order

1. `main.go`
2. `prompts/prompts.go`
3. `prompts/versions/*.tmpl`
4. `bono-core/subagents.go` + `bono-core/subagent_prompts/plan/v1.0.0.tmpl`

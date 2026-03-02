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

## Practical Reading Order

1. `main.go`
2. `prompts/prompts.go`
3. `prompts/versions/*.tmpl`

# How to Add a New Subagent Slash Command

## Where Things Live

| What | Where |
|------|-------|
| SubAgent interface | `bono-core/subagent.go` |
| Subagent implementations | `bono-core/subagents.go` |
| Versioned prompts | `bono-core/subagent_prompts/<name>/*.tmpl` |
| SubAgent runner | `bono-core/agent.go` → `RunSubAgent()` |
| Auto-registration | `bono-core/subagents.go` → `registerBuiltinSubAgents()` |
| Slash command registration | `bono/tui/slash_commands.go` |

## Runbook

### 1. Create the prompt template

Create `bono-core/subagent_prompts/<name>/v1.0.0.tmpl` with the subagent's system prompt.

### 2. Add the subagent implementation in `bono-core/subagents.go`

Add the embed directive, type, constructor, and interface methods. Follow the existing `planAgent` as a pattern:

```go
//go:embed subagent_prompts/<name>/*.tmpl
var <name>PromptFS embed.FS

const <name>PromptVersion = "v1.0.0"

type <name>Agent struct {
    prompt string
}

func new<Name>Agent() *<name>Agent {
    content, err := <name>PromptFS.ReadFile("subagent_prompts/<name>/" + <name>PromptVersion + ".tmpl")
    if err != nil {
        panic("<name>: missing prompt " + <name>PromptVersion + ": " + err.Error())
    }
    return &<name>Agent{prompt: string(content)}
}

func (a *<name>Agent) Name() string          { return "<name>" }
func (a *<name>Agent) AllowedTools() []string { return []string{"read_file", "run_shell"} }
func (a *<name>Agent) SystemPrompt() string   { return a.prompt }

var _ SubAgent = (*<name>Agent)(nil)
```

### 3. Register in `registerBuiltinSubAgents()`

Add a line to `registerBuiltinSubAgents()` in `subagents.go`:

```go
a.RegisterSubAgent(new<Name>Agent())
```

This ensures every consumer of bono-core gets the subagent automatically.

### 4. Add the slash command in `bono/tui/slash_commands.go`

Add to `DefaultSlashCommandSpecs()`:

```go
{Name: "<name>", Description: "<description>", Handler: handle<Name>},
```

Add the handler — append parent/child lines synchronously, then dispatch:

```go
func handle<Name>(m *Model, arg string) tea.Cmd {
    m.AppendRawMessage("● /<name>")
    if strings.TrimSpace(arg) == "" {
        m.AppendRawMessage("  ↳ Usage: /<name> <description>")
        m.input.Reset()
        return nil
    }
    m.AppendRawMessage("  ↳ Starting <name> subagent...")
    return m.runSubAgent("<name>", arg)
}
```

### 5. Update `helpText`

Add the command to the `helpText` constant in `tui/slash_commands.go`.

### 6. Verify

- `/<name>` appears in slash picker and `/help`
- `/<name>` without args shows usage
- `/<name> <task>` runs the subagent with correct tool restrictions

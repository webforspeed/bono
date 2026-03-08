## 1. bono-core: SubAgent Interface and Registry

- [x] 1.1 Define `SubAgent` interface in bono-core (`subagent.go`) with `Name() string`, `AllowedTools() []string`, `SystemPrompt() string`
- [x] 1.2 Add `subAgents map[string]SubAgent` field to `Agent` struct with `RegisterSubAgent(SubAgent)` and `SubAgent(name string) (SubAgent, bool)` methods

## 2. bono-core: SubAgent Runner

- [x] 2.1 Implement `Agent.RunSubAgent(ctx context.Context, sa SubAgent, input string) error` — isolated message history, tool filtering at API schema level, existing callback wiring
- [x] 2.2 Add tool filtering logic: filter `a.registry.Tools(allowed...)` to only schemas matching `sa.AllowedTools()`, reject disallowed tool calls at runtime as second check
- [x] 2.3 Add `OnSubAgentStart func(name string)` and `OnSubAgentEnd func(name string)` callback fields to Agent, fire them in `RunSubAgent`

## 3. bono-core: Plan SubAgent Implementation

- [x] 3.1 Add `planAgent` struct to `subagents.go` implementing `SubAgent` — Name returns "plan", AllowedTools returns `read_file`, `run_shell`, `code_search`
- [x] 3.2 Create `subagent_prompts/plan/v1.0.0.tmpl` with planning-optimized system prompt (analyze codebase, propose approach, identify files, produce ordered steps)
- [x] 3.3 Implement prompt loading using `//go:embed subagent_prompts/plan/*.tmpl` in `subagents.go`
- [x] 3.4 Add `registerBuiltinSubAgents()` called from `NewAgent()` — auto-registers all built-in subagents so consumers of bono-core get them for free

## 4. bono: Session Events for SubAgents

- [x] 4.1 Add `SubAgentStartEvent` and `SubAgentEndEvent` to `internal/session/events.go`
- [x] 4.2 Wire `OnSubAgentStart` / `OnSubAgentEnd` callbacks in `session.Bind()` to emit these events via `frontend.HandleEvent()`
- [x] 4.3 Handle `SubAgentStartEvent` / `SubAgentEndEvent` in TUI frontend (`tui/session_frontend.go`) — map to Bubble Tea messages
- [x] 4.4 Handle `SubAgentStartEvent` / `SubAgentEndEvent` in headless frontend (`internal/session/headless.go`)

## 5. bono: TUI Slash Command

- [x] 5.1 Add generic `runSubAgent(name, input string) tea.Cmd` method on Model — lookup subagent, set processing, activate spinner, call `agent.RunSubAgent` async, return `SubAgentDoneMsg`
- [x] 5.2 Add `SubAgentStartMsg`, `SubAgentEndMsg`, `SubAgentDoneMsg` to `tui/messages.go`, handle in `tui/update.go` (deactivate spinner, reset processing on done)
- [x] 5.3 Implement `handlePlan(m *Model, arg string) tea.Cmd` — append parent line `● /plan` and child line synchronously, validate arg not empty, show usage if missing, call `m.runSubAgent("plan", arg)`
- [x] 5.4 Register `/plan` in `DefaultSlashCommandSpecs()` and add to `helpText`

## 6. Documentation

- [x] 6.1 Create `docs/explaination/subagent-system.md` — explain how the subagent system works: SubAgent interface in bono-core, registry on Agent, RunSubAgent runner with tool filtering, versioned prompt templates, auto-registration in NewAgent(), event flow from bono-core callbacks through session events to TUI/headless frontends
- [x] 6.2 Create `docs/how-to/new-subagent-slash-command.md` — runbook for adding a new subagent-backed slash command: add prompt template, add implementation to `subagents.go`, register in `registerBuiltinSubAgents()`, add slash command entry and thin handler in TUI
- [x] 6.3 Update `docs/how-to/new-slash-command.md` — add a note that for slash commands that run a subagent, see the subagent-specific how-to instead
- [x] 6.4 Update `docs/explaination/system-prompt-lifecycle.md` — add section on subagent prompt versioning (`subagent_prompts/<name>/*.tmpl` in bono-core)
- [x] 6.5 Update `CLAUDE.md` — add links to new docs under How-to Guides and Explanation sections

## 7. Verification

- [x] 7.1 Build both bono-core and bono, verify no compilation errors
- [ ] 7.2 Test `/plan` without argument shows parent/child usage message
- [ ] 7.3 Test `/plan <task>` triggers the plan subagent, streams output, completes cleanly
- [ ] 7.4 Test `/help` includes the plan command
- [ ] 7.5 Verify subagent only has access to read-only tools (no edit/write in tool schemas sent to LLM)

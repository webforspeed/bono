## ADDED Requirements

### Requirement: SubAgent interface
bono-core SHALL define a `SubAgent` interface with methods: `Name() string`, `AllowedTools() []string`, and `SystemPrompt() string`.

#### Scenario: Interface contract
- **WHEN** a type implements `Name()`, `AllowedTools()`, and `SystemPrompt()`
- **THEN** it SHALL satisfy the `SubAgent` interface and be usable with `RunSubAgent`

### Requirement: SubAgent registration on Agent
The `Agent` struct SHALL support registering and looking up subagents by name.

#### Scenario: Register and retrieve
- **WHEN** `agent.RegisterSubAgent(planAgent)` is called where `planAgent.Name()` returns "plan"
- **THEN** `agent.SubAgent("plan")` SHALL return the registered subagent and true

#### Scenario: Unknown subagent lookup
- **WHEN** `agent.SubAgent("nonexistent")` is called
- **THEN** it SHALL return nil and false

### Requirement: Built-in subagent auto-registration
All built-in subagents SHALL be registered automatically inside `NewAgent()` via `registerBuiltinSubAgents()`. Consumers of bono-core SHALL NOT need to import or register subagents manually.

#### Scenario: Auto-registration
- **WHEN** `core.NewAgent(config)` is called
- **THEN** all built-in subagents (including "plan") SHALL be registered and available via `agent.SubAgent(name)`

### Requirement: SubAgent runner with tool filtering
`Agent.RunSubAgent(ctx, subAgent, input)` SHALL execute the subagent in an isolated message history, filtering available tools to only those in `subAgent.AllowedTools()`.

#### Scenario: Isolated execution
- **WHEN** `RunSubAgent` is called
- **THEN** it SHALL create a new message history with the subagent's system prompt and user input
- **THEN** it SHALL NOT modify the main agent's conversation history

#### Scenario: Tool filtering at API level
- **WHEN** `RunSubAgent` calls the LLM
- **THEN** it SHALL only pass tool schemas for tools in `subAgent.AllowedTools()`
- **THEN** if `AllowedTools()` returns empty, all registered tools SHALL be available

#### Scenario: Disallowed tool call rejection
- **WHEN** the LLM returns a tool call for a tool not in `AllowedTools()`
- **THEN** the runner SHALL return an error result for that tool call

### Requirement: SubAgent lifecycle callbacks
The Agent SHALL fire `OnSubAgentStart(name)` before and `OnSubAgentEnd(name)` after subagent execution.

#### Scenario: Callback firing
- **WHEN** `RunSubAgent` is called for subagent "plan"
- **THEN** `OnSubAgentStart("plan")` SHALL fire before the first LLM call
- **THEN** `OnSubAgentEnd("plan")` SHALL fire after the subagent completes or errors

### Requirement: SubAgent events in session
bono's session layer SHALL translate subagent callbacks into frontend events (`SubAgentStartEvent`, `SubAgentEndEvent`) using the existing `SessionFrontend.HandleEvent()` path.

#### Scenario: Event flow to TUI
- **WHEN** `OnSubAgentStart("plan")` fires
- **THEN** the session SHALL call `frontend.HandleEvent(ctx, SubAgentStartEvent{Name: "plan"})`

#### Scenario: Event flow to headless
- **WHEN** `OnSubAgentStart("plan")` fires in headless mode
- **THEN** the headless frontend SHALL print "● Running plan agent..."

### Requirement: Plan subagent implementation
bono-core SHALL include a plan subagent implemented in `subagents.go` that satisfies the `SubAgent` interface.

#### Scenario: Plan subagent properties
- **WHEN** the plan subagent is instantiated
- **THEN** `Name()` SHALL return "plan"
- **THEN** `AllowedTools()` SHALL return read-only tools: `read_file`, `run_shell`, `code_search`
- **THEN** `SystemPrompt()` SHALL return a planning-optimized prompt loaded from a versioned template

### Requirement: Versioned prompt templates
Each subagent SHALL store its system prompts as versioned `.tmpl` files embedded via `//go:embed`, following the pattern `subagent_prompts/<name>/v*.tmpl` in the bono-core package.

#### Scenario: Prompt loading
- **WHEN** the plan subagent is constructed via `newPlanAgent()`
- **THEN** it SHALL load the active version template from `subagent_prompts/plan/`

### Requirement: Subagent turn limit
The subagent runner SHALL enforce a maximum of 100 turns per subagent run.

#### Scenario: Turn limit reached
- **WHEN** a subagent exceeds 100 turns
- **THEN** the runner SHALL return an error

#### Limitation: No handoff on failure
When a subagent fails mid-run (turn limit, API error, cancellation), the main agent receives no handoff summary. Any work the subagent did is lost from the main agent's context, though the user can still see streamed output in the TUI. **Future fix:** accumulate the last content seen across turns and always inject a handoff (tagged as incomplete on error).

### Requirement: Planning prompt content
The plan subagent's system prompt SHALL instruct the agent to: analyze the codebase relevant to the task, identify key files and architecture, propose an implementation approach, and produce ordered implementation steps.

#### Scenario: Plan output structure
- **WHEN** the plan subagent runs with a task description
- **THEN** it SHALL produce output covering: analysis of relevant code, proposed approach, files to modify, and ordered steps

## ADDED Requirements

### Requirement: SubAgentHook interface
The system SHALL define a `SubAgentHook` interface with an `AfterComplete(ctx context.Context, result *SubAgentResult) error` method. Hooks run after a subagent's final response, before the handoff message is built.

#### Scenario: Hook executes after subagent completion
- **WHEN** a subagent completes with a final text response and has registered hooks
- **THEN** each hook's `AfterComplete` SHALL be called in registration order with the subagent's result

#### Scenario: Subagent with no hooks
- **WHEN** a subagent completes with no registered hooks
- **THEN** `RunSubAgent` SHALL behave exactly as today

### Requirement: SubAgentResult carries context for hooks
The `SubAgentResult` struct SHALL contain: `Name` (subagent name), `Input` (user input), `Content` (final response text), `CWD` (working directory), and `Meta` (mutable `map[string]string` for hook annotations).

#### Scenario: Hook annotates result
- **WHEN** a hook writes `result.Meta["output_path"] = "/some/path"`
- **THEN** subsequent hooks and the handoff builder SHALL see that annotation

### Requirement: RegisterSubAgent accepts optional hooks
`RegisterSubAgent(sa SubAgent, hooks ...SubAgentHook)` SHALL accept zero or more hooks. Hooks SHALL be stored alongside the subagent and passed to `RunSubAgent` at execution time.

#### Scenario: Register with hooks
- **WHEN** `RegisterSubAgent(planAgent, persistHook, approvalHook)` is called
- **THEN** both hooks SHALL run in order whenever the plan subagent completes

#### Scenario: Register without hooks
- **WHEN** `RegisterSubAgent(reviewAgent)` is called with no hooks
- **THEN** no hooks SHALL run after the review subagent completes

### Requirement: PersistHook writes output to a deterministic path
`PersistHook(dirTemplate string)` SHALL return a `SubAgentHook` that writes `SubAgentResult.Content` to `<resolved-dir>/<unix-timestamp>-<slug>.md`. The directory template SHALL support `{cwd}` placeholder expanded from `SubAgentResult.CWD`. The `~` prefix SHALL be resolved to the user's home directory.

#### Scenario: Plan subagent with PersistHook
- **WHEN** the plan subagent completes with CWD `/Users/alice/myproject` and dir template `~/.bono/{cwd}/plans`
- **THEN** the file SHALL be written to `~/.bono/Users/alice/myproject/plans/<timestamp>-<slug>.md`

#### Scenario: Directory does not exist
- **WHEN** the resolved directory does not exist
- **THEN** it SHALL be created with `os.MkdirAll` using `0755` permissions

#### Scenario: Write failure is best-effort
- **WHEN** the file write fails (e.g., permission error)
- **THEN** `AfterComplete` SHALL return the error, `RunSubAgent` SHALL log it but still return the plan content successfully

#### Scenario: Revision overwrites the same file
- **WHEN** `Meta["output_path"]` is already set from a previous PersistHook run (revision cycle)
- **THEN** PersistHook SHALL overwrite the existing file at that path instead of generating a new filename

### Requirement: Deterministic file naming
The file name SHALL be `<unix-timestamp>-<sanitized-slug>.md`. The slug SHALL be derived from the user input: lowercased, non-alphanumeric characters replaced with hyphens, consecutive hyphens collapsed, leading/trailing hyphens stripped, truncated to 50 characters.

#### Scenario: Normal input text
- **WHEN** the user input is `"Add user authentication to the API"` and the unix timestamp is `1709913600`
- **THEN** the file SHALL be named `1709913600-add-user-authentication-to-the-api.md`

#### Scenario: Input with special characters
- **WHEN** the user input is `"Fix bug #123 — can't login!!!"`
- **THEN** the slug SHALL contain only `[a-z0-9-]` with no leading/trailing/consecutive hyphens

#### Scenario: Very long input
- **WHEN** the user input exceeds 50 characters after sanitization
- **THEN** the slug SHALL be truncated to 50 characters with no trailing hyphen

### Requirement: ApprovalHook prompts user after subagent completion
`ApprovalHook()` SHALL return a `SubAgentHook` that calls `Agent.OnSubAgentApproval` (if set) and blocks until the user responds. The response SHALL be a `SubAgentApprovalResponse` with an `Action` (Approve, Reject, Revise) and optional `Feedback` string.

#### Scenario: User approves the plan
- **WHEN** the user presses Enter with empty input
- **THEN** `ApprovalHook` SHALL return nil and `SubAgentResult.Meta["approval"]` SHALL be `"approved"`

#### Scenario: User rejects the plan
- **WHEN** the user presses Escape
- **THEN** `ApprovalHook` SHALL return nil and `SubAgentResult.Meta["approval"]` SHALL be `"rejected"`

#### Scenario: User provides revision feedback
- **WHEN** the user types feedback and presses Enter
- **THEN** `ApprovalHook` SHALL set `SubAgentResult.Meta["approval"]` to `"revise"` and `Meta["feedback"]` to the user's text

#### Scenario: OnSubAgentApproval is nil (headless/tests)
- **WHEN** `OnSubAgentApproval` is not set
- **THEN** `ApprovalHook` SHALL auto-approve

### Requirement: RunSubAgent supports revision loop
When a hook sets `Meta["approval"]` to `"revise"` with feedback, `RunSubAgent` SHALL append the feedback as a user message to the subagent's isolated history, continue the LLM loop for a revised response, and re-run hooks on the new result.

#### Scenario: Single revision
- **WHEN** the user provides feedback "add more detail on step 3"
- **THEN** the subagent SHALL receive the feedback, produce a revised plan, hooks SHALL run again (PersistHook overwrites file, ApprovalHook re-prompts)

#### Scenario: Multiple revisions
- **WHEN** the user provides feedback twice before approving
- **THEN** the revision loop SHALL run twice, each time overwriting the plan file and re-prompting

### Requirement: Handoff message varies by approval outcome
The handoff message injected into the main conversation SHALL differ based on approval:

#### Scenario: Approved plan
- **WHEN** `Meta["approval"]` is `"approved"` and `Meta["output_path"]` is set
- **THEN** the handoff SHALL include the plan content, the file path, and an instruction to implement the plan

#### Scenario: Rejected plan
- **WHEN** `Meta["approval"]` is `"rejected"`
- **THEN** the handoff SHALL include the plan content and file path but no implementation instruction

#### Scenario: No approval hook (legacy/no hooks)
- **WHEN** no approval-related `Meta` keys are set
- **THEN** the handoff SHALL behave as today (content only, no file path)

### Requirement: Plan subagent registers with PersistHook and ApprovalHook
The `planAgent` SHALL be registered with `PersistHook("~/.bono/{cwd}/plans")` followed by `ApprovalHook()` in `registerBuiltinSubAgents()`.

#### Scenario: Plan subagent registration
- **WHEN** `registerBuiltinSubAgents()` runs
- **THEN** the plan subagent SHALL be registered with PersistHook then ApprovalHook in that order

### Requirement: No approval required for plan file writes
The file write SHALL occur inside PersistHook, not through a tool call. It SHALL NOT trigger the `OnToolCall` approval callback. Only the ApprovalHook (which runs after the file is written) prompts the user.

#### Scenario: Plan completes
- **WHEN** the plan subagent produces its final response
- **THEN** the file SHALL be written without approval, then the user SHALL be prompted to approve/reject/revise the plan

### Requirement: TUI renders plan approval prompt
The TUI SHALL display the plan approval as:
- A line showing the saved file path
- A prompt line following existing conventions with `[Enter/Esc]` suffix
- Enter with empty input = approve, Esc = reject, Enter with text = revise with that text as feedback

#### Scenario: Approval prompt displayed
- **WHEN** `ApprovalHook` triggers in the TUI
- **THEN** the prompt SHALL show the file path and instructions for approve/reject/revise

#### Scenario: Revision clears approval prompt
- **WHEN** the user types feedback and presses Enter
- **THEN** the approval prompt SHALL be cleared and the spinner SHALL show the subagent is revising

### Requirement: RunSubAgent returns SubAgentResult
`RunSubAgent` SHALL return `(*SubAgentResult, error)` so callers can inspect hook annotations (e.g., `Meta["approval"]`).

#### Scenario: Caller checks approval status
- **WHEN** `RunSubAgent` completes
- **THEN** the caller SHALL receive the `SubAgentResult` with all hook-annotated `Meta` fields

### Requirement: Auto-implementation after plan approval
When `SubAgentDoneMsg.Approved` is true, the TUI SHALL automatically trigger `agent.Chat(ctx, "Implement the plan.")` without waiting for user input. The handoff message is already in the main conversation context.

#### Scenario: Approved plan triggers main agent
- **WHEN** `SubAgentDoneMsg` is received with `Approved=true`
- **THEN** the TUI SHALL keep the spinner active, call `agent.Chat("Implement the plan.")`, and return `AgentResponseMsg` on completion

#### Scenario: Rejected plan does not trigger main agent
- **WHEN** `SubAgentDoneMsg` is received with `Approved=false`
- **THEN** the TUI SHALL deactivate the spinner and reset to idle

### Requirement: Revision cycle preserves output path
The agent SHALL track `Meta["output_path"]` across revision cycles so PersistHook overwrites the same file rather than generating a new one.

#### Scenario: Revision overwrites same file
- **WHEN** the user requests a revision and PersistHook runs again
- **THEN** PersistHook SHALL reuse the previously stored `output_path` and overwrite the existing file

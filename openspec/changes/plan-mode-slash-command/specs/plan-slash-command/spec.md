## ADDED Requirements

### Requirement: Plan command registration
The `/plan` command SHALL be registered in `DefaultSlashCommandSpecs()` with name "plan", description "Plan a task before implementing", and a handler function.

#### Scenario: Command appears in slash picker
- **WHEN** user types "/" in the TUI input
- **THEN** the slash picker modal SHALL show "plan" with description "Plan a task before implementing"

#### Scenario: Command appears in help
- **WHEN** user runs `/help`
- **THEN** the help text SHALL include `/plan <task>` with its description

### Requirement: Plan command requires task argument
The `/plan` handler SHALL validate that a non-empty task argument is provided.

#### Scenario: No argument provided
- **WHEN** user submits `/plan` with no argument or whitespace-only argument
- **THEN** the handler SHALL display `● /plan` as parent line and `  ↳ Usage: /plan <task description>` as child line
- **THEN** the handler SHALL NOT start any subagent

#### Scenario: Valid argument provided
- **WHEN** user submits `/plan refactor the session module`
- **THEN** the handler SHALL dispatch to bono-core's subagent runner with name "plan" and input "refactor the session module"

### Requirement: Plan command blocked during processing
The handler SHALL NOT start a new subagent run if another operation is already processing.

#### Scenario: Agent already processing
- **WHEN** user submits `/plan <task>` while `m.processing` is true
- **THEN** the handler SHALL return nil without starting the subagent

### Requirement: Generic subagent dispatch from TUI
The TUI SHALL have a generic `runSubAgent(name, input)` method that any slash command can call to run a subagent by name. The `/plan` handler SHALL use this method.

#### Scenario: Subagent dispatch
- **WHEN** `runSubAgent("plan", "some task")` is called
- **THEN** it SHALL look up the subagent by name on the agent, set processing state, activate spinner, and call `agent.RunSubAgent(ctx, subagent, input)` asynchronously

#### Scenario: Unknown subagent
- **WHEN** `runSubAgent("nonexistent", "task")` is called
- **THEN** it SHALL display an error message and not start processing

### Requirement: Subagent TUI feedback
The TUI SHALL display subagent lifecycle using the parent/child UX contract. Both lines SHALL be appended synchronously in the handler before async work begins.

#### Scenario: Subagent starts
- **WHEN** a subagent starts via `/plan <task>`
- **THEN** the TUI SHALL display `● /plan` as the parent line
- **THEN** the TUI SHALL display `  ↳ Starting planning subagent...` as the child line
- **THEN** the spinner SHALL activate
- **THEN** the input SHALL be reset

#### Scenario: Subagent completes successfully
- **WHEN** a subagent finishes without error
- **THEN** the spinner SHALL deactivate
- **THEN** processing state SHALL be set to false

#### Scenario: Subagent fails
- **WHEN** a subagent finishes with an error
- **THEN** the TUI SHALL display `  ↳ Failed: <error message>` as a child line
- **THEN** the spinner SHALL deactivate
- **THEN** processing state SHALL be set to false

#### Scenario: No argument provided
- **WHEN** user submits `/plan` with no argument
- **THEN** the TUI SHALL display `● /plan` as the parent line
- **THEN** the TUI SHALL display `  ↳ Usage: /plan <task description>` as the child line

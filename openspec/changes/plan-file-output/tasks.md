## 1. Hook Types and SubAgentResult

- [x] 1.1 Add `SubAgentResult` struct to `bono-core/subagent.go` with fields: `Name`, `Input`, `Content`, `CWD`, `Meta map[string]string`
- [x] 1.2 Add `SubAgentHook` interface to `bono-core/subagent.go` with `AfterComplete(ctx context.Context, result *SubAgentResult) error`
- [x] 1.3 Add `SubAgentApprovalAction` type (Approve/Reject/Revise) and `SubAgentApprovalResponse` struct to `bono-core/subagent.go`

## 2. Registry Change

- [x] 2.1 Update `Agent.subAgents` from `map[string]SubAgent` to a struct holding `SubAgent` + `[]SubAgentHook`
- [x] 2.2 Update `RegisterSubAgent(sa SubAgent, hooks ...SubAgentHook)` signature and storage
- [x] 2.3 Update `SubAgent(name)` lookup to return subagent and hooks

## 3. Slug Helper

- [x] 3.1 Add `slugify(input string, maxLen int) string` in `bono-core/subagent_hooks.go`
- [x] 3.2 Add tests for slugify: normal text, special chars, long input, empty input, edge cases

## 4. PersistHook Implementation

- [x] 4.1 Add `PersistHook(dirTemplate string) SubAgentHook` in `bono-core/subagent_hooks.go` — resolve `~`, expand `{cwd}`, build filename, MkdirAll, WriteFile, set `Meta["output_path"]`; on revision reuse existing `Meta["output_path"]`
- [x] 4.2 Add tests for PersistHook: file creation, directory creation, template expansion, revision overwrite, write failure handling

## 5. ApprovalHook Implementation

- [x] 5.1 Add `OnSubAgentApproval func(SubAgentResult) SubAgentApprovalResponse` callback field to `Agent`
- [x] 5.2 Add `ApprovalHook() SubAgentHook` in `bono-core/subagent_hooks.go` — calls `OnSubAgentApproval` if set, auto-approves if nil, sets `Meta["approval"]` and `Meta["feedback"]`
- [x] 5.3 Add tests for ApprovalHook: approve, reject, revise, nil callback (auto-approve)

## 6. RunSubAgent Revision Loop

- [x] 6.1 After final response in `RunSubAgent`: build `SubAgentResult`, iterate hooks, check `Meta["approval"]` — if `"revise"`, append feedback as user message to isolated history and continue the LLM loop; re-run hooks on new result
- [x] 6.2 Update handoff message to vary by `Meta["approval"]` and include `Meta["output_path"]` when present

## 7. Frontend Plumbing (bono repo)

- [x] 7.1 Add `AgentPlanApprovalMsg` to `tui/messages.go` with fields for file path and a response channel carrying `SubAgentApprovalResponse`
- [x] 7.2 Bind `OnSubAgentApproval` in `session.go` to send `AgentPlanApprovalMsg` via the frontend and block on the response channel
- [x] 7.3 Add `pendingPlanApproval` field to TUI `Model`
- [x] 7.4 Handle `AgentPlanApprovalMsg` in `tui/update.go`: display prompt line, set pending state
- [x] 7.5 Handle Enter (empty = approve, text = revise) and Esc (reject) for plan approval in the key handling section of `tui/update.go`

## 8. Plan Subagent Registration

- [x] 8.1 Update `registerBuiltinSubAgents()` to register planAgent with `PersistHook("~/.bono/{cwd}/plans")` then `ApprovalHook()`

## 9. Verify

- [x] 9.1 Run existing tests to confirm nothing breaks
- [x] 9.2 Manual test: `/plan <task>` → verify file at `~/.bono/<cwd>/plans/`, approval prompt appears, Enter implements, Esc skips, typed feedback revises

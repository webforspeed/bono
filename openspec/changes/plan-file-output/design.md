## Context

The plan subagent runs in an isolated message loop inside `RunSubAgent`. When the LLM's final turn has no tool calls, the runner extracts the text content, injects a handoff summary into the main conversation, and returns. Today this content is purely in-memory and the handoff is automatic.

The user wants:
- Plans persisted to `~/.bono/<cwd>/plans/` with deterministic naming, auto-approved
- An interactive approval gate after plan creation: approve (implement), reject (discard), or revise (with feedback)
- Extensibility: other subagents may need similar persistence or hooks; the plan subagent itself may produce different artifact types in the future

## Goals / Non-Goals

**Goals:**
- Persist the plan subagent's final output to `~/.bono/<cwd-path>/plans/<timestamp>-<slug>.md`
- Use a `SubAgentHook` middleware pattern so any subagent can compose post-completion behaviors
- Add an approval loop after plan completion with three outcomes: approve, reject, revise
- Keep `SubAgent` interface minimal — hooks and approval are orthogonal to identity

**Non-Goals:**
- Giving the plan subagent a write tool — the LLM must not control the path
- Plan management (listing, deleting, searching plans) — future work
- Persisting artifacts other than the final response — future work

## Decisions

### 1. `SubAgentHook` interface for post-completion behaviors

```go
type SubAgentHook interface {
    AfterComplete(ctx context.Context, result *SubAgentResult) error
}
```

`SubAgentResult` carries: subagent name, user input, final content, CWD, and a mutable `Meta map[string]string` for hook annotations. Hooks are attached at registration time and executed in order.

**Why not add methods to SubAgent?** Persistence and approval are cross-cutting behaviors. A review subagent might also want persistence. Hooks compose without touching the interface.

### 2. `PersistHook` — a concrete hook for file output

```go
func PersistHook(dirTemplate string) SubAgentHook
```

`dirTemplate` uses `{cwd}` placeholder: `"~/.bono/{cwd}/plans"`. The hook resolves `~`, expands `{cwd}` from `SubAgentResult.CWD`, generates `<timestamp>-<slug>.md`, writes the file, and sets `Meta["output_path"]`.

Reusable: `PersistHook("~/.bono/{cwd}/reviews")` for a future review subagent.

### 3. Approval loop via `OnSubAgentApproval` callback

The existing `RequestApproval` returns `bool` — insufficient for the three-outcome flow (approve/reject/revise-with-feedback). Rather than widening `RequestApproval`, a new callback on `Agent` handles this:

```go
// SubAgentApprovalResponse represents the user's decision after plan review.
type SubAgentApprovalResponse struct {
    Action   SubAgentApprovalAction // Approve, Reject, Revise
    Feedback string                 // non-empty when Action == Revise
}

// OnSubAgentApproval is called after hooks complete when the subagent requests approval.
// The runner blocks until the callback returns.
Agent.OnSubAgentApproval func(result SubAgentResult) SubAgentApprovalResponse
```

**Why a new callback instead of extending `RequestApproval`?** `RequestApproval` is a `SessionFrontend` method — it returns `bool` and is used for tool-level decisions. The plan approval has different semantics (three outcomes, free-text feedback, displayed differently). Mixing them would complicate the frontend interface. A callback on `Agent` keeps it in the same pattern as `OnToolCall`, `OnSandboxFallback`, etc.

**Revision loop:** When the user provides feedback, `RunSubAgent` appends the feedback as a user message to the subagent's isolated history and continues the loop. The subagent revises, hooks run again (PersistHook overwrites the same file since the slug is derived from the original input), and approval is re-prompted.

### 4. Which subagents get approval?

Approval is opt-in per subagent via a hook:

```go
func ApprovalHook() SubAgentHook
```

`ApprovalHook` checks if `OnSubAgentApproval` is set and calls it. If not set (e.g., headless mode), it auto-approves. This keeps approval composable — not all subagents need it, and it's independent of persistence.

The plan subagent registers with both: `PersistHook(...)` then `ApprovalHook()`. Persist runs first (writes the file), then Approval runs (prompts the user with the file path visible).

### 5. Project-scoped paths: `~/.bono/<cwd>/plans/`

CWD absolute path is embedded in the directory structure (e.g., `~/.bono/Users/nanda/code/go/bono/plans/`). Human-browsable, mirrors how `.bono/index.db` already scopes per-project.

### 6. File naming: `<unix-timestamp>-<sanitized-slug>.md`

- Timestamp prefix for uniqueness and sort order
- Slug from user input, sanitized to `[a-z0-9-]`, truncated to 50 chars

### 7. Revision overwrites the same file

On revise, the subagent produces updated content. PersistHook runs again but uses the same `Meta["output_path"]` if already set (same timestamp + slug from the first write). This avoids accumulating draft files.

### 8. Handoff message varies by approval outcome

- **Approve:** `[plan agent summary]\n<content>\n\nPlan saved to <path>.\nImplement the plan above. Follow each step in order.`
- **Reject:** `[plan agent summary]\n<content>\n\nPlan saved to <path>.\nPlan was not approved for implementation.`
- Both include the plan content so the main agent has context regardless.

### 9. Auto-implementation after approval

When `SubAgentDoneMsg.Approved=true`, the TUI auto-triggers `agent.Chat(ctx, "Implement the plan.")`. The handoff is already in context, so the main agent picks up the plan + instruction without requiring user input. This avoids the dead state where the handoff is injected but nothing triggers the main loop.

### 9. TUI renders approval as a prompt line

Following existing conventions (like batch review):
```
  ↳ Plan saved to ~/.bono/.../plans/1709913600-add-auth.md
  ↳ Press Enter to implement, Esc to skip, or type feedback to revise [Enter/Esc]
```

The input bar accepts free text. Enter with empty input = approve. Enter with text = revise. Esc = reject.

## Risks / Trade-offs

- [Disk accumulation] Plans accumulate with no cleanup → Acceptable for small text files; future plan management will address.
- [Revision overwrites] If the user wants to compare revisions, the old content is lost → Acceptable; git-level history in `~/.bono` is a future concern.
- [Callback nil check] If `OnSubAgentApproval` is nil (headless, tests), ApprovalHook auto-approves → Correct default for non-interactive contexts.
- [Hook ordering matters] PersistHook must run before ApprovalHook so the file path is available in the approval prompt → Document that hook order matches registration order.

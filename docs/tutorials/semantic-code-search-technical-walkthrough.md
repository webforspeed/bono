# Semantic Code Search: Technical Walkthrough

This tutorial is the first place to start before changing semantic code search behavior.

Goal:
- understand the full control flow
- identify the moving pieces
- know exactly where to edit for common changes

## Step 1: Start from the TUI entrypoint

Open `bono/main.go`.

What happens:
1. Bono builds `core.Config` with `CodeSearch` settings.
2. Bono calls `core.NewAgent(config)`.
3. Core initializes and registers `code_search` (if configured and init succeeds).
4. Bono renders status from `agent.CodeSearchService().CodeSearchStats()`.

Why this matters:
- if code search is missing, this is where config and startup warnings are surfaced first.

## Step 2: Follow `/index` command flow

Open `bono/tui/slash_commands.go` and find `handleIndex`.

Flow:
1. user runs `/index`
2. TUI gets `svc := m.agent.CodeSearchService()`
3. TUI calls `svc.CodeSearchIndex(ctx, ".", ...)`
4. progress callback emits `IndexProgressMsg` (`scanning`, `chunking`, `embedding`, `storing`)
5. UI receives `IndexDoneMsg` and updates status + watcher state

Why this matters:
- indexing UX changes should stay in Bono TUI
- indexing behavior itself should stay in `bono-core`

## Step 3: Follow query execution (`code_search` tool)

Open:
- `bono-core/agent.go`
- `bono-core/code_search.go`
- `bono-core/tool_code_search.go`

Flow:
1. model emits tool call `code_search`
2. agent loop dispatches uniformly through `registry.Get(name).Execute(args)`
3. `CodeSearchTool` validates args and delegates to service
4. service parses search options and calls engine search
5. results are formatted and returned as `ToolResult`
6. Bono TUI shows tool trace and status via `OnToolCall` / `OnToolDone`

Why this matters:
- `code_search` is not a special execution path; it uses the same tool pipeline as other tools.

## Step 4: Understand core search engine pieces

Open `bono-core/codesearch/`:
- `engine.go`: search strategy selection (`semantic`, `hybrid`, `exact`)
- `indexer.go`: file scanning, chunk generation, embedding, persistence
- `store.go`: sqlite schema + FTS + vector operations
- `chunker.go`: code chunk boundaries/symbol extraction
- `embedder.go`: embedding API calls and batching
- `types.go`: option and progress contracts

Mental model:
- index-time pipeline: `scan -> chunk -> embed -> store`
- query-time pipeline: `parse options -> strategy -> rank -> format`

## Step 5: Know the search modes

- `semantic`: vector similarity first
- `hybrid`: vector + FTS merged ranking
- `exact`: FTS only

If sqlite vector support is unavailable:
- core degrades semantic/hybrid toward text behavior
- Bono surfaces startup warning

## Step 6: Use this change map for future edits

If you need to change...

- tool schema/args:
  - edit `bono-core/tool_code_search.go`

- option parsing from tool args:
  - edit `bono-core/code_search.go` (`parseCodeSearchOptions`)

- ranking/strategy behavior:
  - edit `bono-core/codesearch/engine.go` (+ `rrf.go` if hybrid merge changes)

- indexing behavior/chunk boundaries:
  - edit `bono-core/codesearch/indexer.go` and `chunker.go`

- persistence/index schema:
  - edit `bono-core/codesearch/store.go`

- TUI display/progress text:
  - edit `bono/tui/slash_commands.go` and `bono/tui/update.go`

## Step 7: Validate after each change

From `bono-core`:
```bash
go test ./...
```

From `bono`:
```bash
go test ./...
```

Then smoke test in TUI:
1. clear index (optional): `rm -f .bono/index.db .bono/index.db-shm .bono/index.db-wal`
2. run Bono
3. run `/index`
4. ask for a `code_search` query with each mode (`semantic`, `hybrid`, `exact`)

## Quick troubleshooting

- `code search unavailable` at startup:
  - check API key/base URL/config and startup logs

- always `0 results`:
  - ensure index exists and is current (`/index`)
  - test with `exact` mode first

- stale results:
  - watcher warning means files changed after last index; rerun `/index`

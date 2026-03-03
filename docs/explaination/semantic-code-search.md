# Semantic Code Search in Bono

## Architecture

Semantic code search behavior lives in `bono-core`. Bono TUI only triggers indexing (`/index`) and renders status/tool traces.

Core pieces:
- `core.NewAgent(config)` optionally enables code search via `Config.CodeSearch`
- `CodeSearchService` owns index/search lifecycle
- `CodeSearchTool` is registered in the normal tool registry path
- `code_search` executes through the same `registry.Get(name).Execute(args)` flow as other tools

## Lifecycle

1. Bono builds `core.Config` and sets `Config.CodeSearch`.
2. `core.NewAgent` initializes `CodeSearchService`.
3. If service init succeeds, core registers `code_search` in the registry.
4. Bono reads index stats for status bar display.
5. User runs `/index` to build/refresh index for the current workspace.
6. During chat, model invokes `code_search` as needed.

## Search Modes

- `semantic`: vector similarity search by intent/meaning.
- `hybrid`: vector similarity merged with keyword/FTS ranking.
- `exact`: keyword/FTS only.

TUI traces this as `Search('<query>', <search_type>)`.

## Operational Scenarios

### No index exists

- Service and tool can still be available.
- Search results are typically empty until indexing runs.
- TUI prompts: `No index found. Run /index to enable semantic code search.`

### Files changed after indexing

- File watcher reports changed file count.
- Index does not auto-refresh.
- User reruns `/index` to sync semantic index with workspace state.

### Vector support unavailable (sqlite-vec missing)

- Startup warning indicates text-only behavior.
- `semantic` and `hybrid` degrade toward FTS/text behavior.

### Code search initialization fails

- Bono shows startup warning (`code search unavailable`).
- `code_search` is not registered for that agent instance.

## Reindex and Reset

Rebuild in place:
- Run `/index`.

Clear index and rebuild from scratch:

```bash
rm -f .bono/index.db .bono/index.db-shm .bono/index.db-wal
```

Then run `/index` again.

## Config Surface

- `OPENROUTER_API_KEY`: required for embeddings.
- `BASE_URL`: API base URL override.
- `EMBEDDING_MODEL`: embedding model override.
- `EMBEDDING_DIMS`: embedding dimensions override.

## Practical Reading Order

1. `bono/main.go` (config wiring + status behavior)
2. `bono/tui/slash_commands.go` (`/index` path)
3. `bono-core/config.go` (`CodeSearch` config)
4. `bono-core/agent.go` (service init + tool registration)
5. `bono-core/code_search.go` (service internals + tool adapter)
6. `bono-core/tool_code_search.go` (tool schema and contract)

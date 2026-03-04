# How Web Search Works in Bono

## Overview

WebSearch and WebFetch are tools that give the agent live web access. WebSearch routes a query to either a **search backend** (returns a ranked list of URLs) or an **answer backend** (returns a synthesized answer with citations). WebFetch reads and summarizes a specific URL.

Both tools are implemented in `bono-core/web.go` and registered via `WebService`.

---

## Tool Arguments

### WebSearch

| Param | Required | Description |
|-------|----------|-------------|
| `query` | yes | Natural language query |
| `mode` | no | `"search"` or `"answer"` — bypasses the classifier when set |

When `mode` is omitted the classifier decides which backend to use. The model should set `mode` explicitly when it already knows what kind of result it needs — a URL list vs a synthesized answer.

### WebFetch

| Param | Required | Description |
|-------|----------|-------------|
| `url` | yes | URL to fetch |
| `question` | no | Focuses the summary on a specific question |

---

## Lifecycle: WebSearch call

```
WebSearch(query, mode?)
    │
    ├─ mode="search"  ──────────────────────────► openRouterSearcher
    ├─ mode="answer"  ──────────────────────────► sonarAnswerer
    └─ mode=""
           │
           └─ llmClassifier (gpt-4o-mini, max_tokens=50)
                  │
                  ├─ "prefer_search" ────────────► openRouterSearcher
                  ├─ "prefer_answer" ────────────► sonarAnswerer
                  └─ error / empty  ─────────────► sonarAnswerer (fallback)
```

Results include a `<hint>` tag guiding the model toward follow-up actions — see [Progressive Disclosure](#progressive-disclosure) below.

---

## Backends

### openRouterSearcher (search mode)

Sends the query to the main agent model via the **OpenRouter web plugin**:

```json
"plugins": [{"id": "web", "engine": "exa", "max_results": 5}]
```

OpenRouter injects live web results into the model's context. Citations are returned as `annotations` in the response (`type: "url_citation"`, nested under `url_citation.url`). The searcher maps these into `[]SearchResult` — title, URL, optional snippet.

When no citations come back (e.g. the model returned prose instead), it falls back to a single result containing the full response text.

### sonarAnswerer (answer mode)

Sends the query directly to `perplexity/sonar`. Sonar has built-in web access and returns a grounded answer with inline citations. The `[1][2]...` markers in the response body correspond to `annotations` that are extracted as source URLs.

### sonarFetcher (WebFetch)

Prompts `perplexity/sonar` to read and summarize the content at a URL. If a `question` param is provided, the prompt is focused: *"Read the content at X and answer: Y"*.

---

## Query Classifier

Used only when `mode` is not set. Calls `openai/gpt-4o-mini` with `max_tokens=50` and a one-sentence system prompt asking for `prefer_search` or `prefer_answer`. Cheap and fast — the model just outputs a single word.

Fallback is always `answer` (sonar) on error or unrecognized output.

---

## Vendor Swappability

Each capability is behind an interface:

```go
WebSearcher  — Search(ctx, query) ([]SearchResult, error)
WebAnswerer  — Answer(ctx, query) (answer, sources, error)
WebFetcher   — Fetch(ctx, url, question) (content, error)
QueryClassifier — Classify(ctx, query) (Backend, error)
```

Swapping to Tavily, Jina, Brave, etc. means implementing the relevant interface and passing it to `WebService` — no changes to routing, formatting, or tool definitions.

---

## Progressive Disclosure

Each result format includes a `<hint>` embedded in `ToolResult.Output`. This text is sent back to the LLM as a `role: "tool"` message on the next turn, nudging it toward appropriate follow-up actions without hardcoding behaviour in the system prompt.

| Result type | Hint |
|-------------|------|
| Search results | Call `WebFetch` on a URL for full content, or call `WebSearch` again with `mode="answer"` for a synthesized answer |
| Answer | Call `WebSearch` again with `mode="search"` for raw URLs and primary sources |
| Fetch | Call `WebSearch` if more pages on the topic are needed |

The hints are minimal by design — they name the exact tool and parameter the model should use, without over-explaining. The model decides whether to follow them based on the task.

> **Note:** `<hint>` text appears in `request_payload.messages` (as a tool role message) in the *next* API call in `api_calls.jsonl`, not in the response payload of the web call that produced it.

---

## Configuration (`WebConfig`)

| Field | Default | Override |
|-------|---------|----------|
| `Model` | `perplexity/sonar` | Answer, fetch, and classifier base |
| `SearchModel` | inherits main agent model | Model used with web plugin |
| `SearchEngine` | `exa` | OpenRouter web plugin engine |
| `MaxResults` | `5` | Max citations from web plugin |
| `ClassifierModel` | `openai/gpt-4o-mini` | Model for query routing |
| `APIKey` | inherits main config | OpenRouter API key |
| `APILogPath` | inherits main config | JSONL log path |

All web API calls (classifier + search/answer/fetch) are logged to the same JSONL file as main agent calls via a `loggingProvider` wrapper with a `capturingTransport`.

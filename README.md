# bono - An autonomous agent in the terminal

A terminal coding agent frontend written in Go. Bono provides the fullscreen TUI, the headless prompt mode, shared session frontends, and the user-facing approval flow, while `bono-core` handles the agent loop and tool execution.

## Screenshot
![bono screenshot](./docs/assets/screenshot_2.png)

## Slash Commands

| Command | Description |
|---------|-------------|
| `/index` | Index codebase for semantic code search |
| `/plan` | Launch a planning subagent with its own context window to think through architecture and approach |
| `/reasoning` | Set reasoning effort (`minimal`, `low`, `medium`, `high`, `xhigh`) |
| `/model` | Switch LLM at runtime |
| `/clear` | Clear conversation history and reset cost/context meter |

## Features
- **Modes:** Fullscreen TUI by default, plus headless prompt mode via `bono -p "..."` / `bono --prompt "..."`.
- **Automation flag:** `--skip-approvals` disables approval prompts and runtime limits for fully unattended runs.
- **Slash:** Slash-command-first UX (`/init`, `/index`, `/model`, `/spinner`, `/clear`, `/help`, `/exit`)
- **BYOK:** OpenRouter BYOK support via `OPENROUTER_API_KEY`
- **Local Models:** Auto-discovers local Ollama models and exposes them in `/model`
- **Search:** Semantic code search with vector indexing and repo stats in the status row
- **Chunking:** AST-based chunking/indexing pipeline (powered by `bono-core`)
- **Sandbox:** Default sandboxed command execution with approval fallbacks for unsandboxed runs
- **Runtime:** Programmatic tool calling with `python_runtime` for complex multi-step workflows that save context window
- **Compaction:** Intelligent context compaction to reduce risk of hitting context limits
- **Telemetry:** Live context and cost telemetry in the TUI, with shared session events available to headless and future frontends
- **Models:** Switch LLMs at runtime via slash command (`/model`)
- **Reasoning:** Configurable reasoning effort via `/reasoning` — supports `minimal`, `low`, `medium`, `high`, and `xhigh` levels.
- **Streaming:** Live token-by-token response streaming with real-time reasoning and content deltas
- **Web:** Live web access via `WebSearch` (search mode returns ranked URLs, answer mode returns a synthesized answer with citations) and `WebFetch` (reads and summarizes a URL). Auto-routes between modes using a fast LLM classifier; model can override with `mode="search"` or `mode="answer"`
- **Planning:** Dedicated planning subagent mode (`/plan`) for thinking through architecture and breaking down tasks before writing code

## Tools

| Tool | What it does |
|------|-------------|
| `read_file` | Read any file in the project |
| `write_file` | Create or overwrite a file |
| `edit_file` | Apply targeted edits to specific sections of a file |
| `run_shell` | Run shell commands in a sandbox (falls back to approval if unsandboxed) |
| `python_runtime` | Execute Python snippets for data processing, scripting, and multi-step workflows |
| `code_search` | Semantic, hybrid, and exact code search across the indexed codebase |
| `web_search` | Search the web — returns ranked URLs or a synthesized answer with citations |
| `web_fetch` | Fetch and summarize a URL, optionally focused on a specific question |
| `compact_context` | Summarize conversation history to free up context window |
| `enter_plan_mode` | Launch a planning subagent to think through architecture and produce a structured plan before coding |

## Install (GitHub Releases)
Install the latest release with:

```bash
curl -fsSL https://raw.githubusercontent.com/webforspeed/bono/main/install | bash
```

This installs `bono` to `~/.local/bin/bono`.

## Local Install / Deploy
Build and install from source:

```bash
make deploy
```

By default this installs `bono` to `~/.local/bin/bono`. If `~/.local/bin` is not on your `PATH`, add this to your shell config (`~/.zshrc`, `~/.bashrc`, etc.):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Publishing Releases
Tag pushes matching `v*` trigger `.github/workflows/release.yml`.

Example:

```bash
make release TAG=v0.1.0
```

## Run

```bash
go run .
```

Run a single prompt in headless mode:

```bash
go run . -p "Find and fix the bug in auth.py"
```

Installed binary usage:

```bash
bono -p "Find and fix the bug in auth.py"
```

Run without approval prompts or runtime limits:

```bash
bono --skip-approvals
```

## Model Providers (OpenRouter + Ollama)

Bono supports both remote OpenRouter models and local Ollama models in the same `/model` picker.

- OpenRouter models require `OPENROUTER_API_KEY`.
- Ollama models are discovered from `http://127.0.0.1:11434/api/tags`.
- Ollama chat requests use the OpenAI-compatible endpoint `http://127.0.0.1:11434/v1`.
- You can switch between remote and local models at runtime with `/model`.

### OpenRouter setup

```bash
export OPENROUTER_API_KEY="your-key"
```

### Ollama setup

1. Install and run Ollama locally.
2. Pull at least one model (example: `ollama pull qwen3-coder-next`).
3. Start Bono and run `/model` to select an Ollama model.

Optional: force local-by-default startup with environment variables:

```bash
export MODEL="qwen3-coder-next:latest"
export BASE_URL="http://127.0.0.1:11434/v1"
```

## Notes
- `OPENROUTER_API_KEY` is required only for remote OpenRouter models.
- Ollama can be used without `OPENROUTER_API_KEY` when local models are available.
- Detailed local setup guide: `docs/how-to/use-ollama-models.md`.
- Set `SHELL_TIMEOUT_SEC` to change the default timeout for `run_shell` and `python_runtime` commands.
- `--skip-approvals` auto-approves tool, sandbox fallback, change-batch, and subagent plan approvals in both TUI and headless mode.
- `--skip-approvals` also removes guardrails like turn limits and tool-call limits; use only in trusted environments.
- Bono status footer shows build mode/version: `Bono (dev)` for local builds and `Bono vX.Y.Z` for release builds.
- Bono checks GitHub releases in the background and shows `new version available` in the footer for newer tags.
- Set `BONO_DISABLE_UPDATE_CHECK=1` to skip update checks.
- In headless mode, Bono streams the same session events into the terminal transcript and uses inline approval prompts like `Approve? [y/N]`.
- Bono repo owns terminal-facing UX behavior and session frontends; `bono-core` owns agent loop, tools, and web/tool internals.

## Vision and Philosophy
Bono is built around a simple thesis: the best coding agents do not need heavy scaffolding, sprawling configuration, or vendor lock-in. Models are already highly capable and getting better quickly, so the harness should stay small, portable, and opinionated only where it meaningfully improves the experience.

- **Minimal top-level system prompt:** The main agent runs on a deliberately small system prompt (~[100 tokens](./prompts/versions/v1.0.5.tmpl)) so the model does the heavy reasoning rather than following a script. Less scaffolding also means behavior stays portable across models — what works on one capable model should work on another without prompt surgery.
- **Targeted subagent prompts:** Subagents are different. Because each one has a narrow, well-defined job — planning architecture, executing a focused search, producing a structured output — they get a tight, task-specific system prompt that keeps them on-track without the overhead of general-purpose reasoning context. The philosophy isn't "always minimal" — it's *right-sized*: lean at the top where generality matters, precise at the edge where outcomes are specific.
- **Single-binary experience:** The logic and scaffolding needed to operate Bono should ship in one binary. You should not need to configure the agent to search for skills in special directories, depend on tools that may or may not be installed, or edit obscure config files just to get productive.
- **Minimal tool surface:** More tools are not always better. Every tool adds context and decision overhead, so Bono prefers a small set of tools that models actually use well instead of bloating the context window with rarely used capabilities.
- **Multi-host future:** Bono is not just a CLI idea. The goal is for the same agent experience to be available across multiple surfaces, including the terminal, desktop apps, extensions, mobile apps, and the web.
- **Model and vendor freedom:** Bono should work with any capable model, including locally hosted models, and should never be restricted to a single vendor.

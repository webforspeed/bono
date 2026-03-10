# How to Use Ollama Models in Bono

Use this guide when you want Bono to run against locally hosted models via Ollama.

## What Bono Does

- Discovers local models from `http://127.0.0.1:11434/api/tags`.
- Uses Ollama's OpenAI-compatible API at `http://127.0.0.1:11434/v1`.
- Shows discovered local models in the same `/model` picker as remote models.

## Quick Start

1. Start Ollama.
2. Pull at least one model, for example:
   ```bash
   ollama pull qwen3-coder-next
   ```
3. Start Bono.
4. Run `/model` and pick the Ollama entry (provider: `Ollama`).

## Environment Options

If you want local Ollama by default at startup, set:

```bash
export MODEL="qwen3-coder-next:latest"
export BASE_URL="http://127.0.0.1:11434/v1"
```

If you want remote OpenRouter models, set:

```bash
export OPENROUTER_API_KEY="your-key"
```

`OPENROUTER_API_KEY` is not required for local Ollama models.

## Notes

- If Ollama is not running, Bono simply won't add local models to the catalog.
- Switching models with `/model` updates the active model and endpoint at runtime.

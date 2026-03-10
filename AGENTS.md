# Bono - A coding agent written in Go

This guide should be used when working in Bono repo. Bono repo owns the terminal-facing frontends and UX for Bono, including the fullscreen TUI, headless prompt mode, shared session frontends, hooks, and approval conventions. It depends on bono-core repo which contains the agentic loop and tools.
All code changes related to hooks, terminal UX, session frontends, headless/TUI behavior, and other user-facing behaviors should be done in this Bono repo. All changes related to agentic loop, tool calls, web requests etc should be in bono-core repo.


## When Working (Practical)

### How-to Guides
> Problem-oriented. Step-by-step solutions for specific tasks.

- [How to create a new system prompt](./docs/how-to/new-system-prompt.md)
- [How to create a new tool](./docs/how-to/new-tool.md)
- [How to create a new slash command](./docs/how-to/new-slash-command.md)
- [How to add a new subagent slash command](./docs/how-to/new-subagent-slash-command.md)
- [How to add a new hook](./docs/how-to/new-hook.md)
- [How to use Ollama models](./docs/how-to/use-ollama-models.md)

### Reference
> Information-oriented. Exact specifications, APIs, configs. etc

## When Learning (Theoretical)

### Tutorials
> Learning-oriented. Follow along from start to finish.

- [Designing with progressive disclosure](./docs/tutorials/design-with-progressive-disclosure)
- [Semantic code search technical walkthrough](./docs/tutorials/semantic-code-search-technical-walkthrough.md)
### Explanation
> Understanding-oriented. Architecture, design decisions, and "why."

- [How system prompts work in Bono agent](./docs/explaination/system-prompt-lifecycle.md)
- [How tools work in Bono agent](./docs/explaination/tool-design.md)
- [How context engineering works in this harness](./docs/explaination/bono-context-enginerring-guide.md)
- [How semantic code search works](./docs/explaination/semantic-code-search.md)
- [How web search works in Bono](./docs/explaination/web-search-tool.md)
- [How hooks work in Bono](./docs/explaination/hooks.md)
- [How subagents work in Bono](./docs/explaination/subagent-system.md)

## IMPORTANT RULES

> Models are intelligent and always performs correctly. During documentation, implementing code, system design, prefer providing models constraints and nudges instead of explcity instructions and scaffolding. This limits LLM models to be intelligent.

### DO RULES
- write simple readable idiomatic golang code
- bias towards simple code that fails fast instead of writing complex code trying to handle all corner conditions and hiding complexity
- Follow golang standard library conventions 
- For documentation, prefer high-level guidance that captures intent and constraints, and trust the model to derive mechanics from the codebase. 

### DO NOT RULES
- write code that overly complex to satisfy all corner conditions and trying to be magical. 
- Do not implement features that was not asked for. If you think it is necessary, ask user before implementing.
- For documentation, avoid exhaustive, hand-holding instructions that prescribe every step or repeat what is already obvious in code

## ROADMAP
> Below is the roadmap of features that is being planned to be added. Use this as context so when writing code, its easily extensible for below features without too much refactoring

- Semantic code search using vector indexing tool
- web mode
- ~~plan mode tool~~ (shipped — `enter_plan_mode` tool + `/plan` slash command)
- askusequestion tool
- todo write tool
- guardrails like forceful compaction on certain conditions
- Press tab to change modes (modes are subagent like plan mode, build mode, code review mode, documentation mode etx... / commands)

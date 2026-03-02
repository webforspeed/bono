# Why prefer progressive disclosure

## Thesis
Models are very intelligent and are capable of making the right decision. LLM models just wants to use tools and by adding too much scaffolding, the model becomes less intelligent. 
To truly make the most of an LLM model, we should provide necessary constraints and guardrails and let the models figure out how to solve a problem or achive a task without explicitly forcing it.

## Some examples of progressive discloures used in Bono

- system prompt - Current [system prompts](../../prompts/versions/) is minimal with just setting in the identity for the model.
- tools constraints - Instead of specififying all tools, we just mention that it has access to tools and trust it to discover
- context constraints - Instead of forcing to perform compaction or summary, we just inform the model its operating in a limited context window. We will then provide nudges in tool responses when we hit 30% or 70% context usage, but we dont need to provide it in system prompt.
- read file tool constraint - when models dont ask for start or end line, we "nudge" the model saying its truncated and it can get more details if it wants. We do this in tool response under certain conditions and we dont mention explicitly in system prompt or tool description.
- max tool compaction - if there are too many tool calls, we nudge gently suggesting to execute compaction. instead of forcing it to execute.
- python_executution - Implements programmatic tool calling so models can call multiple tools in one container and return only whats relevant and pollute the context with intermididatite tool results.

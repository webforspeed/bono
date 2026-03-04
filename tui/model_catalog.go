package tui

func DefaultModelCatalog() []ModelInfo {
	return []ModelInfo{
{
	ID:           "anthropic/claude-sonnet-4.6",
	Name:         "Claude Sonnet 4.6",
	Provider:     "Anthropic",
	Capabilities: []string{"recommended", "balanced", "high cost"},
	Context:      "1M",
	Tier:         "balanced",
},
{
	ID:           "anthropic/claude-opus-4.6",
	Name:         "Claude Opus 4.6",
	Provider:     "Anthropic",
	Capabilities: []string{"recommended", "frontier", "very high cost"},
	Context:      "1M",
	Tier:         "frontier",
},
{
	ID:           "anthropic/claude-haiku-4.5",
	Name:         "Claude Haiku 4.5",
	Provider:     "Anthropic",
	Capabilities: []string{"recommended", "balanced", "mid cost"},
	Context:      "200K",
	Tier:         "balanced",
},

{
	ID:           "openai/gpt-5.3-chat",
	Name:         "GPT-5.3 Chat",
	Provider:     "OpenAI",
	Capabilities: []string{"recommended", "balanced", "high cost"},
	Context:      "128K",
	Tier:         "balanced",
},
{
	ID:           "openai/gpt-5.3-codex",
	Name:         "GPT-5.3 Codex",
	Provider:     "OpenAI",
	Capabilities: []string{"recommended", "frontier", "high cost"},
	Context:      "400K",
	Tier:         "frontier",
},

{
	ID:           "openai/gpt-oss-120b",
	Name:         "GPT-OSS 120B",
	Provider:     "OpenAI",
	Capabilities: []string{"very fast", "low intelligence", "ultra low cost"},
	Context:      "131K",
	Tier:         "mid",
},
{
	ID:           "openai/gpt-oss-20b",
	Name:         "GPT-OSS 20B",
	Provider:     "OpenAI",
	Capabilities: []string{"very fast", "very low intelligence", "ultra low cost"},
	Context:      "131K",
	Tier:         "mid",
},
{
	ID:           "openai/gpt-oss-safeguard-20b:nitro",
	Name:         "GPT-OSS Safeguard 20B (Nitro)",
	Provider:     "OpenAI",
	Capabilities: []string{"very fast", "very low intelligence", "ultra low cost"},
	Context:      "131K",
	Tier:         "mid",
},
{
	ID:           "qwen/qwen3-32b:nitro",
	Name:         "Qwen3 32B (Nitro)",
	Provider:     "Qwen",
	Capabilities: []string{"very fast", "very low intelligence", "ultra low cost"},
	Context:      "40,960",
	Tier:         "mid",
},

{
	ID:           "google/gemini-3.1-pro-preview",
	Name:         "Gemini 3.1 Pro (Preview)",
	Provider:     "Google",
	Capabilities: []string{"balanced", "frontier", "high cost"},
	Context:      "1.05M",
	Tier:         "frontier",
},
{
	ID:           "google/gemini-3-flash-preview",
	Name:         "Gemini 3 Flash (Preview)",
	Provider:     "Google",
	Capabilities: []string{"recommended", "balanced", "low cost"},
	Context:      "1.05M",
	Tier:         "balanced",
},

{
	ID:           "minimax/minimax-m2.5",
	Name:         "MiniMax M2.5",
	Provider:     "MiniMax",
	Capabilities: []string{"recommended", "balanced", "low cost"},
	Context:      "196K",
	Tier:         "balanced",
},
{
	ID:           "moonshotai/kimi-k2.5",
	Name:         "Kimi K2.5",
	Provider:     "MoonshotAI",
	Capabilities: []string{"recommended", "balanced", "low cost"},
	Context:      "262K",
	Tier:         "balanced",
},
{
	ID:           "deepseek/deepseek-v3.2",
	Name:         "DeepSeek V3.2",
	Provider:     "DeepSeek",
	Capabilities: []string{"recommended", "cheap", "ultra low cost"},
	Context:      "163K",
	Tier:         "mid",
},
{
	ID:           "z-ai/glm-5",
	Name:         "GLM-5",
	Provider:     "Z.ai",
	Capabilities: []string{"recommended", "frontier", "low cost"},
	Context:      "202K",
	Tier:         "frontier",
},


	}
}
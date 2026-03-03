package tui

// DefaultModelCatalog returns the built-in model list used by /model.
// Edit this file to add, remove, or change available models.
func DefaultModelCatalog() []ModelInfo {
	return []ModelInfo{
		{
			ID:           "anthropic/claude-opus-4",
			Name:         "Claude Opus 4",
			Provider:     "Anthropic",
			Capabilities: []string{"coding", "reasoning", "analysis", "vision"},
			Context:      "200K",
			Tier:         "flagship",
		},
		{
			ID:           "anthropic/claude-sonnet-4",
			Name:         "Claude Sonnet 4",
			Provider:     "Anthropic",
			Capabilities: []string{"coding", "reasoning", "fast"},
			Context:      "200K",
			Tier:         "balanced",
		},
		{
			ID:           "openai/gpt-4.1",
			Name:         "GPT-4.1",
			Provider:     "OpenAI",
			Capabilities: []string{"coding", "reasoning", "analysis", "vision"},
			Context:      "1M",
			Tier:         "flagship",
		},
		{
			ID:           "openai/gpt-4.1-mini",
			Name:         "GPT-4.1 Mini",
			Provider:     "OpenAI",
			Capabilities: []string{"coding", "fast"},
			Context:      "1M",
			Tier:         "balanced",
		},
		{
			ID:           "openai/o3",
			Name:         "O3",
			Provider:     "OpenAI",
			Capabilities: []string{"reasoning", "coding", "analysis"},
			Context:      "200K",
			Tier:         "flagship",
		},
		{
			ID:           "google/gemini-2.5-pro-preview",
			Name:         "Gemini 2.5 Pro",
			Provider:     "Google",
			Capabilities: []string{"coding", "reasoning", "analysis", "vision"},
			Context:      "1M",
			Tier:         "flagship",
		},
		{
			ID:           "google/gemini-2.5-flash-preview",
			Name:         "Gemini 2.5 Flash",
			Provider:     "Google",
			Capabilities: []string{"coding", "fast"},
			Context:      "1M",
			Tier:         "balanced",
		},
		{
			ID:           "deepseek/deepseek-r1",
			Name:         "DeepSeek R1",
			Provider:     "DeepSeek",
			Capabilities: []string{"reasoning", "coding"},
			Context:      "64K",
			Tier:         "balanced",
		},
		{
			ID:           "deepseek/deepseek-chat",
			Name:         "DeepSeek V3",
			Provider:     "DeepSeek",
			Capabilities: []string{"coding", "fast"},
			Context:      "64K",
			Tier:         "budget",
		},
	}
}

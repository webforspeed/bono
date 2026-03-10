package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaAPIURL is the default base URL for the Ollama API.
// Used for the /api/tags endpoint to list models.
const OllamaAPIURL = "http://127.0.0.1:11434"

// OllamaOpenAIBaseURL is the base URL for Ollama's OpenAI-compatible API.
// The /v1 prefix is required for OpenAI-compatible endpoints like /v1/chat/completions.
const OllamaOpenAIBaseURL = "http://127.0.0.1:11434/v1"

// OllamaModel represents a model from the Ollama API.
type OllamaModel struct {
	Name      string   `json:"name"`
	Model     string   `json:"model"`
	Modified  string   `json:"modified_at"`
	Size      int64    `json:"size"`
	Digest    string   `json:"digest"`
	Details   OllamaModelDetails `json:"details,omitempty"`
}

// OllamaModelDetails contains additional model information.
type OllamaModelDetails struct {
	ParentModel string   `json:"parent_model,omitempty"`
	Format      string   `json:"format,omitempty"`
	Family      string   `json:"family,omitempty"`
	Families    []string `json:"families,omitempty"`
	ParameterSize string `json:"parameter_size,omitempty"`
	QuantizationLevel string `json:"quantization_level,omitempty"`
}

// OllamaTagsResponse is the response from the /api/tags endpoint.
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// FetchOllamaModels queries the local Ollama API for available models.
// It returns nil if Ollama is not available.
func FetchOllamaModels(ctx context.Context) []ModelInfo {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", OllamaAPIURL+"/api/tags", nil)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var tagsResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil
	}

	models := make([]ModelInfo, 0, len(tagsResp.Models))
	for _, m := range tagsResp.Models {
		models = append(models, ModelInfo{
			ID:           m.Name,
			Name:         formatOllamaModelName(m.Name),
			Provider:     "Ollama",
			Capabilities: ollamaCapabilities(m),
			Context:      ollamaContextSize(m),
			Tier:         "local",
			BaseURL:      OllamaOpenAIBaseURL,
			IsLocal:      true,
		})
	}

	return models
}

// formatOllamaModelName converts "modelname:tag" to "Model Name (tag)".
func formatOllamaModelName(name string) string {
	// Handle "modelname:tag" format
	if idx := len(name) - 1; idx >= 0 {
		for i := len(name) - 1; i >= 0; i-- {
			if name[i] == ':' {
				modelName := name[:i]
				tag := name[i+1:]
				// Convert snake_case or kebab-case to Title Case
				displayName := toTitleCase(modelName)
				return fmt.Sprintf("%s (%s)", displayName, tag)
			}
		}
	}
	// No tag, just convert to title case
	return toTitleCase(name)
}

// toTitleCase converts "some-model-name" to "Some Model Name".
func toTitleCase(s string) string {
	result := make([]rune, 0, len(s))
	capitalizeNext := true
	for _, r := range s {
		if r == '-' || r == '_' || r == ' ' {
			capitalizeNext = true
			result = append(result, ' ')
			continue
		}
		if capitalizeNext {
			result = append(result, toUpper(r))
			capitalizeNext = false
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}

// ollamaCapabilities returns a list of capabilities for an Ollama model.
func ollamaCapabilities(m OllamaModel) []string {
	caps := []string{"local", "offline"}
	
	if m.Details.Family != "" {
		caps = append(caps, m.Details.Family)
	}
	
	if m.Details.ParameterSize != "" {
		caps = append(caps, m.Details.ParameterSize)
	}
	
	return caps
}

// ollamaContextSize returns a default context size for Ollama models.
// Ollama typically uses 4096 by default, but this varies by model.
func ollamaContextSize(m OllamaModel) string {
	// Return a reasonable default - actual context size varies by model
	return "4K-128K"
}

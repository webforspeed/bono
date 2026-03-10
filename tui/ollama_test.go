package tui

import (
	"context"
	"testing"
	"time"
)

func TestFetchOllamaModels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models := FetchOllamaModels(ctx)
	
	if len(models) == 0 {
		t.Log("No Ollama models found - Ollama may not be running")
		return
	}
	
	t.Logf("Found %d Ollama models:", len(models))
	for _, m := range models {
		t.Logf("  - %s (Provider: %s, ID: %s, BaseURL: %s, IsLocal: %v)", 
			m.Name, m.Provider, m.ID, m.BaseURL, m.IsLocal)
	}
	
	// Verify each model has the expected properties
	for _, m := range models {
		if m.Provider != "Ollama" {
			t.Errorf("Expected provider 'Ollama', got '%s'", m.Provider)
		}
		if !m.IsLocal {
			t.Errorf("Expected IsLocal to be true")
		}
		if m.BaseURL == "" {
			t.Errorf("Expected BaseURL to be set")
		}
	}
}

func TestLoadModelCatalog(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models := LoadModelCatalog(ctx)
	
	t.Logf("Total models in catalog: %d", len(models))
	
	// Check that we have remote models
	remoteCount := 0
	localCount := 0
	for _, m := range models {
		if m.IsLocal {
			localCount++
		} else {
			remoteCount++
		}
	}
	
	t.Logf("Remote models: %d, Local (Ollama) models: %d", remoteCount, localCount)
	
	if remoteCount == 0 {
		t.Error("Expected at least one remote model")
	}
	// localCount may be 0 if Ollama is not running
}

func TestFormatOllamaModelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"qwen3-coder-next:latest", "Qwen3 Coder Next (latest)"},
		{"gpt-oss:20b", "Gpt Oss (20b)"},
		{"nemotron-3-nano:30b", "Nemotron 3 Nano (30b)"},
		{"llama3", "Llama3"}, // No tag
		{"phi4:latest", "Phi4 (latest)"},
	}

	for _, tt := range tests {
		result := formatOllamaModelName(tt.input)
		if result != tt.expected {
			t.Errorf("formatOllamaModelName(%q) = %q; want %q", tt.input, result, tt.expected)
		}
	}
}

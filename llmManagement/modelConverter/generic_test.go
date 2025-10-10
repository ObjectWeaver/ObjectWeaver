package modelConverter

import (
	"testing"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

func TestProviderModelConverter_Convert(t *testing.T) {
	converter := NewModelConverter()

	tests := []struct {
		name     string
		input    jsonSchema.ModelType
		expected string
	}{
		// OpenAI Models
		{"Gpt3", jsonSchema.Gpt3, "gpt-3.5-turbo"},
		{"Gpt4", jsonSchema.Gpt4, "gpt-4-turbo"},
		{"Gpt4Mini", jsonSchema.Gpt4Mini, "gpt-4o-mini"},

		// Anthropic Models
		{"ClaudeSonnet", jsonSchema.ClaudeSonnet, "claude-3-sonnet-20240229"},
		{"ClaudeHaiku", jsonSchema.ClaudeHaiku, "claude-3-haiku-20240307"},

		// Google Gemini Models
		{"GeminiPro", jsonSchema.GeminiPro, "gemini-2.5-pro"},
		{"GeminiFlash", jsonSchema.GeminiFlash, "gemini-2.5-flash-lite"},
		{"GeminiFlash2", jsonSchema.GeminiFlash2, "gemini-2.5-flash"},
		{"GeminiFlash2Lite", jsonSchema.GeminiFlash2Lite, "gemini-2.0-flash-lite-001"},
		{"GeminiFlash8B", jsonSchema.GeminiFlash8B, "gemini-2.0-flash-lite-001"},

		// Meta Llama Models
		{"Llama8b", jsonSchema.Llama8b, "llama3-8b-8192"},
		{"Llama8bInstant", jsonSchema.Llama8bInstant, "llama3-8b-8192"},
		{"Llama70b", jsonSchema.Llama70b, "llama3-70b-8192"},
		{"Llama70bVersatile", jsonSchema.Llama70bVersatile, "llama3-70b-8192"},

		// Provider-specific or Custom Models
		{"O1", jsonSchema.O1, "o1-preview"},
		{"O1Mini", jsonSchema.O1Mini, "o1-mini"},

		// Hypothetical or less common models
		{"Llama1B", jsonSchema.Llama1B, "llama-1b-chat"},
		{"Llama3B", jsonSchema.Llama3B, "llama-3b-chat"},
		{"Llama405b", jsonSchema.Llama405b, "llama-405b-chat"},

		// Unknown models - pass-through
		{"UnknownModel", jsonSchema.ModelType("unknown-model"), "unknown-model"},
		{"EmptyString", jsonSchema.ModelType(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if result != tt.expected {
				t.Errorf("Convert(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

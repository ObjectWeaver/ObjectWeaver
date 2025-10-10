package modelConverter

import "github.com/henrylamb/object-generation-golang/jsonSchema"

// ModelConverter is an interface for converting a local ModelType to its provider-specific string.
type ModelConverter interface {
	// Convert takes a local ModelType and returns the corresponding provider-specific model string.
	// If the model is not recognized, it returns an empty string.
	Convert(model jsonSchema.ModelType) string
}

// providerModelConverter is a concrete implementation of the ModelConverter interface.
// It maps local ModelType enums to their official string representations for various LLM providers.
type providerModelConverter struct{}

// NewModelConverter creates and returns a new instance of the providerModelConverter.
func NewModelConverter() ModelConverter {
	return &providerModelConverter{}
}

// Convert implements the logic to translate a ModelType into the correct API string.
// It handles known model constants and falls back to treating unknown models as
// pass-through strings, enabling environment variable configuration and custom models.
func (c *providerModelConverter) Convert(model jsonSchema.ModelType) string {
	modelStr := string(model)

	// Standard model conversions
	switch model {
	// OpenAI Models
	case jsonSchema.Gpt3:
		return "gpt-3.5-turbo"
	case jsonSchema.Gpt4:
		return "gpt-4-turbo"
	case jsonSchema.Gpt4Mini:
		return "gpt-4o-mini"

	// Anthropic Models
	case jsonSchema.ClaudeSonnet:
		return "claude-3-sonnet-20240229"
	case jsonSchema.ClaudeHaiku:
		return "claude-3-haiku-20240307"

	// Google Gemini Models
	case jsonSchema.GeminiPro:
		return "gemini-2.5-pro"
	case jsonSchema.GeminiFlash:
		return "gemini-2.5-flash-lite"
	case jsonSchema.GeminiFlash2:
		return "gemini-2.5-flash"
	case jsonSchema.GeminiFlash2Lite:
		return "gemini-2.0-flash-lite-001"
	case jsonSchema.GeminiFlash8B:
		return "gemini-2.0-flash-lite-001"

	// Meta Llama Models (often used via providers like Groq, Together.ai, etc.)
	case jsonSchema.Llama8b, jsonSchema.Llama8bInstant:
		return "llama3-8b-8192"
	case jsonSchema.Llama70b, jsonSchema.Llama70bVersatile:
		return "llama3-70b-8192"

	// Provider-specific or Custom Models
	case jsonSchema.O1:
		return "o1-preview"
	case jsonSchema.O1Mini:
		return "o1-mini"

	// Hypothetical or less common models - returning a placeholder
	case jsonSchema.Llama1B:
		return "llama-1b-chat"
	case jsonSchema.Llama3B:
		return "llama-3b-chat"
	case jsonSchema.Llama405b:
		return "llama-405b-chat" // Placeholder, actual name may vary

	// Pass-through for any unhandled case
	// This enables custom models, environment variable overrides, and future models
	// without requiring code changes. Simply treat the ModelType string value as-is.
	default:
		return modelStr
	}
}

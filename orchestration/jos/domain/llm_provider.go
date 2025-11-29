package domain

import (
	"context"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// LLMProvider - Abstract LLM interaction for generating content
//
// Implementation: infrastructure/llm/openai_provider.go (OpenAIProvider)
// Created by: factory/generator_factory.go:createLLMProvider()
// Used by: TypeProcessor implementations
//
// Responsibilities:
//   - Generate text using LLM (wraps job submission system)
//   - Provide capability information (streaming support, model type)
//   - Return generation metadata (tokens, cost, finish reason)
//
// Extensions:
//   - TokenStreamingProvider: For token-by-token streaming
//   - ByteOperationProvider: For TTS, Image generation, STT
type LLMProvider interface {
	Generate(prompt string, config *GenerationConfig) (any, *ProviderMetadata, error)
	SupportsStreaming() bool
	ModelType() string
}

// TokenStreamingProvider - LLM that supports token streaming
type TokenStreamingProvider interface {
	LLMProvider
	GenerateStream(prompt string, config *GenerationConfig) (<-chan any, error)
	GenerateTokenStream(prompt string, config *GenerationConfig) (<-chan *TokenChunk, error)
	SupportsTokenStreaming() bool
}

// ByteOperationProvider - Provider that supports byte operations (TTS, Image, STT)
type ByteOperationProvider interface {
	GenerateAudio(request *AudioGenerationRequest) ([]byte, *ProviderMetadata, error)
	GenerateImage(request *ImageGenerationRequest) ([]byte, *ProviderMetadata, error)
	TranscribeAudio(request *AudioTranscriptionRequest) (string, *ProviderMetadata, error)
	SupportsByteOperations() bool
}

// AudioGenerationRequest contains parameters for text-to-speech
type AudioGenerationRequest struct {
	Model          string
	Input          string
	Voice          string
	ResponseFormat string
	Speed          float64
}

// ImageGenerationRequest contains parameters for image generation
type ImageGenerationRequest struct {
	Model          string
	Prompt         string
	Size           string
	ResponseFormat string
	N              int
}

// AudioTranscriptionRequest contains parameters for speech-to-text
type AudioTranscriptionRequest struct {
	Model          string
	AudioData      []byte
	Language       string
	ResponseFormat string
	Prompt         string
}

// ProviderMetadata contains metadata from LLM provider
type ProviderMetadata struct {
	TokensUsed   int
	Cost         float64
	Model        string
	FinishReason string
	Prompt       string
	Choices      []Choice
	VerboseData map[string]any
}

type Choice struct {
	Prompt     string
	Completion any
	FieldTask  FieldTask
	Score      int
	Confidence float64
	Model      string
	Embedding  []float64
}

// GenerationConfig configures LLM generation
type GenerationConfig struct {
	Context       context.Context // Context for request cancellation
	Model         string
	Temperature   float64
	MaxTokens     int
	SystemPrompt  string
	Granularity   StreamGranularity
	BufferSize    int
	StopSequences []string
	Definition    *jsonSchema.Definition // Schema definition for this generation (includes SendImage, etc.)
}

func DefaultGenerationConfig() *GenerationConfig {
	return &GenerationConfig{
		Temperature:   0.7,
		MaxTokens:     2000,
		Granularity:   GranularityField,
		BufferSize:    10,
		StopSequences: []string{},
	}
}

package llm

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	byteoperations "objectweaver/llmManagement/byteOperations"
	"objectweaver/orchestration/jobSubmitter"
	"objectweaver/orchestration/jos/domain"

	gogpt "github.com/sashabaranov/go-openai"
)

// OpenAIProvider adapts the existing job submitter to the LLMProvider interface
type OpenAIProvider struct {
	submitter    jobSubmitter.JobEntryPoint
	defaultModel string
	client       *gogpt.Client
}

func NewOpenAIProvider() *OpenAIProvider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Printf("[OpenAIProvider WARNING] OPENAI_API_KEY not set, byte operations (image/audio) will fail")
	}

	// Create OpenAI client configuration
	config := gogpt.DefaultConfig(apiKey)

	// Note: OpenAI byte operations don't support custom URLs like LLM_API_URL
	// Those are only for text completion APIs

	return &OpenAIProvider{
		submitter:    jobSubmitter.NewDefaultJobEntryPoint(),
		defaultModel: getDefaultModelForProvider(),
		client:       gogpt.NewClientWithConfig(config),
	}
}

// getDefaultModelForProvider returns the appropriate default model based on LLM_PROVIDER
func getDefaultModelForProvider() string {
	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))

	switch provider {
	case "gemini":
		return "gemini-2.0-flash"
	case "openai":
		return "gpt-4o-mini"
	case "local":
		// For local, try to detect from URL or default to a common model
		return "gpt-4o-mini"
	default:
		// If provider not set or unknown, check if we have an API URL (local)
		if os.Getenv("LLM_API_URL") != "" {
			return "gpt-4o-mini"
		}
		// Default to Gemini Flash if using Gemini keys
		if os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("LLM_API_KEY") != "" {
			return "gemini-2.0-flash"
		}
		// Final fallback
		return "gpt-4o-mini"
	}
}

func (p *OpenAIProvider) Generate(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
	// Determine model - use config model if provided, otherwise use provider-specific default
	model := config.Model
	if model == "" {
		model = p.defaultModel
	}

	// Log the request details in development mode
	log.Printf("[LLM] Submitting job with model: %s, prompt length: %d chars", model, len(prompt))

	// Submit job with Definition (includes SendImage if present)
	completion, _, err := p.submitter.SubmitJob(model, config.Definition, prompt, config.SystemPrompt, nil)
	if err != nil {
		log.Printf("[LLM ERROR] Job submission failed: %v", err)
		return "", nil, fmt.Errorf("job submission failed: %w", err)
	}

	metadata := &domain.ProviderMetadata{
		Model: string(model),
	}

	return completion, metadata, nil
}

func (p *OpenAIProvider) SupportsStreaming() bool {
	return false // Base provider doesn't support streaming yet
}

func (p *OpenAIProvider) ModelType() string {
	return p.defaultModel
}

func calculateCost(model string, usage interface{}) float64 {
	// Simplified cost calculation
	// In reality, this would use the cost package
	return 0.0
}

// StreamingOpenAIProvider with token streaming support
type StreamingOpenAIProvider struct {
	*OpenAIProvider
}

func NewStreamingOpenAIProvider() *StreamingOpenAIProvider {
	return &StreamingOpenAIProvider{
		OpenAIProvider: NewOpenAIProvider(),
	}
}

func (p *StreamingOpenAIProvider) GenerateStream(prompt string, config *domain.GenerationConfig) (<-chan any, error) {
	// Placeholder for streaming implementation
	// This would use the actual streaming API
	out := make(chan any)

	go func() {
		defer close(out)

		// For now, simulate by breaking up response
		response, _, err := p.Generate(prompt, config)
		if err != nil {
			return
		}

		// Send as single chunk
		out <- response
	}()

	return out, nil
}

func (p *StreamingOpenAIProvider) GenerateTokenStream(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error) {
	out := make(chan *domain.TokenChunk, 100)

	go func() {
		defer close(out)

		// Get string stream
		stringStream, err := p.GenerateStream(prompt, config)
		if err != nil {
			return
		}

		// Convert to token chunks
		for str := range stringStream {
			// Simulate token-by-token streaming
			// In reality, this would come directly from the LLM API
			for _, char := range str.(string) {
				chunk := domain.NewTokenChunk(string(char))
				out <- chunk
			}
		}

		// Send final marker
		finalChunk := domain.NewTokenChunk("")
		finalChunk.IsFinal = true
		out <- finalChunk
	}()

	return out, nil
}

func (p *StreamingOpenAIProvider) SupportsTokenStreaming() bool {
	return true
}

func (p *StreamingOpenAIProvider) SupportsStreaming() bool {
	return true
}

// ============================================================================
// ByteOperationProvider Interface Implementation
// ============================================================================

// GenerateAudio implements text-to-speech functionality
func (p *OpenAIProvider) GenerateAudio(request *domain.AudioGenerationRequest) ([]byte, *domain.ProviderMetadata, error) {
	log.Printf("[OpenAIProvider] Generating audio with TTS...")
	log.Printf("[OpenAIProvider] Input: %s", request.Input)
	log.Printf("[OpenAIProvider] Voice: %s", request.Voice)
	log.Printf("[OpenAIProvider] Model: %s", request.Model)

	// Create OpenAI TTS request
	ttsReq := gogpt.CreateSpeechRequest{
		Model:          gogpt.SpeechModel(request.Model),
		Input:          request.Input,
		Voice:          gogpt.SpeechVoice(request.Voice),
		ResponseFormat: gogpt.SpeechResponseFormat(request.ResponseFormat),
		Speed:          request.Speed,
	}

	// Call OpenAI CreateSpeech API
	ctx := context.Background()
	response, err := p.client.CreateSpeech(ctx, ttsReq)
	if err != nil {
		log.Printf("[OpenAIProvider ERROR] TTS generation failed: %v", err)
		return nil, nil, fmt.Errorf("TTS generation failed: %w", err)
	}
	defer response.Close()

	// Read the audio data
	audioBytes, err := io.ReadAll(response)
	if err != nil {
		log.Printf("[OpenAIProvider ERROR] Failed to read TTS audio: %v", err)
		return nil, nil, fmt.Errorf("failed to read TTS audio: %w", err)
	}

	log.Printf("[OpenAIProvider] TTS generated successfully: %d bytes", len(audioBytes))

	// Create metadata
	metadata := &domain.ProviderMetadata{
		Model: request.Model,
		// TODO: Add cost calculation for TTS
		Cost:       0,
		TokensUsed: 0,
	}

	return audioBytes, metadata, nil
}

// GenerateImage implements image generation functionality
func (p *OpenAIProvider) GenerateImage(request *domain.ImageGenerationRequest) ([]byte, *domain.ProviderMetadata, error) {
	log.Printf("[OpenAIProvider] Generating image with DALL-E...")
	log.Printf("[OpenAIProvider] Prompt: %s", request.Prompt)
	log.Printf("[OpenAIProvider] Model: %s", request.Model)
	log.Printf("[OpenAIProvider] Size: %s", request.Size)

	// Convert string model to string
	modelType := string(request.Model)

	// Use the byteoperations package to generate the image with our client
	byteops := byteoperations.NewImageGenerator(p.client)
	imageBytes, err := byteops.GenerateImage(request.Prompt, modelType, request.Size)
	if err != nil {
		log.Printf("[OpenAIProvider ERROR] Image generation failed: %v", err)
		return nil, nil, fmt.Errorf("image generation failed: %w", err)
	}

	log.Printf("[OpenAIProvider] Image generated successfully: %d bytes", len(imageBytes))

	// Create metadata
	metadata := &domain.ProviderMetadata{
		Model: string(request.Model),
		// TODO: Add cost calculation for image generation
		Cost:       0,
		TokensUsed: 0,
	}

	return imageBytes, metadata, nil
}

// TranscribeAudio implements speech-to-text functionality
func (p *OpenAIProvider) TranscribeAudio(request *domain.AudioTranscriptionRequest) (string, *domain.ProviderMetadata, error) {
	log.Printf("[OpenAIProvider] Transcribing audio with Whisper...")
	log.Printf("[OpenAIProvider] Model: %s", request.Model)
	log.Printf("[OpenAIProvider] Language: %s", request.Language)
	log.Printf("[OpenAIProvider] Audio data size: %d bytes", len(request.AudioData))

	// Create a reader from the audio bytes
	audioReader := strings.NewReader(string(request.AudioData))

	// Create OpenAI Whisper request
	whisperReq := gogpt.AudioRequest{
		Model:    request.Model,
		Reader:   audioReader,
		Prompt:   request.Prompt,
		Format:   gogpt.AudioResponseFormat(request.ResponseFormat),
		Language: request.Language,
		FilePath: "audio.mp3", // Required by OpenAI SDK even though we're using Reader
	}

	// Call OpenAI CreateTranscription API
	ctx := context.Background()
	response, err := p.client.CreateTranscription(ctx, whisperReq)
	if err != nil {
		log.Printf("[OpenAIProvider ERROR] Audio transcription failed: %v", err)
		return "", nil, fmt.Errorf("audio transcription failed: %w", err)
	}

	log.Printf("[OpenAIProvider] Audio transcribed successfully: %s", response.Text)

	// Create metadata
	metadata := &domain.ProviderMetadata{
		Model: request.Model,
		// TODO: Add cost calculation for STT
		Cost:       0,
		TokensUsed: 0,
	}

	return response.Text, metadata, nil
}

// SupportsByteOperations returns true since OpenAI supports DALL-E image generation
// (and will support TTS/STT in the future)
func (p *OpenAIProvider) SupportsByteOperations() bool {
	return true
}

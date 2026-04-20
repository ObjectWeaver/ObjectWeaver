package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"github.com/ObjectWeaver/ObjectWeaver/logger"
	"os"
	"strings"
	"time"

	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jobSubmitter"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"

	gogpt "github.com/sashabaranov/go-openai"
)

// byteOpsClient abstracts the OpenAI client methods used for byte operations (TTS, image, STT)
// so they can be mocked in unit tests without making real API calls.
type byteOpsClient interface {
	CreateSpeech(ctx context.Context, request gogpt.CreateSpeechRequest) (gogpt.RawResponse, error)
	CreateImage(ctx context.Context, request gogpt.ImageRequest) (gogpt.ImageResponse, error)
	CreateTranscription(ctx context.Context, request gogpt.AudioRequest) (gogpt.AudioResponse, error)
}

// OpenAIProvider adapts the existing job submitter to the LLMProvider interface
type OpenAIProvider struct {
	submitter         jobSubmitter.JobEntryPoint
	defaultModel      string
	byteOps           byteOpsClient
	throughputManager *DefaultThroughputManager
	maxRetries        int
}

func NewOpenAIProvider() *OpenAIProvider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		logger.Printf("[OpenAIProvider WARNING] OPENAI_API_KEY not set, byte operations (image/audio) will fail")
	}

	// Create OpenAI client configuration
	config := gogpt.DefaultConfig(apiKey)

	// Note: OpenAI byte operations don't support custom URLs like LLM_API_URL
	// Those are only for text completion APIs

	//setup for the throughput manager here
	envModels := os.Getenv("THROUGHPUT_MODELS") // Comma-separated list of models to cycle through
	models := []string{}
	if envModels != "" {
		models = strings.Split(envModels, ",")
	}

	if StandardThroughputManager == nil {
		StandardThroughputManager = NewDefaultThroughputManager(models)
	}

	return &OpenAIProvider{
		submitter:         jobSubmitter.NewDefaultJobEntryPoint(),
		defaultModel:      getDefaultModelForProvider(),
		byteOps:           gogpt.NewClientWithConfig(config),
		throughputManager: StandardThroughputManager,
		maxRetries:        3,
	}
}

// getDefaultModelForProvider returns the appropriate default model based on LLM_PROVIDER
func getDefaultModelForProvider() string {
	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))

	switch provider {
	case "gemini":
		return "gemini-2.5-flash-lite"
	case "openai":
		return "gpt-4.1-nano-2025-04-14"
	case "local":
		// For local, try to detect from URL or default to a common model
		return "gpt-4.1-nano-2025-04-14"
	default:
		// If provider not set or unknown, check if we have an API URL (local)
		if os.Getenv("LLM_API_URL") != "" {
			return "gpt-4.1-nano-2025-04-14"
		}
		// Default to Gemini Flash Lite if using Gemini keys
		if os.Getenv("GEMINI_API_KEY") != "" {
			return "gemini-2.5-flash-lite"
		}
		// Final fallback
		return "gpt-4.1-nano-2025-04-14"
	}
}

func (p *OpenAIProvider) Generate(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
	// Determine model - use config model if provided, otherwise use provider-specific default
	model := config.Model
	if model == "" {
		model = p.defaultModel
	} else if os.Getenv("THROUGHPUT_MANAGER") == "true" {
		// Override with model from throughput manager - use wait version to handle rate limits
		model = p.throughputManager.GetModelForRequestWithWait()
		if model == "" {
			// fallback to default if throughput manager has no models configured
			model = p.defaultModel
			logger.Printf("[LLM] Throughput manager returned empty, falling back to default: %s", model)
		} else {
			logger.Printf("[LLM] Overriding model with throughput manager selection: %s", model)
		}
	}

	// Log the request details in development mode
	logger.Printf("[LLM] Submitting job with model: %s, prompt length: %d chars", model, len(prompt))

	// Pass context through to job submitter for cancellation support
	ctx := config.Context
	if ctx == nil {
		ctx = context.Background()
		logger.Printf("[LLM] No context provided, using background context")
	} else {
		logger.Printf("[LLM] Using provided context for cancellation support")
	}

	// Submit job with Definition (includes SendImage if present)
	completion, usage, err := p.submitter.SubmitJob(ctx, model, config.Definition, prompt, config.SystemPrompt, nil)

	if err != nil {
		if strings.Contains(err.Error(), "429") && os.Getenv("THROUGHPUT_MANAGER") == "true" {
			p.throughputManager.ReportRateLimitError(model)
		}

		// retry with exponential backoff
		for i := 0; i < p.maxRetries; i++ {
			// exponential backoff: 500ms, 1s, 2s, 4s...
			backoffDuration := time.Duration(500*(1<<i)) * time.Millisecond
			logger.Printf("[LLM] Retrying job submission (%d/%d) after %v...", i+1, p.maxRetries, backoffDuration)
			time.Sleep(backoffDuration)

			// on rate limit, try to get a different model
			if strings.Contains(err.Error(), "429") && os.Getenv("THROUGHPUT_MANAGER") == "true" {
				newModel := p.throughputManager.GetModelForRequestWithWait()
				if newModel != "" && newModel != model {
					logger.Printf("[LLM] Switching to different model for retry: %s -> %s", model, newModel)
					model = newModel
				}
			}

			completion, usage, err = p.submitter.SubmitJob(ctx, model, config.Definition, prompt, config.SystemPrompt, nil)
			if err == nil {
				break
			}
			if strings.Contains(err.Error(), "429") && os.Getenv("THROUGHPUT_MANAGER") == "true" {
				p.throughputManager.ReportRateLimitError(model)
			}
		}

		if err != nil {
			logger.Printf("[LLM ERROR] Job submission failed after %d retries: %v", p.maxRetries, err)
			if strings.Contains(err.Error(), "reasoning_effort") {
				logger.Printf("[LLM ERROR] The request includes a reasoning_effort parameter but model %q does not support it. Remove reasoning_effort from ModelConfig or use a reasoning model (e.g. o1, o3, o4 series).", model)
			}
			return "", nil, fmt.Errorf("job submission failed: %w", err)
		}
	}

	metadata := &domain.ProviderMetadata{
		Model: string(model),
	}

	if usage != nil {
		metadata.TokensUsed = usage.TotalTokens
		metadata.PromptTokens = usage.PromptTokens
		metadata.CompletionTokens = usage.CompletionTokens
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
	logger.Printf("[OpenAIProvider] Generating audio with TTS...")
	logger.Printf("[OpenAIProvider] Input: %s", request.Input)
	logger.Printf("[OpenAIProvider] Voice: %s", request.Voice)
	logger.Printf("[OpenAIProvider] Model: %s", request.Model)

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
	response, err := p.byteOps.CreateSpeech(ctx, ttsReq)
	if err != nil {
		logger.Printf("[OpenAIProvider ERROR] TTS generation failed: %v", err)
		return nil, nil, fmt.Errorf("TTS generation failed: %w", err)
	}
	defer response.Close()

	// Read the audio data
	audioBytes, err := io.ReadAll(response)
	if err != nil {
		logger.Printf("[OpenAIProvider ERROR] Failed to read TTS audio: %v", err)
		return nil, nil, fmt.Errorf("failed to read TTS audio: %w", err)
	}

	logger.Printf("[OpenAIProvider] TTS generated successfully: %d bytes", len(audioBytes))

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
	logger.Printf("[OpenAIProvider] Generating image with DALL-E...")
	logger.Printf("[OpenAIProvider] Prompt: %s", request.Prompt)
	logger.Printf("[OpenAIProvider] Model: %s", request.Model)
	logger.Printf("[OpenAIProvider] Size: %s", request.Size)

	// Build image request
	req := gogpt.ImageRequest{
		Prompt:         request.Prompt,
		Size:           request.Size,
		ResponseFormat: gogpt.CreateImageResponseFormatB64JSON,
		N:              1,
	}

	modelStr := string(request.Model)
	switch modelStr {
	case "dall-e-3":
		req.Model = gogpt.CreateImageModelDallE3
	default:
		req.Model = gogpt.CreateImageModelDallE2
	}

	resp, err := p.byteOps.CreateImage(context.Background(), req)
	if err != nil {
		logger.Printf("[OpenAIProvider ERROR] Image generation failed: %v", err)
		return nil, nil, fmt.Errorf("image generation failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, nil, fmt.Errorf("no image data returned from API")
	}

	imageData := resp.Data[0]
	var imageBytes []byte

	if imageData.B64JSON != "" {
		imageBytes, err = base64.StdEncoding.DecodeString(imageData.B64JSON)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode base64 image: %w", err)
		}
	} else if imageData.URL != "" {
		httpResp, err := http.Get(imageData.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to download image from URL: %w", err)
		}
		defer httpResp.Body.Close()
		imageBytes, err = io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read image data: %w", err)
		}
	} else {
		return nil, nil, fmt.Errorf("no image data in response")
	}

	logger.Printf("[OpenAIProvider] Image generated successfully: %d bytes", len(imageBytes))

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
	logger.Printf("[OpenAIProvider] Transcribing audio with Whisper...")
	logger.Printf("[OpenAIProvider] Model: %s", request.Model)
	logger.Printf("[OpenAIProvider] Language: %s", request.Language)
	logger.Printf("[OpenAIProvider] Audio data size: %d bytes", len(request.AudioData))

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
	response, err := p.byteOps.CreateTranscription(ctx, whisperReq)
	if err != nil {
		logger.Printf("[OpenAIProvider ERROR] Audio transcription failed: %v", err)
		return "", nil, fmt.Errorf("audio transcription failed: %w", err)
	}

	logger.Printf("[OpenAIProvider] Audio transcribed successfully: %s", response.Text)

	// Create metadata
	metadata := &domain.ProviderMetadata{
		Model: request.Model,
		// TODO: Add cost calculation for STT
		Cost:       0,
		TokensUsed: 0,
	}

	if request.ResponseFormat == "verbose_json" || request.ResponseFormat == "diarized_json" {
		metadata.VerboseData = map[string]any{
			"language": response.Language,
			"duration": response.Duration,
			"segments": response.Segments,
			"words":    response.Words,
		}
	}

	return response.Text, metadata, nil
}

// SupportsByteOperations returns true since OpenAI supports DALL-E image generation
// (and will support TTS/STT in the future)
func (p *OpenAIProvider) SupportsByteOperations() bool {
	return true
}

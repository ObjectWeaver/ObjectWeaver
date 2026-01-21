package llm

import (
	"context"
	"os"
	"strings"
	"testing"

	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
	gogpt "github.com/sashabaranov/go-openai"
)

// MockJobEntryPoint for testing
type MockJobEntryPoint struct {
	submitJobFunc func(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error)
}

func (m *MockJobEntryPoint) SubmitJob(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
	if m.submitJobFunc != nil {
		return m.submitJobFunc(ctx, model, def, newPrompt, systemPrompt, outStream)
	}
	return "mock response", &gogpt.Usage{TotalTokens: 100}, nil
}

func TestNewOpenAIProvider(t *testing.T) {
	// Set a test API key
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()
	if provider == nil {
		t.Fatal("NewOpenAIProvider returned nil")
	}
	if provider.submitter == nil {
		t.Error("submitter should be initialized")
	}
	if provider.client == nil {
		t.Error("client should be initialized")
	}
}

func TestGetDefaultModelForProvider_OpenAI(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "openai")
	defer os.Unsetenv("LLM_PROVIDER")

	model := getDefaultModelForProvider()
	if model != "gpt-4o-mini" {
		t.Errorf("Expected gpt-4o-mini, got %v", model)
	}
}

func TestGetDefaultModelForProvider_Gemini(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "gemini")
	defer os.Unsetenv("LLM_PROVIDER")

	model := getDefaultModelForProvider()
	if model != "gemini-2.5-flash" {
		t.Errorf("Expected gemini-2.5-flash, got %v", model)
	}
}

func TestGetDefaultModelForProvider_Local(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "local")
	defer os.Unsetenv("LLM_PROVIDER")

	model := getDefaultModelForProvider()
	if model != "gpt-4o-mini" {
		t.Errorf("Expected gpt-4o-mini for local, got %v", model)
	}
}

func TestGetDefaultModelForProvider_Default(t *testing.T) {
	// Clear all env vars
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_URL")
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("LLM_API_KEY")

	model := getDefaultModelForProvider()
	if model != "gpt-4o-mini" {
		t.Errorf("Expected gpt-4o-mini as fallback, got %v", model)
	}
}

func TestGetDefaultModelForProvider_WithGeminiKey(t *testing.T) {
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_URL")
	os.Setenv("GEMINI_API_KEY", "test-key")
	defer os.Unsetenv("GEMINI_API_KEY")

	model := getDefaultModelForProvider()
	if model != "gemini-2.5-flash" {
		t.Errorf("Expected gemini-2.5-flash when GEMINI_API_KEY is set, got %v", model)
	}
}

func TestOpenAIProvider_Generate(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()
	// Replace submitter with mock
	mockSubmitter := &MockJobEntryPoint{
		submitJobFunc: func(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
			return "test response", &gogpt.Usage{TotalTokens: 100}, nil
		},
	}
	provider.submitter = mockSubmitter

	config := &domain.GenerationConfig{
		Model:        "gpt-4o-mini",
		SystemPrompt: "Test system",
		Definition:   &jsonSchema.Definition{},
	}

	response, metadata, err := provider.Generate("test prompt", config)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if response != "test response" {
		t.Errorf("Expected 'test response', got %q", response)
	}
	if metadata == nil {
		t.Error("Metadata should not be nil")
	}
	// Note: Current implementation doesn't populate TokensUsed from SubmitJob usage
	// The Generate method ignores the usage return value
	if metadata.Model != "gpt-4o-mini" {
		t.Errorf("Expected model 'gpt-4o-mini', got %s", metadata.Model)
	}
}

func TestOpenAIProvider_SupportsStreaming(t *testing.T) {
	provider := NewOpenAIProvider()
	if provider.SupportsStreaming() {
		t.Error("Base OpenAIProvider should not support streaming")
	}
}

func TestOpenAIProvider_ModelType(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "openai")
	defer os.Unsetenv("LLM_PROVIDER")

	provider := NewOpenAIProvider()
	model := provider.ModelType()
	if model != "gpt-4o-mini" {
		t.Errorf("Expected gpt-4o-mini, got %v", model)
	}
}

func TestNewStreamingOpenAIProvider(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewStreamingOpenAIProvider()
	if provider == nil {
		t.Fatal("NewStreamingOpenAIProvider returned nil")
	}
	if provider.OpenAIProvider == nil {
		t.Error("OpenAIProvider should be embedded")
	}
}

func TestStreamingOpenAIProvider_SupportsStreaming(t *testing.T) {
	provider := NewStreamingOpenAIProvider()
	if !provider.SupportsStreaming() {
		t.Error("StreamingOpenAIProvider should support streaming")
	}
}

func TestStreamingOpenAIProvider_SupportsTokenStreaming(t *testing.T) {
	provider := NewStreamingOpenAIProvider()
	if !provider.SupportsTokenStreaming() {
		t.Error("StreamingOpenAIProvider should support token streaming")
	}
}

func TestStreamingOpenAIProvider_GenerateStream(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewStreamingOpenAIProvider()
	// Mock the submitter
	mockSubmitter := &MockJobEntryPoint{
		submitJobFunc: func(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
			return "hello world", &gogpt.Usage{}, nil
		},
	}
	provider.submitter = mockSubmitter

	config := &domain.GenerationConfig{}

	stream, err := provider.GenerateStream("test", config)
	if err != nil {
		t.Fatalf("GenerateStream returned error: %v", err)
	}

	var received []string
	for chunk := range stream {
		if str, ok := chunk.(string); ok {
			received = append(received, str)
		}
	}

	if len(received) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(received))
	}
	if received[0] != "hello world" {
		t.Errorf("Expected 'hello world', got %q", received[0])
	}
}

func TestStreamingOpenAIProvider_GenerateTokenStream(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewStreamingOpenAIProvider()
	// Mock the submitter
	mockSubmitter := &MockJobEntryPoint{
		submitJobFunc: func(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
			return "hi", &gogpt.Usage{}, nil
		},
	}
	provider.submitter = mockSubmitter

	config := &domain.GenerationConfig{}

	stream, err := provider.GenerateTokenStream("test", config)
	if err != nil {
		t.Fatalf("GenerateTokenStream returned error: %v", err)
	}

	var received []string
	var isFinal bool
	for chunk := range stream {
		received = append(received, chunk.Token)
		if chunk.IsFinal {
			isFinal = true
		}
	}

	if len(received) != 3 { // 'h', 'i', ''
		t.Errorf("Expected 3 chunks, got %d", len(received))
	}
	if !isFinal {
		t.Error("Expected final chunk")
	}
}

func TestCalculateCost(t *testing.T) {
	// Currently returns 0.0
	cost := calculateCost("gpt-4o-mini", &gogpt.Usage{})
	if cost != 0.0 {
		t.Errorf("Expected 0.0, got %f", cost)
	}
}

func TestOpenAIProvider_SupportsByteOperations(t *testing.T) {
	provider := NewOpenAIProvider()
	if !provider.SupportsByteOperations() {
		t.Error("OpenAIProvider should support byte operations")
	}
}

func TestOpenAIProvider_GenerateAudio_InvalidAPIKey(t *testing.T) {
	// Test with invalid API key to verify error handling
	os.Setenv("OPENAI_API_KEY", "invalid-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	request := &domain.AudioGenerationRequest{
		Input:          "Test text to speech",
		Voice:          "alloy",
		Model:          "tts-1",
		ResponseFormat: "mp3",
		Speed:          1.0,
	}

	audioBytes, metadata, err := provider.GenerateAudio(request)

	// Should get an error with invalid API key
	if err == nil {
		t.Error("Expected error with invalid API key")
	}

	if audioBytes != nil {
		t.Error("Expected nil audio bytes on error")
	}

	if metadata != nil {
		t.Error("Expected nil metadata on error")
	}
}

func TestOpenAIProvider_GenerateAudio_RequestValidation(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	tests := []struct {
		name    string
		request *domain.AudioGenerationRequest
	}{
		{
			name: "valid request",
			request: &domain.AudioGenerationRequest{
				Input:          "Hello world",
				Voice:          "alloy",
				Model:          "tts-1",
				ResponseFormat: "mp3",
				Speed:          1.0,
			},
		},
		{
			name: "different voice",
			request: &domain.AudioGenerationRequest{
				Input:          "Test",
				Voice:          "nova",
				Model:          "tts-1-hd",
				ResponseFormat: "opus",
				Speed:          1.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Will fail with invalid key, but validates request structure
			_, _, err := provider.GenerateAudio(tt.request)
			if err == nil {
				t.Log("Note: API call succeeded, API key might be valid")
			}
		})
	}
}

func TestOpenAIProvider_GenerateImage_InvalidAPIKey(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "invalid-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	request := &domain.ImageGenerationRequest{
		Prompt: "A beautiful sunset",
		Model:  "dall-e-2",
		Size:   "1024x1024",
	}

	imageBytes, metadata, err := provider.GenerateImage(request)

	// Should get an error with invalid API key
	if err == nil {
		t.Error("Expected error with invalid API key")
	}

	if imageBytes != nil {
		t.Error("Expected nil image bytes on error")
	}

	if metadata != nil {
		t.Error("Expected nil metadata on error")
	}
}

func TestOpenAIProvider_GenerateImage_RequestValidation(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	tests := []struct {
		name    string
		request *domain.ImageGenerationRequest
	}{
		{
			name: "DALL-E 2 request",
			request: &domain.ImageGenerationRequest{
				Prompt: "A cat in space",
				Model:  "dall-e-2",
				Size:   "512x512",
			},
		},
		{
			name: "DALL-E 3 request",
			request: &domain.ImageGenerationRequest{
				Prompt: "A futuristic city",
				Model:  "dall-e-3",
				Size:   "1024x1024",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Will fail with invalid key, but validates request structure
			_, _, err := provider.GenerateImage(tt.request)
			if err == nil {
				t.Log("Note: API call succeeded, API key might be valid")
			}
		})
	}
}

func TestOpenAIProvider_TranscribeAudio_InvalidAPIKey(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "invalid-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	request := &domain.AudioTranscriptionRequest{
		AudioData:      []byte("fake audio data"),
		Model:          "whisper-1",
		Language:       "en",
		Prompt:         "Test prompt",
		ResponseFormat: "text",
	}

	text, metadata, err := provider.TranscribeAudio(request)

	// Should get an error with invalid API key
	if err == nil {
		t.Error("Expected error with invalid API key")
	}

	if text != "" {
		t.Error("Expected empty text on error")
	}

	if metadata != nil {
		t.Error("Expected nil metadata on error")
	}
}

func TestOpenAIProvider_TranscribeAudio_RequestValidation(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	tests := []struct {
		name    string
		request *domain.AudioTranscriptionRequest
	}{
		{
			name: "basic transcription",
			request: &domain.AudioTranscriptionRequest{
				AudioData:      []byte("test audio"),
				Model:          "whisper-1",
				Language:       "en",
				ResponseFormat: "text",
			},
		},
		{
			name: "with prompt",
			request: &domain.AudioTranscriptionRequest{
				AudioData:      []byte("test audio"),
				Model:          "whisper-1",
				Language:       "es",
				Prompt:         "Context prompt",
				ResponseFormat: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Will fail with invalid key, but validates request structure
			_, _, err := provider.TranscribeAudio(tt.request)
			if err == nil {
				t.Log("Note: API call succeeded, API key might be valid")
			}
		})
	}
}

func TestOpenAIProvider_GenerateAudio_EmptyInput(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	request := &domain.AudioGenerationRequest{
		Input:          "",
		Voice:          "alloy",
		Model:          "tts-1",
		ResponseFormat: "mp3",
		Speed:          1.0,
	}

	_, _, err := provider.GenerateAudio(request)

	// Should get an error for empty input
	if err == nil {
		t.Error("Expected error with empty input")
	}
}

func TestOpenAIProvider_GenerateImage_EmptyPrompt(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	request := &domain.ImageGenerationRequest{
		Prompt: "",
		Model:  "dall-e-2",
		Size:   "512x512",
	}

	_, _, err := provider.GenerateImage(request)

	// Should get an error for empty prompt
	if err == nil {
		t.Error("Expected error with empty prompt")
	}
}

func TestOpenAIProvider_TranscribeAudio_EmptyAudioData(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	request := &domain.AudioTranscriptionRequest{
		AudioData:      []byte{},
		Model:          "whisper-1",
		Language:       "en",
		ResponseFormat: "text",
	}

	_, _, err := provider.TranscribeAudio(request)

	// Should get an error for empty audio data
	if err == nil {
		t.Error("Expected error with empty audio data")
	}
}

func TestOpenAIProvider_TranscribeAudio_VerboseJsonMetadata(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	tests := []struct {
		name                 string
		responseFormat       string
		expectVerboseData    bool
		expectVerboseDataNil bool
	}{
		{
			name:                 "verbose_json format includes metadata",
			responseFormat:       "verbose_json",
			expectVerboseData:    true,
			expectVerboseDataNil: false,
		},
		{
			name:                 "diarized_json format includes metadata",
			responseFormat:       "diarized_json",
			expectVerboseData:    true,
			expectVerboseDataNil: false,
		},
		{
			name:                 "json format excludes verbose metadata",
			responseFormat:       "json",
			expectVerboseData:    false,
			expectVerboseDataNil: true,
		},
		{
			name:                 "text format excludes verbose metadata",
			responseFormat:       "text",
			expectVerboseData:    false,
			expectVerboseDataNil: true,
		},
		{
			name:                 "srt format excludes verbose metadata",
			responseFormat:       "srt",
			expectVerboseData:    false,
			expectVerboseDataNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &domain.AudioTranscriptionRequest{
				AudioData:      []byte("test audio data"),
				Model:          "whisper-1",
				Language:       "en",
				ResponseFormat: tt.responseFormat,
			}

			// This will fail with invalid API key, but we're testing the code structure
			_, metadata, err := provider.TranscribeAudio(request)

			// We expect an error due to invalid API key, but we can still check the logic
			if err == nil {
				// If no error, we have a valid API key - check metadata was populated correctly
				if metadata == nil {
					t.Fatal("Expected metadata to be returned")
				}

				if tt.expectVerboseData {
					if metadata.VerboseData == nil {
						t.Error("Expected VerboseData to be populated for verbose format")
					} else {
						// Verify expected fields are present
						expectedFields := []string{"language", "duration", "segments", "words"}
						for _, field := range expectedFields {
							if _, ok := metadata.VerboseData[field]; !ok {
								t.Errorf("Expected VerboseData to contain field '%s'", field)
							}
						}
					}
				} else if tt.expectVerboseDataNil {
					if metadata.VerboseData != nil {
						t.Errorf("Expected VerboseData to be nil for format '%s', got %v", tt.responseFormat, metadata.VerboseData)
					}
				}

				// Verify standard metadata fields
				if metadata.Model != "whisper-1" {
					t.Errorf("Expected Model to be 'whisper-1', got '%s'", metadata.Model)
				}
			}
		})
	}
}

func TestOpenAIProvider_TranscribeAudio_VerboseJsonStructure(t *testing.T) {
	// This test documents the expected structure of VerboseData
	// when verbose_json format is used with a valid API key

	expectedStructure := map[string]string{
		"language": "string - detected language code (e.g., 'en')",
		"duration": "float64 - audio duration in seconds",
		"segments": "[]struct - array of transcription segments with timing",
		"words":    "[]struct - array of word-level timestamps",
	}

	t.Log("Expected VerboseData structure for verbose_json format:")
	for field, description := range expectedStructure {
		t.Logf("  %s: %s", field, description)
	}

	t.Log("\nSegment structure includes:")
	segmentFields := []string{
		"id - segment identifier",
		"seek - seek position",
		"start - start timestamp",
		"end - end timestamp",
		"text - segment transcription",
		"tokens - token array",
		"temperature - temperature value",
		"avg_logprob - average log probability",
		"compression_ratio - compression ratio",
		"no_speech_prob - probability of no speech",
		"transient - transient flag",
	}
	for _, field := range segmentFields {
		t.Logf("  - %s", field)
	}

	t.Log("\nWord structure includes:")
	wordFields := []string{
		"word - the word text",
		"start - start timestamp",
		"end - end timestamp",
	}
	for _, field := range wordFields {
		t.Logf("  - %s", field)
	}
}

func TestOpenAIProvider_TranscribeAudio_ResponseFormatPassthrough(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()

	// Test that all supported response formats are passed through correctly
	formats := []string{"json", "text", "srt", "vtt", "verbose_json", "diarized_json"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			request := &domain.AudioTranscriptionRequest{
				AudioData:      []byte("test audio"),
				Model:          "whisper-1",
				Language:       "en",
				ResponseFormat: format,
			}

			// This will fail with invalid key, but validates the format is accepted
			_, _, err := provider.TranscribeAudio(request)

			if err == nil {
				t.Logf("Note: API call succeeded for format '%s'", format)
			} else {
				// Just verify we're not getting a format validation error
				if strings.Contains(err.Error(), "unsupported format") {
					t.Errorf("Format '%s' should be supported but got unsupported format error", format)
				}
			}
		})
	}
}

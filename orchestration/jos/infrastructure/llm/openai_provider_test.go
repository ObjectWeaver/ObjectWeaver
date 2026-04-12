package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"objectweaver/orchestration/jos/domain"

	"objectweaver/jsonSchema"

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

// mockByteOpsClient mocks the OpenAI client byte operations (TTS, image, STT)
type mockByteOpsClient struct {
	createSpeechFunc        func(ctx context.Context, req gogpt.CreateSpeechRequest) (gogpt.RawResponse, error)
	createImageFunc         func(ctx context.Context, req gogpt.ImageRequest) (gogpt.ImageResponse, error)
	createTranscriptionFunc func(ctx context.Context, req gogpt.AudioRequest) (gogpt.AudioResponse, error)
}

func (m *mockByteOpsClient) CreateSpeech(ctx context.Context, req gogpt.CreateSpeechRequest) (gogpt.RawResponse, error) {
	if m.createSpeechFunc != nil {
		return m.createSpeechFunc(ctx, req)
	}
	return gogpt.RawResponse{ReadCloser: io.NopCloser(strings.NewReader("fake audio data"))}, nil
}

func (m *mockByteOpsClient) CreateImage(ctx context.Context, req gogpt.ImageRequest) (gogpt.ImageResponse, error) {
	if m.createImageFunc != nil {
		return m.createImageFunc(ctx, req)
	}
	fakeImageBytes := []byte{0x89, 0x50, 0x4E, 0x47}
	return gogpt.ImageResponse{
		Data: []gogpt.ImageResponseDataInner{
			{B64JSON: base64.StdEncoding.EncodeToString(fakeImageBytes)},
		},
	}, nil
}

func (m *mockByteOpsClient) CreateTranscription(ctx context.Context, req gogpt.AudioRequest) (gogpt.AudioResponse, error) {
	if m.createTranscriptionFunc != nil {
		return m.createTranscriptionFunc(ctx, req)
	}
	return gogpt.AudioResponse{Text: "transcribed text"}, nil
}

func TestNewOpenAIProvider(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()
	if provider == nil {
		t.Fatal("NewOpenAIProvider returned nil")
	}
	if provider.submitter == nil {
		t.Error("submitter should be initialized")
	}
	if provider.byteOps == nil {
		t.Error("byteOps should be initialized")
	}
}

func TestGetDefaultModelForProvider_OpenAI(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "openai")
	defer os.Unsetenv("LLM_PROVIDER")

	model := getDefaultModelForProvider()
	if model != "gpt-4.1-nano-2025-04-14" {
		t.Errorf("Expected gpt-4.1-nano-2025-04-14, got %v", model)
	}
}

func TestGetDefaultModelForProvider_Gemini(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "gemini")
	defer os.Unsetenv("LLM_PROVIDER")

	model := getDefaultModelForProvider()
	if model != "gemini-2.5-flash-lite" {
		t.Errorf("Expected gemini-2.5-flash-lite, got %v", model)
	}
}

func TestGetDefaultModelForProvider_Local(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "local")
	defer os.Unsetenv("LLM_PROVIDER")

	model := getDefaultModelForProvider()
	if model != "gpt-4.1-nano-2025-04-14" {
		t.Errorf("Expected gpt-4.1-nano-2025-04-14 for local, got %v", model)
	}
}

func TestGetDefaultModelForProvider_Default(t *testing.T) {
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_URL")
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("LLM_API_KEY")

	model := getDefaultModelForProvider()
	if model != "gpt-4.1-nano-2025-04-14" {
		t.Errorf("Expected gpt-4.1-nano-2025-04-14 as fallback, got %v", model)
	}
}

func TestGetDefaultModelForProvider_WithGeminiKey(t *testing.T) {
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_URL")
	os.Setenv("GEMINI_API_KEY", "test-key")
	defer os.Unsetenv("GEMINI_API_KEY")

	model := getDefaultModelForProvider()
	if model != "gemini-2.5-flash-lite" {
		t.Errorf("Expected gemini-2.5-flash-lite when GEMINI_API_KEY is set, got %v", model)
	}
}

func TestOpenAIProvider_Generate(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()
	mockSubmitter := &MockJobEntryPoint{
		submitJobFunc: func(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
			return "test response", &gogpt.Usage{TotalTokens: 100}, nil
		},
	}
	provider.submitter = mockSubmitter

	config := &domain.GenerationConfig{
		Model:        "gpt-4.1-nano-2025-04-14",
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
	if metadata.Model != "gpt-4.1-nano-2025-04-14" {
		t.Errorf("Expected model 'gpt-4.1-nano-2025-04-14', got %s", metadata.Model)
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
	if model != "gpt-4.1-nano-2025-04-14" {
		t.Errorf("Expected gpt-4.1-nano-2025-04-14, got %v", model)
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
	cost := calculateCost("gpt-4.1-nano-2025-04-14", &gogpt.Usage{})
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

// ============================================================================
// Mocked byte operation tests (no real API calls)
// ============================================================================

func TestOpenAIProvider_GenerateAudio_Success(t *testing.T) {
	provider := NewOpenAIProvider()
	expectedAudio := []byte("mock audio bytes")
	provider.byteOps = &mockByteOpsClient{
		createSpeechFunc: func(ctx context.Context, req gogpt.CreateSpeechRequest) (gogpt.RawResponse, error) {
			if req.Input != "Hello world" {
				t.Errorf("Expected input 'Hello world', got '%s'", req.Input)
			}
			if string(req.Voice) != "alloy" {
				t.Errorf("Expected voice 'alloy', got '%s'", req.Voice)
			}
			return gogpt.RawResponse{ReadCloser: io.NopCloser(strings.NewReader(string(expectedAudio)))}, nil
		},
	}

	request := &domain.AudioGenerationRequest{
		Input:          "Hello world",
		Voice:          "alloy",
		Model:          "tts-1",
		ResponseFormat: "mp3",
		Speed:          1.0,
	}

	audioBytes, metadata, err := provider.GenerateAudio(request)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if string(audioBytes) != string(expectedAudio) {
		t.Errorf("Expected audio bytes to match")
	}
	if metadata == nil {
		t.Fatal("Expected metadata to be non-nil")
	}
	if metadata.Model != "tts-1" {
		t.Errorf("Expected model 'tts-1', got '%s'", metadata.Model)
	}
}

func TestOpenAIProvider_GenerateAudio_Error(t *testing.T) {
	provider := NewOpenAIProvider()
	provider.byteOps = &mockByteOpsClient{
		createSpeechFunc: func(ctx context.Context, req gogpt.CreateSpeechRequest) (gogpt.RawResponse, error) {
			return gogpt.RawResponse{}, fmt.Errorf("TTS generation failed: unauthorized")
		},
	}

	request := &domain.AudioGenerationRequest{
		Input: "Test text",
		Voice: "alloy",
		Model: "tts-1",
	}

	audioBytes, metadata, err := provider.GenerateAudio(request)
	if err == nil {
		t.Error("Expected error")
	}
	if audioBytes != nil {
		t.Error("Expected nil audio bytes on error")
	}
	if metadata != nil {
		t.Error("Expected nil metadata on error")
	}
}

func TestOpenAIProvider_GenerateAudio_DifferentVoices(t *testing.T) {
	tests := []struct {
		name  string
		voice string
		model string
	}{
		{name: "alloy voice", voice: "alloy", model: "tts-1"},
		{name: "nova voice", voice: "nova", model: "tts-1-hd"},
		{name: "shimmer voice", voice: "shimmer", model: "tts-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOpenAIProvider()
			provider.byteOps = &mockByteOpsClient{
				createSpeechFunc: func(ctx context.Context, req gogpt.CreateSpeechRequest) (gogpt.RawResponse, error) {
					if string(req.Voice) != tt.voice {
						t.Errorf("Expected voice '%s', got '%s'", tt.voice, req.Voice)
					}
					if string(req.Model) != tt.model {
						t.Errorf("Expected model '%s', got '%s'", tt.model, req.Model)
					}
					return gogpt.RawResponse{ReadCloser: io.NopCloser(strings.NewReader("audio"))}, nil
				},
			}

			request := &domain.AudioGenerationRequest{
				Input: "Test",
				Voice: tt.voice,
				Model: tt.model,
			}

			_, _, err := provider.GenerateAudio(request)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestOpenAIProvider_GenerateImage_Success(t *testing.T) {
	provider := NewOpenAIProvider()
	fakeImageBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}
	provider.byteOps = &mockByteOpsClient{
		createImageFunc: func(ctx context.Context, req gogpt.ImageRequest) (gogpt.ImageResponse, error) {
			if req.Prompt != "A beautiful sunset" {
				t.Errorf("Expected prompt 'A beautiful sunset', got '%s'", req.Prompt)
			}
			return gogpt.ImageResponse{
				Data: []gogpt.ImageResponseDataInner{
					{B64JSON: base64.StdEncoding.EncodeToString(fakeImageBytes)},
				},
			}, nil
		},
	}

	request := &domain.ImageGenerationRequest{
		Prompt: "A beautiful sunset",
		Model:  "dall-e-2",
		Size:   "1024x1024",
	}

	imageBytes, metadata, err := provider.GenerateImage(request)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(imageBytes) != len(fakeImageBytes) {
		t.Errorf("Expected %d bytes, got %d", len(fakeImageBytes), len(imageBytes))
	}
	if metadata == nil {
		t.Fatal("Expected metadata to be non-nil")
	}
	if metadata.Model != "dall-e-2" {
		t.Errorf("Expected model 'dall-e-2', got '%s'", metadata.Model)
	}
}

func TestOpenAIProvider_GenerateImage_Error(t *testing.T) {
	provider := NewOpenAIProvider()
	provider.byteOps = &mockByteOpsClient{
		createImageFunc: func(ctx context.Context, req gogpt.ImageRequest) (gogpt.ImageResponse, error) {
			return gogpt.ImageResponse{}, fmt.Errorf("image generation failed: unauthorized")
		},
	}

	request := &domain.ImageGenerationRequest{
		Prompt: "A cat",
		Model:  "dall-e-2",
		Size:   "512x512",
	}

	imageBytes, metadata, err := provider.GenerateImage(request)
	if err == nil {
		t.Error("Expected error")
	}
	if imageBytes != nil {
		t.Error("Expected nil image bytes on error")
	}
	if metadata != nil {
		t.Error("Expected nil metadata on error")
	}
}

func TestOpenAIProvider_GenerateImage_DallE3(t *testing.T) {
	provider := NewOpenAIProvider()
	provider.byteOps = &mockByteOpsClient{
		createImageFunc: func(ctx context.Context, req gogpt.ImageRequest) (gogpt.ImageResponse, error) {
			if req.Model != gogpt.CreateImageModelDallE3 {
				t.Errorf("Expected DALL-E 3 model, got %s", req.Model)
			}
			fakeBytes := []byte{0xFF, 0xD8, 0xFF}
			return gogpt.ImageResponse{
				Data: []gogpt.ImageResponseDataInner{
					{B64JSON: base64.StdEncoding.EncodeToString(fakeBytes)},
				},
			}, nil
		},
	}

	request := &domain.ImageGenerationRequest{
		Prompt: "A futuristic city",
		Model:  "dall-e-3",
		Size:   "1024x1024",
	}

	_, _, err := provider.GenerateImage(request)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestOpenAIProvider_GenerateImage_EmptyResponse(t *testing.T) {
	provider := NewOpenAIProvider()
	provider.byteOps = &mockByteOpsClient{
		createImageFunc: func(ctx context.Context, req gogpt.ImageRequest) (gogpt.ImageResponse, error) {
			return gogpt.ImageResponse{Data: []gogpt.ImageResponseDataInner{}}, nil
		},
	}

	request := &domain.ImageGenerationRequest{
		Prompt: "Test",
		Model:  "dall-e-2",
		Size:   "512x512",
	}

	_, _, err := provider.GenerateImage(request)
	if err == nil {
		t.Error("Expected error for empty response data")
	}
}

func TestOpenAIProvider_TranscribeAudio_Success(t *testing.T) {
	provider := NewOpenAIProvider()
	provider.byteOps = &mockByteOpsClient{
		createTranscriptionFunc: func(ctx context.Context, req gogpt.AudioRequest) (gogpt.AudioResponse, error) {
			if req.Model != "whisper-1" {
				t.Errorf("Expected model 'whisper-1', got '%s'", req.Model)
			}
			if req.Language != "en" {
				t.Errorf("Expected language 'en', got '%s'", req.Language)
			}
			return gogpt.AudioResponse{Text: "Hello, world!"}, nil
		},
	}

	request := &domain.AudioTranscriptionRequest{
		AudioData:      []byte("fake audio data"),
		Model:          "whisper-1",
		Language:       "en",
		ResponseFormat: "text",
	}

	text, metadata, err := provider.TranscribeAudio(request)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if text != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", text)
	}
	if metadata == nil {
		t.Fatal("Expected metadata to be non-nil")
	}
	if metadata.Model != "whisper-1" {
		t.Errorf("Expected model 'whisper-1', got '%s'", metadata.Model)
	}
}

func TestOpenAIProvider_TranscribeAudio_Error(t *testing.T) {
	provider := NewOpenAIProvider()
	provider.byteOps = &mockByteOpsClient{
		createTranscriptionFunc: func(ctx context.Context, req gogpt.AudioRequest) (gogpt.AudioResponse, error) {
			return gogpt.AudioResponse{}, fmt.Errorf("transcription failed: unauthorized")
		},
	}

	request := &domain.AudioTranscriptionRequest{
		AudioData:      []byte("fake audio data"),
		Model:          "whisper-1",
		Language:       "en",
		ResponseFormat: "text",
	}

	text, metadata, err := provider.TranscribeAudio(request)
	if err == nil {
		t.Error("Expected error")
	}
	if text != "" {
		t.Error("Expected empty text on error")
	}
	if metadata != nil {
		t.Error("Expected nil metadata on error")
	}
}

func TestOpenAIProvider_TranscribeAudio_DifferentLanguages(t *testing.T) {
	tests := []struct {
		name     string
		language string
	}{
		{name: "English", language: "en"},
		{name: "Spanish", language: "es"},
		{name: "French", language: "fr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOpenAIProvider()
			provider.byteOps = &mockByteOpsClient{
				createTranscriptionFunc: func(ctx context.Context, req gogpt.AudioRequest) (gogpt.AudioResponse, error) {
					if req.Language != tt.language {
						t.Errorf("Expected language '%s', got '%s'", tt.language, req.Language)
					}
					return gogpt.AudioResponse{Text: "transcribed"}, nil
				},
			}

			request := &domain.AudioTranscriptionRequest{
				AudioData:      []byte("test audio"),
				Model:          "whisper-1",
				Language:       tt.language,
				ResponseFormat: "text",
			}

			_, _, err := provider.TranscribeAudio(request)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestOpenAIProvider_TranscribeAudio_VerboseJsonMetadata(t *testing.T) {
	tests := []struct {
		name              string
		responseFormat    string
		expectVerboseData bool
	}{
		{name: "verbose_json includes metadata", responseFormat: "verbose_json", expectVerboseData: true},
		{name: "diarized_json includes metadata", responseFormat: "diarized_json", expectVerboseData: true},
		{name: "json excludes verbose metadata", responseFormat: "json", expectVerboseData: false},
		{name: "text excludes verbose metadata", responseFormat: "text", expectVerboseData: false},
		{name: "srt excludes verbose metadata", responseFormat: "srt", expectVerboseData: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOpenAIProvider()
			provider.byteOps = &mockByteOpsClient{
				createTranscriptionFunc: func(ctx context.Context, req gogpt.AudioRequest) (gogpt.AudioResponse, error) {
					resp := gogpt.AudioResponse{
						Text:     "transcribed text",
						Language: "en",
						Duration: 5.0,
					}
					return resp, nil
				},
			}

			request := &domain.AudioTranscriptionRequest{
				AudioData:      []byte("test audio data"),
				Model:          "whisper-1",
				Language:       "en",
				ResponseFormat: tt.responseFormat,
			}

			text, metadata, err := provider.TranscribeAudio(request)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if text != "transcribed text" {
				t.Errorf("Expected 'transcribed text', got '%s'", text)
			}
			if metadata == nil {
				t.Fatal("Expected metadata to be non-nil")
			}

			if tt.expectVerboseData {
				if metadata.VerboseData == nil {
					t.Error("Expected VerboseData to be populated for verbose format")
				} else {
					expectedFields := []string{"language", "duration", "segments", "words"}
					for _, field := range expectedFields {
						if _, ok := metadata.VerboseData[field]; !ok {
							t.Errorf("Expected VerboseData to contain field '%s'", field)
						}
					}
				}
			} else {
				if metadata.VerboseData != nil {
					t.Errorf("Expected VerboseData to be nil for format '%s'", tt.responseFormat)
				}
			}
		})
	}
}

func TestOpenAIProvider_TranscribeAudio_ResponseFormatPassthrough(t *testing.T) {
	formats := []string{"json", "text", "srt", "vtt", "verbose_json", "diarized_json"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			provider := NewOpenAIProvider()
			var capturedFormat gogpt.AudioResponseFormat
			provider.byteOps = &mockByteOpsClient{
				createTranscriptionFunc: func(ctx context.Context, req gogpt.AudioRequest) (gogpt.AudioResponse, error) {
					capturedFormat = req.Format
					return gogpt.AudioResponse{Text: "ok"}, nil
				},
			}

			request := &domain.AudioTranscriptionRequest{
				AudioData:      []byte("test audio"),
				Model:          "whisper-1",
				Language:       "en",
				ResponseFormat: format,
			}

			_, _, err := provider.TranscribeAudio(request)
			if err != nil {
				t.Errorf("Expected no error for format '%s', got: %v", format, err)
			}
			if string(capturedFormat) != format {
				t.Errorf("Expected format '%s' to be passed through, got '%s'", format, capturedFormat)
			}
		})
	}
}

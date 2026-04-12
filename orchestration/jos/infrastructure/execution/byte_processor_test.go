package execution

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"objectweaver/orchestration/jos/domain"

	"objectweaver/jsonSchema"
)

// Mock implementations for testing
type mockByteOperationProvider struct {
	generateAudioFunc   func(request *domain.AudioGenerationRequest) ([]byte, *domain.ProviderMetadata, error)
	generateImageFunc   func(request *domain.ImageGenerationRequest) ([]byte, *domain.ProviderMetadata, error)
	transcribeAudioFunc func(request *domain.AudioTranscriptionRequest) (string, *domain.ProviderMetadata, error)
	supportsByteOps     bool
}

func (m *mockByteOperationProvider) GenerateAudio(request *domain.AudioGenerationRequest) ([]byte, *domain.ProviderMetadata, error) {
	if m.generateAudioFunc != nil {
		return m.generateAudioFunc(request)
	}
	return []byte("mock audio"), &domain.ProviderMetadata{Cost: 0.1, Model: "tts-1", TokensUsed: 10}, nil
}

func (m *mockByteOperationProvider) GenerateImage(request *domain.ImageGenerationRequest) ([]byte, *domain.ProviderMetadata, error) {
	if m.generateImageFunc != nil {
		return m.generateImageFunc(request)
	}
	return []byte("mock image"), &domain.ProviderMetadata{Cost: 0.2, Model: "dall-e-3", TokensUsed: 20}, nil
}

func (m *mockByteOperationProvider) TranscribeAudio(request *domain.AudioTranscriptionRequest) (string, *domain.ProviderMetadata, error) {
	if m.transcribeAudioFunc != nil {
		return m.transcribeAudioFunc(request)
	}
	return "mock transcription", &domain.ProviderMetadata{Cost: 0.05, Model: "whisper-1", TokensUsed: 5}, nil
}

func (m *mockByteOperationProvider) SupportsByteOperations() bool {
	return m.supportsByteOps
}

func (m *mockByteOperationProvider) Generate(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
	return "", nil, errors.New("not implemented")
}

func (m *mockByteOperationProvider) SupportsStreaming() bool { return false }
func (m *mockByteOperationProvider) ModelType() string       { return "gpt-4-0613" }

func TestNewByteProcessor(t *testing.T) {
	llmProvider := &mockByteOperationProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewByteProcessor(llmProvider, promptBuilder)

	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.ttsBuilder == nil {
		t.Error("Expected ttsBuilder to be initialized")
	}
	if processor.imageBuilder == nil {
		t.Error("Expected imageBuilder to be initialized")
	}
	if processor.sttBuilder == nil {
		t.Error("Expected sttBuilder to be initialized")
	}
}

func TestByteProcessor_CanProcess(t *testing.T) {
	processor := NewByteProcessor(nil, nil)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.Byte, true},
		{jsonSchema.Object, false},
		{jsonSchema.String, false},
		{jsonSchema.Number, false},
		{jsonSchema.Boolean, false},
	}

	for _, test := range tests {
		result := processor.CanProcess(test.schemaType)
		if result != test.expected {
			t.Errorf("CanProcess(%v) = %v, expected %v", test.schemaType, result, test.expected)
		}
	}
}

func TestByteProcessor_Process_NoByteOperationsSupport(t *testing.T) {
	llmProvider := &mockByteOperationProvider{supportsByteOps: false}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
	}
	task := domain.NewFieldTask("test", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(testContext(t), task, context)
	if err == nil {
		t.Error("Expected error for unsupported byte operations")
	}

	expectedMsg := "LLM provider does not support byte operations"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestByteProcessor_Process_NoConfig(t *testing.T) {
	llmProvider := &mockByteOperationProvider{supportsByteOps: true}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
	}
	task := domain.NewFieldTask("test", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(testContext(t), task, context)
	if err == nil {
		t.Error("Expected error for missing byte operation config")
	}

	expectedMsg := "byte type requires TextToSpeech, Image, or SpeechToText configuration"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestByteProcessor_Process_TextToSpeech_Success(t *testing.T) {
	llmProvider := &mockByteOperationProvider{
		supportsByteOps: true,
		generateAudioFunc: func(request *domain.AudioGenerationRequest) ([]byte, *domain.ProviderMetadata, error) {
			return []byte("test audio data"), &domain.ProviderMetadata{
				Cost:       0.1,
				Model:      "tts-1",
				TokensUsed: 10,
			}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
		TextToSpeech: &jsonSchema.TextToSpeech{
			StringToAudio: "Hello world",
			Voice:         "alloy",
			Model:         "tts",
		},
		Instruction: "Say this clearly",
	}
	task := domain.NewFieldTask("audioField", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "audioField" {
		t.Errorf("Expected key 'audioField', got %v", result.Key())
	}

	// Check base64 encoding
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("test audio data"))
	if result.Value() != expectedB64 {
		t.Errorf("Expected base64 encoded audio, got %v", result.Value())
	}

	// Check metadata
	if result.Metadata().Cost != 0.1 {
		t.Errorf("Expected cost 0.1, got %v", result.Metadata().Cost)
	}
	if result.Metadata().ModelUsed != "tts-1" {
		t.Errorf("Expected model 'tts-1', got %v", result.Metadata().ModelUsed)
	}
	if result.Metadata().TokensUsed != 10 {
		t.Errorf("Expected tokens 10, got %v", result.Metadata().TokensUsed)
	}
}

func TestByteProcessor_Process_ImageGeneration_Success(t *testing.T) {
	llmProvider := &mockByteOperationProvider{
		supportsByteOps: true,
		generateImageFunc: func(request *domain.ImageGenerationRequest) ([]byte, *domain.ProviderMetadata, error) {
			return []byte("test image data"), &domain.ProviderMetadata{
				Cost:       0.2,
				Model:      "dall-e-3",
				TokensUsed: 20,
			}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
		Image: &jsonSchema.Image{
			Size:  "1024x1024",
			Model: "dall-e-3",
		},
		Instruction: "A beautiful sunset",
	}
	task := domain.NewFieldTask("imageField", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "imageField" {
		t.Errorf("Expected key 'imageField', got %v", result.Key())
	}

	// Check base64 encoding
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("test image data"))
	if result.Value() != expectedB64 {
		t.Errorf("Expected base64 encoded image, got %v", result.Value())
	}

	// Check metadata
	if result.Metadata().Cost != 0.2 {
		t.Errorf("Expected cost 0.2, got %v", result.Metadata().Cost)
	}
	if result.Metadata().ModelUsed != "dall-e-3" {
		t.Errorf("Expected model 'dall-e-3', got %v", result.Metadata().ModelUsed)
	}
	if result.Metadata().TokensUsed != 20 {
		t.Errorf("Expected tokens 20, got %v", result.Metadata().TokensUsed)
	}
}

func TestByteProcessor_Process_SpeechToText_Success(t *testing.T) {
	llmProvider := &mockByteOperationProvider{
		supportsByteOps: true,
		transcribeAudioFunc: func(request *domain.AudioTranscriptionRequest) (string, *domain.ProviderMetadata, error) {
			return "transcribed text", &domain.ProviderMetadata{
				Cost:       0.05,
				Model:      "whisper-1",
				TokensUsed: 5,
			}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
		SpeechToText: &jsonSchema.SpeechToText{
			AudioToTranscribe: []byte("audio data"),
			Language:          "en",
			ToString:          true,
			Model:             "whisper",
		},
		Instruction: "Transcribe this audio",
	}
	task := domain.NewFieldTask("textField", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "textField" {
		t.Errorf("Expected key 'textField', got %v", result.Key())
	}

	if result.Value() != "transcribed text" {
		t.Errorf("Expected 'transcribed text', got %v", result.Value())
	}

	// Check metadata
	if result.Metadata().Cost != 0.05 {
		t.Errorf("Expected cost 0.05, got %v", result.Metadata().Cost)
	}
	if result.Metadata().ModelUsed != "whisper-1" {
		t.Errorf("Expected model 'whisper-1', got %v", result.Metadata().ModelUsed)
	}
	if result.Metadata().TokensUsed != 5 {
		t.Errorf("Expected tokens 5, got %v", result.Metadata().TokensUsed)
	}
}

func TestByteProcessor_Process_TextToSpeech_Error(t *testing.T) {
	llmProvider := &mockByteOperationProvider{
		supportsByteOps: true,
		generateAudioFunc: func(request *domain.AudioGenerationRequest) ([]byte, *domain.ProviderMetadata, error) {
			return nil, nil, errors.New("TTS generation failed")
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
		TextToSpeech: &jsonSchema.TextToSpeech{
			StringToAudio: "Hello",
		},
	}
	task := domain.NewFieldTask("audioField", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(testContext(t), task, context)
	if err == nil {
		t.Error("Expected error from TTS generation")
	}

	if !errors.Is(err, errors.New("TTS generation failed")) && !contains(err.Error(), "TTS generation failed") {
		t.Errorf("Expected TTS error, got: %v", err)
	}
}

func TestByteProcessor_Process_ImageGeneration_Error(t *testing.T) {
	llmProvider := &mockByteOperationProvider{
		supportsByteOps: true,
		generateImageFunc: func(request *domain.ImageGenerationRequest) ([]byte, *domain.ProviderMetadata, error) {
			return nil, nil, errors.New("image generation failed")
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
		Image: &jsonSchema.Image{
			Model: "dall-e-3",
		},
		Instruction: "Generate image",
	}
	task := domain.NewFieldTask("imageField", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(testContext(t), task, context)
	if err == nil {
		t.Error("Expected error from image generation")
	}

	if !contains(err.Error(), "image generation failed") {
		t.Errorf("Expected image generation error, got: %v", err)
	}
}

func TestByteProcessor_Process_SpeechToText_Error(t *testing.T) {
	llmProvider := &mockByteOperationProvider{
		supportsByteOps: true,
		transcribeAudioFunc: func(request *domain.AudioTranscriptionRequest) (string, *domain.ProviderMetadata, error) {
			return "", nil, errors.New("transcription failed")
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
		SpeechToText: &jsonSchema.SpeechToText{
			AudioToTranscribe: []byte("audio"),
		},
	}
	task := domain.NewFieldTask("textField", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(testContext(t), task, context)
	if err == nil {
		t.Error("Expected error from transcription")
	}

	if !contains(err.Error(), "speech-to-text failed") {
		t.Errorf("Expected transcription error, got: %v", err)
	}
}

func TestByteProcessor_Process_SpeechToText_VerboseJsonMetadata(t *testing.T) {
	// Mock provider that simulates returning verbose metadata based on request format
	llmProvider := &mockByteOperationProvider{
		supportsByteOps: true,
		transcribeAudioFunc: func(request *domain.AudioTranscriptionRequest) (string, *domain.ProviderMetadata, error) {
			metadata := &domain.ProviderMetadata{
				Cost:       0.05,
				Model:      "whisper-1",
				TokensUsed: 10,
			}

			// Simulate OpenAI behavior: return verbose data for specific formats
			// Note: In current implementation, verbose_json must be explicitly set in request
			// This test validates that when provider returns VerboseData, it flows through correctly
			if request.ResponseFormat == "json" {
				// Simulate that even "json" format returns verbose data from OpenAI
				// (this is the current behavior - we could use this for testing)
				metadata.VerboseData = map[string]any{
					"language": "en",
					"duration": 3.5,
					"segments": []any{
						map[string]any{
							"id":    0,
							"start": 0.0,
							"end":   3.5,
							"text":  "transcribed text",
						},
					},
					"words": []any{
						map[string]any{
							"word":  "transcribed",
							"start": 0.0,
							"end":   1.0,
						},
						map[string]any{
							"word":  "text",
							"start": 1.5,
							"end":   3.5,
						},
					},
				}
			}

			return "transcribed text", metadata, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	tests := []struct {
		name              string
		toString          bool
		toCaptions        bool
		expectVerboseData bool
	}{
		{
			name:              "json format with verbose data from provider",
			toString:          false,
			toCaptions:        false,
			expectVerboseData: true, // Provider returns verbose data
		},
		{
			name:              "text format excludes verbose metadata",
			toString:          true,
			toCaptions:        false,
			expectVerboseData: false, // Provider doesn't return verbose data
		},
		{
			name:              "srt format excludes verbose metadata",
			toString:          false,
			toCaptions:        true,
			expectVerboseData: false, // Provider doesn't return verbose data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &jsonSchema.Definition{
				Type: jsonSchema.Byte,
				SpeechToText: &jsonSchema.SpeechToText{
					AudioToTranscribe: []byte("audio data"),
					Language:          "en",
					ToString:          tt.toString,
					ToCaptions:        tt.toCaptions,
					Model:             "whisper",
				},
				Instruction: "Transcribe this audio",
			}
			task := domain.NewFieldTask("textField", schema, nil)
			context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

			result, err := processor.Process(testContext(t), task, context)
			if err != nil {
				t.Fatalf("Process failed: %v", err)
			}

			// Check standard fields
			if result.Key() != "textField" {
				t.Errorf("Expected key 'textField', got %v", result.Key())
			}
			if result.Value() != "transcribed text" {
				t.Errorf("Expected 'transcribed text', got %v", result.Value())
			}

			// Check metadata
			if result.Metadata() == nil {
				t.Fatal("Expected metadata to be populated")
			}

			if tt.expectVerboseData {
				if result.Metadata().VerboseData == nil {
					t.Error("Expected VerboseData to be populated when provider returns it")
				} else {
					// Verify verbose data fields
					verboseData := result.Metadata().VerboseData
					if verboseData["language"] != "en" {
						t.Errorf("Expected language 'en', got %v", verboseData["language"])
					}
					if verboseData["duration"] != 3.5 {
						t.Errorf("Expected duration 3.5, got %v", verboseData["duration"])
					}
					if verboseData["segments"] == nil {
						t.Error("Expected segments to be populated")
					}
					if verboseData["words"] == nil {
						t.Error("Expected words to be populated")
					}
				}
			} else {
				if result.Metadata().VerboseData != nil {
					t.Errorf("Expected VerboseData to be nil when provider doesn't return it, got %v", result.Metadata().VerboseData)
				}
			}
		})
	}
}

func TestByteProcessor_Process_SpeechToText_VerboseDataPreservation(t *testing.T) {
	// Test that verbose data flows through the entire pipeline correctly
	expectedVerboseData := map[string]any{
		"language": "es",
		"duration": 5.2,
		"segments": []any{
			map[string]any{"id": 0, "text": "Hola mundo", "start": 0.0, "end": 2.0},
			map[string]any{"id": 1, "text": "Esto es una prueba", "start": 2.5, "end": 5.2},
		},
		"words": []any{
			map[string]any{"word": "Hola", "start": 0.0, "end": 0.5},
			map[string]any{"word": "mundo", "start": 0.6, "end": 2.0},
		},
	}

	llmProvider := &mockByteOperationProvider{
		supportsByteOps: true,
		transcribeAudioFunc: func(request *domain.AudioTranscriptionRequest) (string, *domain.ProviderMetadata, error) {
			return "Hola mundo. Esto es una prueba", &domain.ProviderMetadata{
				Cost:        0.08,
				Model:       "whisper-1",
				TokensUsed:  15,
				VerboseData: expectedVerboseData,
			}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewByteProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Byte,
		SpeechToText: &jsonSchema.SpeechToText{
			AudioToTranscribe: []byte("audio data"),
			Language:          "es",
			Model:             "whisper",
		},
	}
	task := domain.NewFieldTask("transcription", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify verbose data is preserved exactly
	if result.Metadata().VerboseData == nil {
		t.Fatal("Expected VerboseData to be preserved")
	}

	verboseData := result.Metadata().VerboseData

	// Check language
	if lang, ok := verboseData["language"].(string); !ok || lang != "es" {
		t.Errorf("Expected language 'es', got %v", verboseData["language"])
	}

	// Check duration
	if dur, ok := verboseData["duration"].(float64); !ok || dur != 5.2 {
		t.Errorf("Expected duration 5.2, got %v", verboseData["duration"])
	}

	// Check segments exist
	if verboseData["segments"] == nil {
		t.Error("Expected segments to be preserved")
	}

	// Check words exist
	if verboseData["words"] == nil {
		t.Error("Expected words to be preserved")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || strings.Contains(s, substr)))
}

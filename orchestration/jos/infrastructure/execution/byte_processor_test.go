package execution

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"firechimp/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
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

func (m *mockByteOperationProvider) Generate(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
	return "", nil, errors.New("not implemented")
}

func (m *mockByteOperationProvider) SupportsStreaming() bool         { return false }
func (m *mockByteOperationProvider) ModelType() jsonSchema.ModelType { return jsonSchema.Gpt4 }

func TestNewByteProcessor(t *testing.T) {
	llmProvider := &mockByteOperationProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewByteProcessor(llmProvider, promptBuilder)

	if processor.llmProvider != llmProvider {
		t.Error("Expected llmProvider to be set")
	}
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

	_, err := processor.Process(task, context)
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

	_, err := processor.Process(task, context)
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

	result, err := processor.Process(task, context)
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
			Model: jsonSchema.OpenAiDalle3,
		},
		Instruction: "A beautiful sunset",
	}
	task := domain.NewFieldTask("imageField", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
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

	result, err := processor.Process(task, context)
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

	_, err := processor.Process(task, context)
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
			Model: jsonSchema.OpenAiDalle3,
		},
		Instruction: "Generate image",
	}
	task := domain.NewFieldTask("imageField", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(task, context)
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

	_, err := processor.Process(task, context)
	if err == nil {
		t.Error("Expected error from transcription")
	}

	if !contains(err.Error(), "speech-to-text failed") {
		t.Errorf("Expected transcription error, got: %v", err)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || strings.Contains(s, substr)))
}

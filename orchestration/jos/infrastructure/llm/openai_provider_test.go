package llm

import (
	"os"
	"testing"

	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
	gogpt "github.com/sashabaranov/go-openai"
)

// MockJobEntryPoint for testing
type MockJobEntryPoint struct {
	submitJobFunc func(model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error)
}

func (m *MockJobEntryPoint) SubmitJob(model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
	if m.submitJobFunc != nil {
		return m.submitJobFunc(model, def, newPrompt, systemPrompt, outStream)
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
	if model != "gemini-2.0-flash" {
		t.Errorf("Expected gemini-2.0-flash, got %v", model)
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
	os.Setenv("GEMINI_API_KEY", "test-key")
	defer os.Unsetenv("GEMINI_API_KEY")

	model := getDefaultModelForProvider()
	if model != "gemini-2.0-flash" {
		t.Errorf("Expected gemini-2.0-flash when GEMINI_API_KEY is set, got %v", model)
	}
}

func TestOpenAIProvider_Generate(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	provider := NewOpenAIProvider()
	// Replace submitter with mock
	mockSubmitter := &MockJobEntryPoint{
		submitJobFunc: func(model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
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
		submitJobFunc: func(model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
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
		submitJobFunc: func(model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *gogpt.Usage, error) {
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

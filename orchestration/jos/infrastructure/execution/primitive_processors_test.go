package execution

import (
	"testing"

	"objectweaver/orchestration/jos/domain"

	"objectweaver/jsonSchema"
)

func TestNewBooleanProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewBooleanProcessor(llmProvider, promptBuilder)

	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.systemPromptProvider == nil {
		t.Error("Expected systemPromptProvider to be set")
	}
}

func TestNewBooleanProcessorWithPromptProvider(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	promptProvider := &mockSystemPromptProvider{}

	processor := NewBooleanProcessorWithPromptProvider(llmProvider, promptBuilder, promptProvider)

	// Just verify it doesn't panic and fields are set
	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.systemPromptProvider == nil {
		t.Error("Expected systemPromptProvider to be set")
	}
}

func TestBooleanProcessor_CanProcess(t *testing.T) {
	processor := NewBooleanProcessor(nil, nil)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.Boolean, true},
		{jsonSchema.String, false},
		{jsonSchema.Number, false},
		{jsonSchema.Integer, false},
		{jsonSchema.Object, false},
		{jsonSchema.Array, false},
	}

	for _, test := range tests {
		result := processor.CanProcess(test.schemaType)
		if result != test.expected {
			t.Errorf("CanProcess(%v) = %v, expected %v", test.schemaType, result, test.expected)
		}
	}
}

func TestBooleanProcessor_Process(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "true", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewBooleanProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Boolean,
	}

	task := domain.NewFieldTask("isActive", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "isActive" {
		t.Errorf("Expected key 'isActive', got %v", result.Key())
	}

	value, ok := result.Value().(bool)
	if !ok {
		t.Errorf("Expected bool, got %T", result.Value())
	}

	if !value {
		t.Errorf("Expected true, got %v", value)
	}

	if result.Metadata().Cost != 0.01 {
		t.Errorf("Expected cost 0.01, got %v", result.Metadata().Cost)
	}
}

func TestNewNumberProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewNumberProcessor(llmProvider, promptBuilder)

	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.systemPromptProvider == nil {
		t.Error("Expected systemPromptProvider to be set")
	}
}

func TestNewNumberProcessorWithPromptProvider(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	promptProvider := &mockSystemPromptProvider{}

	processor := NewNumberProcessorWithPromptProvider(llmProvider, promptBuilder, promptProvider)

	// Just verify it doesn't panic and fields are set
	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.systemPromptProvider == nil {
		t.Error("Expected systemPromptProvider to be set")
	}
}

func TestNumberProcessor_CanProcess(t *testing.T) {
	processor := NewNumberProcessor(nil, nil)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.Number, true},
		{jsonSchema.Integer, true},
		{jsonSchema.Boolean, false},
		{jsonSchema.String, false},
		{jsonSchema.Object, false},
		{jsonSchema.Array, false},
	}

	for _, test := range tests {
		result := processor.CanProcess(test.schemaType)
		if result != test.expected {
			t.Errorf("CanProcess(%v) = %v, expected %v", test.schemaType, result, test.expected)
		}
	}
}

func TestNumberProcessor_Process(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "42", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewNumberProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Number,
	}

	task := domain.NewFieldTask("count", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "count" {
		t.Errorf("Expected key 'count', got %v", result.Key())
	}

	value, ok := result.Value().(int)
	if !ok {
		t.Errorf("Expected int, got %T", result.Value())
	}

	if value != 42 {
		t.Errorf("Expected 42, got %v", value)
	}

	if result.Metadata().Cost != 0.01 {
		t.Errorf("Expected cost 0.01, got %v", result.Metadata().Cost)
	}
}

func TestNumberProcessor_Process_Integer(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "25", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewNumberProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Integer,
	}

	task := domain.NewFieldTask("age", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	value, ok := result.Value().(int)
	if !ok {
		t.Errorf("Expected int, got %T", result.Value())
	}

	if value != 25 {
		t.Errorf("Expected 25, got %v", value)
	}
}

// Mock for SystemPromptProvider
type mockSystemPromptProvider struct{}

func (m *mockSystemPromptProvider) GetSystemPrompt(dataType jsonSchema.DataType) *string {
	prompt := "mock system prompt"
	return &prompt
}

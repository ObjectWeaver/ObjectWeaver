package execution

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"firechimp/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

// Mock implementations for testing
type mockLLMProvider struct {
	generateFunc func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error)
}

func (m *mockLLMProvider) Generate(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
	if m.generateFunc != nil {
		return m.generateFunc(prompt, config)
	}
	return "", &domain.ProviderMetadata{Cost: 0.1}, nil
}

func (m *mockLLMProvider) SupportsStreaming() bool         { return false }
func (m *mockLLMProvider) ModelType() jsonSchema.ModelType { return jsonSchema.Gpt4 }

type mockPromptBuilder struct {
	buildFunc func(task *domain.FieldTask, context *domain.PromptContext) (string, error)
}

func (m *mockPromptBuilder) Build(task *domain.FieldTask, context *domain.PromptContext) (string, error) {
	if m.buildFunc != nil {
		return m.buildFunc(task, context)
	}
	return fmt.Sprintf("mock prompt for %s", task.Key()), nil
}

func (m *mockPromptBuilder) BuildWithHistory(task *domain.FieldTask, context *domain.PromptContext, history *domain.GenerationHistory) (string, error) {
	return m.Build(task, context)
}

func TestNewArrayProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewArrayProcessor(llmProvider, promptBuilder)

	if processor.llmProvider != llmProvider {
		t.Error("Expected llmProvider to be set")
	}
	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
}

func TestArrayProcessor_CanProcess(t *testing.T) {
	processor := NewArrayProcessor(nil, nil)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.Array, true},
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

func TestArrayProcessor_Process_Success(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			if strings.Contains(prompt, "listString") {
				return "1. Item 1\n2. Item 2\n3. Item 3", &domain.ProviderMetadata{Cost: 0.1}, nil
			}
			if strings.Contains(prompt, "numItems") {
				return "3", &domain.ProviderMetadata{Cost: 0.05}, nil
			}
			// For item generation
			return "item value", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}

	processor := NewArrayProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Array,
		Items: &jsonSchema.Definition{
			Type: jsonSchema.String,
		},
	}

	task := domain.NewFieldTask("testArray", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "testArray" {
		t.Errorf("Expected key 'testArray', got %v", result.Key())
	}

	values, ok := result.Value().([]interface{})
	if !ok {
		t.Errorf("Expected []interface{}, got %T", result.Value())
	}

	if len(values) != 3 {
		t.Errorf("Expected 3 items, got %d", len(values))
	}

	// Cost should be from item generations only (list extraction costs not tracked)
	expectedCost := 3 * 0.01 // 0.03
	if result.Metadata().Cost != expectedCost {
		t.Errorf("Expected cost %v, got %v", expectedCost, result.Metadata().Cost)
	}
}

func TestArrayProcessor_Process_NilItems(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			return "1. Item", &domain.ProviderMetadata{Cost: 0.1}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}

	processor := NewArrayProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:  jsonSchema.Array,
		Items: nil, // Nil items
	}

	task := domain.NewFieldTask("testArray", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(task, context)
	if err == nil {
		t.Error("Expected error for nil items")
	}

	expectedMsg := "array items definition is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestArrayProcessor_Process_SizeDeterminationError(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			if strings.Contains(prompt, "listString") || strings.Contains(prompt, "numItems") {
				return "", nil, errors.New("LLM error")
			}
			// Success for item generation
			return "item value", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}

	processor := NewArrayProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Array,
		Items: &jsonSchema.Definition{
			Type: jsonSchema.String,
		},
	}

	task := domain.NewFieldTask("testArray", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	values, ok := result.Value().([]interface{})
	if !ok {
		t.Errorf("Expected []interface{}, got %T", result.Value())
	}

	if len(values) != 3 { // Default size
		t.Errorf("Expected 3 items (default), got %d", len(values))
	}
}

func TestArrayProcessor_Process_ItemProcessingError(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			if strings.Contains(prompt, "listString") {
				return "1. Item 1", &domain.ProviderMetadata{Cost: 0.1}, nil
			}
			if strings.Contains(prompt, "numItems") {
				return "1", &domain.ProviderMetadata{Cost: 0.05}, nil
			}
			// Fail item generation
			return "", nil, errors.New("item generation failed")
		},
	}
	promptBuilder := &mockPromptBuilder{}

	processor := NewArrayProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Array,
		Items: &jsonSchema.Definition{
			Type: jsonSchema.String,
		},
	}

	task := domain.NewFieldTask("testArray", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(task, context)
	if err == nil {
		t.Error("Expected error from item processing")
	}

	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("Expected generation error, got: %v", err)
	}
}

func TestArrayProcessor_determineArraySize_Success(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			if strings.Contains(prompt, "listString") {
				return "1. Apple\n2. Banana\n3. Cherry", &domain.ProviderMetadata{Cost: 0.1}, nil
			}
			if strings.Contains(prompt, "numItems") {
				return "3", &domain.ProviderMetadata{Cost: 0.05}, nil
			}
			return "", &domain.ProviderMetadata{Cost: 0.1}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}

	processor := NewArrayProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:  jsonSchema.Array,
		Items: &jsonSchema.Definition{Type: jsonSchema.String},
	}

	task := domain.NewFieldTask("fruits", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	size, listString, err := processor.determineArraySize(task, context)
	if err != nil {
		t.Fatalf("determineArraySize failed: %v", err)
	}

	if size != 3 {
		t.Errorf("Expected size 3, got %d", size)
	}

	expectedList := "1. Apple\n2. Banana\n3. Cherry"
	if listString != expectedList {
		t.Errorf("Expected list string '%s', got '%s'", expectedList, listString)
	}
}

func TestArrayProcessor_determineArraySize_Error(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			return "", nil, errors.New("LLM error")
		},
	}
	promptBuilder := &mockPromptBuilder{}

	processor := NewArrayProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:  jsonSchema.Array,
		Items: &jsonSchema.Definition{Type: jsonSchema.String},
	}

	task := domain.NewFieldTask("test", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	size, listString, err := processor.determineArraySize(task, context)
	if err == nil {
		t.Error("Expected error from LLM")
	}

	if size != 3 { // Default
		t.Errorf("Expected default size 3, got %d", size)
	}

	if listString != "" {
		t.Errorf("Expected empty list string, got '%s'", listString)
	}
}

func TestArrayProcessor_extractListInfo(t *testing.T) {
	processor := NewArrayProcessor(nil, nil)

	tests := []struct {
		name         string
		result       map[string]interface{}
		expectedSize int
		expectedList string
	}{
		{
			name: "valid data",
			result: map[string]interface{}{
				"numItems":   5,
				"listString": "1. A\n2. B",
			},
			expectedSize: 5,
			expectedList: "1. A\n2. B",
		},
		{
			name: "string numItems",
			result: map[string]interface{}{
				"numItems":   "7",
				"listString": "items",
			},
			expectedSize: 7,
			expectedList: "items",
		},
		{
			name: "float numItems",
			result: map[string]interface{}{
				"numItems":   2.0,
				"listString": "list",
			},
			expectedSize: 2,
			expectedList: "list",
		},
		{
			name:         "empty result",
			result:       map[string]interface{}{},
			expectedSize: 3, // default
			expectedList: "",
		},
		{
			name: "negative numItems",
			result: map[string]interface{}{
				"numItems": -1,
			},
			expectedSize: 1, // clamped
			expectedList: "",
		},
		{
			name: "too large numItems",
			result: map[string]interface{}{
				"numItems": 200,
			},
			expectedSize: 100, // clamped
			expectedList: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			size, list := processor.extractListInfo(test.result)
			if size != test.expectedSize {
				t.Errorf("Expected size %d, got %d", test.expectedSize, size)
			}
			if list != test.expectedList {
				t.Errorf("Expected list '%s', got '%s'", test.expectedList, list)
			}
		})
	}
}

func TestArrayProcessor_createProcessorForType(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewArrayProcessor(llmProvider, promptBuilder)

	tests := []struct {
		schemaType   jsonSchema.DataType
		expectedType string
	}{
		{jsonSchema.Object, "*execution.ObjectProcessor"},
		{jsonSchema.Array, "*execution.ArrayProcessor"},
		{jsonSchema.String, "*execution.PrimitiveProcessor"},
		{jsonSchema.Number, "*execution.PrimitiveProcessor"},
		{jsonSchema.Boolean, "*execution.PrimitiveProcessor"},
	}

	for _, test := range tests {
		result := processor.createProcessorForType(test.schemaType)
		actualType := fmt.Sprintf("%T", result)
		if actualType != test.expectedType {
			t.Errorf("createProcessorForType(%v) = %s, expected %s", test.schemaType, actualType, test.expectedType)
		}
	}
}

func TestArrayProcessor_createEnhancedContext(t *testing.T) {
	processor := NewArrayProcessor(nil, nil)

	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", &jsonSchema.Definition{Type: jsonSchema.Array}))
	context.PromptContext().AddPrompt("original prompt")

	enhanced := processor.createEnhancedContext(context, "test list")

	// Should return the same context instance but with added prompt
	if enhanced != context {
		t.Error("Expected same context instance")
	}

	prompts := enhanced.PromptContext().Prompts
	if len(prompts) != 2 {
		t.Errorf("Expected 2 prompts, got %d", len(prompts))
	}

	expectedPrompt := "\n\nGeneral information:\ntest list\n\nPlease continue processing items from this list.\n"
	if prompts[1] != expectedPrompt {
		t.Errorf("Expected enhanced prompt '%s', got '%s'", expectedPrompt, prompts[1])
	}
}

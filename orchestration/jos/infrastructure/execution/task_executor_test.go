package execution

import (
	"errors"
	"os"
	"testing"

	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// Mock implementations for testing
type mockTypeProcessor struct {
	canProcessFunc func(schemaType jsonSchema.DataType) bool
	processFunc    func(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error)
}

func (m *mockTypeProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	if m.canProcessFunc != nil {
		return m.canProcessFunc(schemaType)
	}
	return false
}

func (m *mockTypeProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	if m.processFunc != nil {
		return m.processFunc(task, context)
	}
	return domain.NewTaskResult(task.ID(), task.Key(), "mock_value", nil), nil
}

func TestNewCompositeTaskExecutor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	processors := []domain.TypeProcessor{&mockTypeProcessor{}}

	executor := NewCompositeTaskExecutor(llmProvider, promptBuilder, processors)

	if executor.llmProvider != llmProvider {
		t.Error("Expected llmProvider to be set")
	}
	if executor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if len(executor.processors) != 1 {
		t.Error("Expected processors to be set")
	}
	if executor.defaultProc == nil {
		t.Error("Expected default processor to be set")
	}
}

func TestCompositeTaskExecutor_Execute_WithMatchingProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	mockProc := &mockTypeProcessor{
		canProcessFunc: func(schemaType jsonSchema.DataType) bool {
			return schemaType == jsonSchema.String
		},
		processFunc: func(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
			return domain.NewTaskResult(task.ID(), task.Key(), "processed_by_mock", nil), nil
		},
	}
	processors := []domain.TypeProcessor{mockProc}

	executor := NewCompositeTaskExecutor(llmProvider, promptBuilder, processors)

	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("test", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := executor.Execute(task, context)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Value() != "processed_by_mock" {
		t.Errorf("Expected 'processed_by_mock', got %v", result.Value())
	}
}

func TestCompositeTaskExecutor_Execute_FallbackToDefault(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	mockProc := &mockTypeProcessor{
		canProcessFunc: func(schemaType jsonSchema.DataType) bool {
			return false // Never matches
		},
	}
	processors := []domain.TypeProcessor{mockProc}

	executor := NewCompositeTaskExecutor(llmProvider, promptBuilder, processors)

	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("test", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := executor.Execute(task, context)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should use default PrimitiveProcessor
	if result.Key() != "test" {
		t.Errorf("Expected key 'test', got %v", result.Key())
	}
}

func TestCompositeTaskExecutor_ExecuteBatch_Success(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	processors := []domain.TypeProcessor{}

	executor := NewCompositeTaskExecutor(llmProvider, promptBuilder, processors)

	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	task1 := domain.NewFieldTask("field1", schema, nil)
	task2 := domain.NewFieldTask("field2", schema, nil)
	tasks := []*domain.FieldTask{task1, task2}
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	results, err := executor.ExecuteBatch(tasks, context)
	if err != nil {
		t.Fatalf("ExecuteBatch failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestCompositeTaskExecutor_ExecuteBatch_Error(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			return "", nil, errors.New("generation failed")
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processors := []domain.TypeProcessor{}

	executor := NewCompositeTaskExecutor(llmProvider, promptBuilder, processors)

	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	task1 := domain.NewFieldTask("field1", schema, nil)
	task2 := domain.NewFieldTask("field2", schema, nil)
	tasks := []*domain.FieldTask{task1, task2}
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := executor.ExecuteBatch(tasks, context)
	if err == nil {
		t.Error("Expected error from ExecuteBatch")
	}
}

func TestNewPrimitiveProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewPrimitiveProcessor(llmProvider, promptBuilder)

	if processor.llmProvider != llmProvider {
		t.Error("Expected llmProvider to be set")
	}
	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.systemPromptProvider == nil {
		t.Error("Expected systemPromptProvider to be set")
	}
	if processor.maxRetries != 3 {
		t.Error("Expected maxRetries to be 3")
	}
}

func TestNewPrimitiveProcessorWithPromptProvider(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	promptProvider := &mockSystemPromptProvider{}

	processor := NewPrimitiveProcessorWithPromptProvider(llmProvider, promptBuilder, promptProvider)

	if processor.systemPromptProvider != promptProvider {
		t.Error("Expected custom systemPromptProvider to be set")
	}
}

func TestPrimitiveProcessor_CanProcess(t *testing.T) {
	processor := NewPrimitiveProcessor(nil, nil)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.String, true},
		{jsonSchema.Number, true},
		{jsonSchema.Integer, true},
		{jsonSchema.Boolean, true},
		{jsonSchema.Byte, false}, // Handled by ByteProcessor
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

func TestPrimitiveProcessor_Process(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			return "42", &domain.ProviderMetadata{Cost: 0.01, TokensUsed: 10, Model: "test-model"}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewPrimitiveProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{Type: jsonSchema.Integer}
	task := domain.NewFieldTask("age", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "age" {
		t.Errorf("Expected key 'age', got %v", result.Key())
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

func TestPrimitiveProcessor_Process_WithSystemPrompt(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	processor := NewPrimitiveProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:         jsonSchema.String,
		SystemPrompt: stringPtr("Custom system prompt"),
	}
	task := domain.NewFieldTask("field", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "field" {
		t.Errorf("Expected key 'field', got %v", result.Key())
	}
}

func TestPrimitiveProcessor_parseValue(t *testing.T) {
	processor := NewPrimitiveProcessor(nil, nil)

	tests := []struct {
		response  string
		fieldType jsonSchema.DataType
		expected  interface{}
	}{
		{"true", jsonSchema.Boolean, true},
		{"false", jsonSchema.Boolean, false},
		{"42", jsonSchema.Integer, 42},
		{"hello", jsonSchema.String, "hello"},
		{"\"quoted\"", jsonSchema.String, "quoted"},
	}

	for _, test := range tests {
		result := processor.parseValue(test.response, test.fieldType)
		if result != test.expected {
			t.Errorf("parseValue(%q, %v) = %v, expected %v", test.response, test.fieldType, result, test.expected)
		}
	}
}

func TestPrimitiveProcessor_determineModel(t *testing.T) {
	processor := NewPrimitiveProcessor(nil, nil)

	// Test with model in definition
	schema := &jsonSchema.Definition{Type: jsonSchema.String, Model: "gpt-4-0613"}
	task := domain.NewFieldTask("test", schema, nil)

	model := processor.determineModel(task.Definition())
	if model != "gpt-4-0613" {
		t.Errorf("Expected gpt-4-0613, got %v", model)
	}

	// Test without model in definition
	schemaNoModel := &jsonSchema.Definition{Type: jsonSchema.String}
	taskNoModel := domain.NewFieldTask("test", schemaNoModel, nil)

	modelDefault := processor.determineModel(taskNoModel.Definition())
	// Should return default based on environment
	if modelDefault == "" {
		t.Error("Expected non-empty model")
	}
}

func TestGetDefaultModelForProvider(t *testing.T) {
	// Test with LLM_PROVIDER set to openai
	os.Setenv("LLM_PROVIDER", "openai")
	defer os.Unsetenv("LLM_PROVIDER")

	model := getDefaultModelForProvider()
	if model != "gpt-4o-mini" {
		t.Errorf("Expected gpt-4o-mini for openai, got %v", model)
	}

	// Test with LLM_PROVIDER set to gemini
	os.Setenv("LLM_PROVIDER", "gemini")
	model = getDefaultModelForProvider()
	if model != "gemini-2.0-flash" {
		t.Errorf("Expected gemini-2.0-flash for gemini, got %v", model)
	}

	// Test with no provider set but API_URL
	os.Unsetenv("LLM_PROVIDER")
	os.Setenv("LLM_API_URL", "http://localhost:8000")
	defer os.Unsetenv("LLM_API_URL")

	model = getDefaultModelForProvider()
	if model != "gpt-4o-mini" {
		t.Errorf("Expected gpt-4o-mini for API_URL, got %v", model)
	}
}

func TestCleanResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`  "world"  `, `  "world"  `}, // Whitespace is not trimmed
		{"no quotes", "no quotes"},
	}

	for _, test := range tests {
		result := cleanResponse(test.input)
		if result != test.expected {
			t.Errorf("cleanResponse(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`"`, `"`},
		{"no quotes", "no quotes"},
		{`""`, ""},
	}

	for _, test := range tests {
		result := trimQuotes(test.input)
		if result != test.expected {
			t.Errorf("trimQuotes(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestTrimWhitespace(t *testing.T) {
	// Note: Current implementation is a no-op
	result := trimWhitespace("  hello  ")
	if result != "  hello  " {
		t.Errorf("trimWhitespace did not preserve whitespace")
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

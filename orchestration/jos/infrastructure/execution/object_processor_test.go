package execution

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"objectGeneration/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

func TestNewObjectProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewObjectProcessor(llmProvider, promptBuilder)

	if processor.llmProvider != llmProvider {
		t.Error("Expected llmProvider to be set")
	}
	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.analyzer != nil {
		t.Error("Expected analyzer to be nil initially")
	}
}

func TestObjectProcessor_CanProcess(t *testing.T) {
	processor := NewObjectProcessor(nil, nil)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.Object, true},
		{jsonSchema.Array, false},
		{jsonSchema.String, false},
		{jsonSchema.Number, false},
		{jsonSchema.Boolean, false},
		{jsonSchema.Map, false},
	}

	for _, test := range tests {
		result := processor.CanProcess(test.schemaType)
		if result != test.expected {
			t.Errorf("CanProcess(%v) = %v, expected %v", test.schemaType, result, test.expected)
		}
	}
}

func TestObjectProcessor_Process_Success(t *testing.T) {
	callCount := 0
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			callCount++
			if callCount == 1 {
				return "John", &domain.ProviderMetadata{Cost: 0.01}, nil
			}
			return "25", &domain.ProviderMetadata{Cost: 0.01}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewObjectProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"name": {
				Type: jsonSchema.String,
			},
			"age": {
				Type: jsonSchema.Number,
			},
		},
	}

	task := domain.NewFieldTask("user", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "user" {
		t.Errorf("Expected key 'user', got %v", result.Key())
	}

	value, ok := result.Value().(map[string]interface{})
	if !ok {
		t.Errorf("Expected map[string]interface{}, got %T", result.Value())
	}

	if len(value) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(value))
	}

	if value["name"] != "John" {
		t.Errorf("Expected name 'John', got %v", value["name"])
	}

	if value["age"] != 25 {
		t.Errorf("Expected age 25, got %v", value["age"])
	}

	// Cost should be 2 * 0.01 = 0.02
	if result.Metadata().Cost != 0.02 {
		t.Errorf("Expected cost 0.02, got %v", result.Metadata().Cost)
	}
}

func TestObjectProcessor_Process_EmptyObject(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	processor := NewObjectProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:       jsonSchema.Object,
		Properties: nil, // No properties
	}

	task := domain.NewFieldTask("empty", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	value, ok := result.Value().(map[string]interface{})
	if !ok {
		t.Errorf("Expected map[string]interface{}, got %T", result.Value())
	}

	if len(value) != 0 {
		t.Errorf("Expected empty map, got %v", value)
	}

	if result.Metadata().Cost != 0.0 {
		t.Errorf("Expected cost 0.0, got %v", result.Metadata().Cost)
	}
}

func TestObjectProcessor_Process_NestedError(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
			return "", nil, errors.New("generation failed")
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewObjectProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"name": {
				Type: jsonSchema.String,
			},
		},
	}

	task := domain.NewFieldTask("user", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(task, context)
	if err == nil {
		t.Error("Expected error from nested task processing")
	}

	if !strings.Contains(err.Error(), "nested task name failed") {
		t.Errorf("Expected nested task error, got: %v", err)
	}
}

func TestObjectProcessor_extractFields(t *testing.T) {
	processor := NewObjectProcessor(nil, nil)

	// Test with properties
	def := &jsonSchema.Definition{
		Properties: map[string]jsonSchema.Definition{
			"field1": {Type: jsonSchema.String},
			"field2": {Type: jsonSchema.Number},
		},
	}

	fields, err := processor.extractFields(def)
	if err != nil {
		t.Fatalf("extractFields failed: %v", err)
	}

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}

	if fields["field1"].Type != jsonSchema.String {
		t.Errorf("Expected field1 type String, got %v", fields["field1"].Type)
	}

	// Test with nil properties
	defNil := &jsonSchema.Definition{
		Properties: nil,
	}

	fieldsNil, err := processor.extractFields(defNil)
	if err != nil {
		t.Fatalf("extractFields failed: %v", err)
	}

	if len(fieldsNil) != 0 {
		t.Errorf("Expected empty fields, got %v", fieldsNil)
	}
}

func TestObjectProcessor_createProcessorForType(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	processor := NewObjectProcessor(llmProvider, promptBuilder)

	tests := []struct {
		schemaType   jsonSchema.DataType
		expectedType string
	}{
		{jsonSchema.Object, "*execution.ObjectProcessor"},
		{jsonSchema.Array, "*execution.ArrayProcessor"},
		{jsonSchema.Map, "*execution.MapProcessor"},
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

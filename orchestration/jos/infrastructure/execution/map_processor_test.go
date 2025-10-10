package execution

import (
	"objectweaver/orchestration/jos/domain"
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

func TestNewMapProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewMapProcessor(llmProvider, promptBuilder)

	if processor.llmProvider != llmProvider {
		t.Error("Expected llmProvider to be set")
	}
	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
}

func TestMapProcessor_CanProcess(t *testing.T) {
	processor := NewMapProcessor(nil, nil)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.Map, true},
		{jsonSchema.Object, false},
		{jsonSchema.String, false},
		{jsonSchema.Number, false},
		{jsonSchema.Boolean, false},
		{jsonSchema.Array, false},
	}

	for _, test := range tests {
		result := processor.CanProcess(test.schemaType)
		if result != test.expected {
			t.Errorf("CanProcess(%v) = %v, expected %v", test.schemaType, result, test.expected)
		}
	}
}

func TestMapProcessor_Process(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	processor := NewMapProcessor(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Map,
	}

	task := domain.NewFieldTask("testMap", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "testMap" {
		t.Errorf("Expected key 'testMap', got %v", result.Key())
	}

	// Check that value is an empty map
	value, ok := result.Value().(map[string]interface{})
	if !ok {
		t.Errorf("Expected map[string]interface{}, got %T", result.Value())
	}

	if len(value) != 0 {
		t.Errorf("Expected empty map, got %v", value)
	}

	// Check metadata is initialized
	if result.Metadata() == nil {
		t.Error("Expected non-nil metadata")
	}

	// Check path is set
	if result.Path() == nil {
		t.Error("Expected non-nil path")
	}
}

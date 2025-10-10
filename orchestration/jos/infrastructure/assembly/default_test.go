package assembly

import (
	"errors"
	"reflect"
	"testing"

	"objectGeneration/orchestration/jos/domain"
)

func TestNewDefaultAssembler(t *testing.T) {
	assembler := NewDefaultAssembler()
	if assembler == nil {
		t.Error("NewDefaultAssembler returned nil")
	}
}

func TestAssemble_EmptyResults(t *testing.T) {
	assembler := NewDefaultAssembler()
	results := []*domain.TaskResult{}

	result, err := assembler.Assemble(results)
	if err != nil {
		t.Errorf("Assemble returned error: %v", err)
	}
	if result == nil {
		t.Error("Assemble returned nil result")
	}
	if result.Data() == nil {
		t.Error("Data is nil")
	}
	if len(result.Data()) != 0 {
		t.Errorf("Expected empty data, got %v", result.Data())
	}
	if result.Metadata() == nil {
		t.Error("Metadata is nil")
	}
	if result.Metadata().FieldCount != 0 {
		t.Errorf("Expected FieldCount 0, got %d", result.Metadata().FieldCount)
	}
}

func TestAssemble_SuccessfulResults(t *testing.T) {
	assembler := NewDefaultAssembler()

	metadata1 := domain.NewResultMetadata()
	metadata1.AddCost(1.5)
	metadata1.AddTokens(100)

	metadata2 := domain.NewResultMetadata()
	metadata2.AddCost(2.0)
	metadata2.AddTokens(200)

	result1 := domain.NewTaskResult("task1", "key1", "value1", metadata1).WithPath([]string{"data", "field1"})
	result2 := domain.NewTaskResult("task2", "key2", 42, metadata2).WithPath([]string{"data", "field2"})

	results := []*domain.TaskResult{result1, result2}

	result, err := assembler.Assemble(results)
	if err != nil {
		t.Errorf("Assemble returned error: %v", err)
	}
	if result == nil {
		t.Error("Assemble returned nil result")
	}

	expectedData := map[string]interface{}{
		"data": map[string]interface{}{
			"field1": "value1",
			"field2": 42,
		},
	}
	if !reflect.DeepEqual(result.Data(), expectedData) {
		t.Errorf("Expected data %v, got %v", expectedData, result.Data())
	}

	if result.Metadata().FieldCount != 2 {
		t.Errorf("Expected FieldCount 2, got %d", result.Metadata().FieldCount)
	}
	if result.Metadata().Cost != 3.5 {
		t.Errorf("Expected Cost 3.5, got %f", result.Metadata().Cost)
	}
	if result.Metadata().TokensUsed != 300 {
		t.Errorf("Expected TokensUsed 300, got %d", result.Metadata().TokensUsed)
	}
}

func TestAssemble_FailedResults(t *testing.T) {
	assembler := NewDefaultAssembler()

	successResult := domain.NewTaskResult("task1", "key1", "value1", domain.NewResultMetadata()).WithPath([]string{"data", "field1"})
	failedResult := domain.NewTaskResultWithError("task2", "key2", errors.New("some error"))

	results := []*domain.TaskResult{successResult, failedResult}

	result, err := assembler.Assemble(results)
	if err != nil {
		t.Errorf("Assemble returned error: %v", err)
	}

	expectedData := map[string]interface{}{
		"data": map[string]interface{}{
			"field1": "value1",
		},
	}
	if !reflect.DeepEqual(result.Data(), expectedData) {
		t.Errorf("Expected data %v, got %v", expectedData, result.Data())
	}

	if result.Metadata().FieldCount != 1 {
		t.Errorf("Expected FieldCount 1, got %d", result.Metadata().FieldCount)
	}
}

func TestAssemble_NestedPaths(t *testing.T) {
	assembler := NewDefaultAssembler()

	result1 := domain.NewTaskResult("task1", "key1", "value1", domain.NewResultMetadata()).WithPath([]string{"level1", "level2", "field1"})
	result2 := domain.NewTaskResult("task2", "key2", "value2", domain.NewResultMetadata()).WithPath([]string{"level1", "level2", "field2"})

	results := []*domain.TaskResult{result1, result2}

	result, err := assembler.Assemble(results)
	if err != nil {
		t.Errorf("Assemble returned error: %v", err)
	}

	expectedData := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"field1": "value1",
				"field2": "value2",
			},
		},
	}
	if !reflect.DeepEqual(result.Data(), expectedData) {
		t.Errorf("Expected data %v, got %v", expectedData, result.Data())
	}

	if result.Metadata().FieldCount != 2 {
		t.Errorf("Expected FieldCount 2, got %d", result.Metadata().FieldCount)
	}
}

func TestDefaultAssembler_SetNestedValue(t *testing.T) {
	assembler := NewDefaultAssembler()
	data := make(map[string]interface{})

	// Test single level
	assembler.setNestedValue(data, []string{"key1"}, "value1")
	expected := map[string]interface{}{"key1": "value1"}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("Expected %v, got %v", expected, data)
	}

	// Test nested
	assembler.setNestedValue(data, []string{"key2", "subkey"}, "value2")
	expected = map[string]interface{}{
		"key1": "value1",
		"key2": map[string]interface{}{
			"subkey": "value2",
		},
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("Expected %v, got %v", expected, data)
	}

	// Test deeper nesting
	assembler.setNestedValue(data, []string{"key3", "sub1", "sub2"}, 123)
	expected["key3"] = map[string]interface{}{
		"sub1": map[string]interface{}{
			"sub2": 123,
		},
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("Expected %v, got %v", expected, data)
	}

	// Test empty path (should do nothing)
	assembler.setNestedValue(data, []string{}, "ignored")
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("Empty path should not change data, expected %v, got %v", expected, data)
	}
}

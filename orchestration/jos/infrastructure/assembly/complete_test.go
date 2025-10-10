package assembly

import (
	"errors"
	"reflect"
	"testing"

	"objectweaver/orchestration/jos/domain"
)

func TestNewCompleteStreamingAssembler(t *testing.T) {
	assembler := NewCompleteStreamingAssembler()
	if assembler == nil {
		t.Fatal("NewCompleteStreamingAssembler returned nil")
	}
	if assembler.accumulated == nil {
		t.Error("accumulated map not initialized")
	}
	if len(assembler.accumulated) != 0 {
		t.Error("accumulated map should be empty initially")
	}
}

func TestAssemble(t *testing.T) {
	assembler := NewCompleteStreamingAssembler()
	results := []*domain.TaskResult{
		domain.NewTaskResult("task1", "key1", "value1", domain.NewResultMetadata()).WithPath([]string{"key1"}),
	}

	result, err := assembler.Assemble(results)
	if err != nil {
		t.Fatalf("Assemble returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Assemble returned nil result")
	}
	// Since it falls back to default assembler, check if data is set
	data := result.Data()
	if data == nil {
		t.Error("Expected data map, got nil")
	}
	if val, ok := data["key1"]; !ok || val != "value1" {
		t.Errorf("Expected key1=value1, got %v", val)
	}
}

func TestAssembleStreaming(t *testing.T) {
	assembler := NewCompleteStreamingAssembler()
	resultsChan := make(chan *domain.TaskResult, 10)

	// Send some results
	go func() {
		resultsChan <- domain.NewTaskResult("task1", "name", "John", domain.NewResultMetadata()).WithPath([]string{"name"})
		resultsChan <- domain.NewTaskResult("task2", "age", 30, domain.NewResultMetadata()).WithPath([]string{"age"})
		resultsChan <- domain.NewTaskResult("task3", "city", "NYC", domain.NewResultMetadata()).WithPath([]string{"address", "city"})
		close(resultsChan)
	}()

	streamChan, err := assembler.AssembleStreaming(resultsChan)
	if err != nil {
		t.Fatalf("AssembleStreaming returned error: %v", err)
	}

	var chunks []*domain.StreamChunk
	for chunk := range streamChan {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 4 { // 3 intermediate + 1 final
		t.Errorf("Expected 4 chunks, got %d", len(chunks))
	}

	// Check intermediate chunks
	expectedKeys := []string{"name", "age", "city"}
	for i, chunk := range chunks[:3] {
		if chunk.NewKey != expectedKeys[i] {
			t.Errorf("Chunk %d: expected NewKey %s, got %s", i, expectedKeys[i], chunk.NewKey)
		}
		if chunk.AccumulatedData == nil {
			t.Errorf("Chunk %d: AccumulatedData is nil", i)
		}
	}

	// Check final chunk
	finalChunk := chunks[3]
	if !finalChunk.IsFinal {
		t.Error("Final chunk should be marked as final")
	}
	if finalChunk.AccumulatedData == nil {
		t.Error("Final chunk AccumulatedData is nil")
	}

	// Check accumulated data
	data := finalChunk.AccumulatedData
	if name, ok := data["name"]; !ok || name != "John" {
		t.Errorf("Expected name=John, got %v", name)
	}
	if age, ok := data["age"]; !ok || age != 30 {
		t.Errorf("Expected age=30, got %v", age)
	}
	if addr, ok := data["address"].(map[string]interface{}); !ok {
		t.Error("address should be a map")
	} else {
		if city, ok := addr["city"]; !ok || city != "NYC" {
			t.Errorf("Expected address.city=NYC, got %v", city)
		}
	}
}

func TestAssembleStreamingWithErrors(t *testing.T) {
	assembler := NewCompleteStreamingAssembler()
	resultsChan := make(chan *domain.TaskResult, 10)

	go func() {
		resultsChan <- domain.NewTaskResult("task1", "name", "John", domain.NewResultMetadata()).WithPath([]string{"name"})
		resultsChan <- domain.NewTaskResultWithError("task2", "age", errors.New("some error")) // Error result
		resultsChan <- domain.NewTaskResult("task3", "city", "NYC", domain.NewResultMetadata()).WithPath([]string{"city"})
		close(resultsChan)
	}()

	streamChan, err := assembler.AssembleStreaming(resultsChan)
	if err != nil {
		t.Fatalf("AssembleStreaming returned error: %v", err)
	}

	var chunks []*domain.StreamChunk
	for chunk := range streamChan {
		chunks = append(chunks, chunk)
	}

	// Should have 3 chunks: 2 successful + 1 final
	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	finalChunk := chunks[2]
	if !finalChunk.IsFinal {
		t.Error("Final chunk should be marked as final")
	}

	data := finalChunk.AccumulatedData
	if len(data) != 2 {
		t.Errorf("Expected 2 fields in data, got %d", len(data))
	}
	if _, ok := data["name"]; !ok {
		t.Error("name should be in data")
	}
	if _, ok := data["city"]; !ok {
		t.Error("city should be in data")
	}
	if _, ok := data["age"]; ok {
		t.Error("age should not be in data (error result)")
	}
}

func TestSetNestedValue(t *testing.T) {
	assembler := NewCompleteStreamingAssembler()

	// Test simple key
	obj := make(map[string]interface{})
	assembler.setNestedValue(obj, []string{"key1"}, "value1")
	if obj["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", obj["key1"])
	}

	// Test nested key
	assembler.setNestedValue(obj, []string{"nested", "key2"}, "value2")
	if nested, ok := obj["nested"].(map[string]interface{}); !ok {
		t.Error("nested should be a map")
	} else {
		if nested["key2"] != "value2" {
			t.Errorf("Expected nested.key2=value2, got %v", nested["key2"])
		}
	}

	// Test deeper nesting
	assembler.setNestedValue(obj, []string{"a", "b", "c"}, 123)
	if a, ok := obj["a"].(map[string]interface{}); !ok {
		t.Error("a should be a map")
	} else {
		if b, ok := a["b"].(map[string]interface{}); !ok {
			t.Error("b should be a map")
		} else {
			if b["c"] != 123 {
				t.Errorf("Expected a.b.c=123, got %v", b["c"])
			}
		}
	}

	// Test empty path (should do nothing)
	assembler.setNestedValue(obj, []string{}, "ignored")
	// obj should remain unchanged
}

func TestDeepCopy(t *testing.T) {
	assembler := NewCompleteStreamingAssembler()

	original := map[string]interface{}{
		"string": "value",
		"number": 42,
		"nested": map[string]interface{}{
			"inner": "data",
		},
		"slice": []interface{}{"a", "b", 1},
	}

	copy := assembler.deepCopy(original)

	// Check deep copy
	if !reflect.DeepEqual(original, copy) {
		t.Error("Deep copy should be equal to original")
	}

	// Modify copy and ensure original is unchanged
	copy["new"] = "added"
	if _, ok := original["new"]; ok {
		t.Error("Original should not have new key")
	}

	if nested, ok := copy["nested"].(map[string]interface{}); ok {
		nested["inner"] = "modified"
		if original["nested"].(map[string]interface{})["inner"] == "modified" {
			t.Error("Original nested should not be modified")
		}
	}

	if slice, ok := copy["slice"].([]interface{}); ok {
		slice[0] = "modified"
		if original["slice"].([]interface{})[0] == "modified" {
			t.Error("Original slice should not be modified")
		}
	}
}

func TestDeepCopySlice(t *testing.T) {
	assembler := NewCompleteStreamingAssembler()

	original := []interface{}{
		"string",
		42,
		map[string]interface{}{"key": "value"},
		[]interface{}{"a", "b"},
	}

	copy := assembler.deepCopySlice(original)

	if !reflect.DeepEqual(original, copy) {
		t.Error("Deep copy slice should be equal to original")
	}

	// Modify copy
	copy[0] = "modified"
	if original[0] == "modified" {
		t.Error("Original slice should not be modified")
	}

	if m, ok := copy[2].(map[string]interface{}); ok {
		m["key"] = "modified"
		if original[2].(map[string]interface{})["key"] == "modified" {
			t.Error("Original nested map in slice should not be modified")
		}
	}

	if s, ok := copy[3].([]interface{}); ok {
		s[0] = "modified"
		if original[3].([]interface{})[0] == "modified" {
			t.Error("Original nested slice should not be modified")
		}
	}
}

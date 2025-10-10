package assembly

import (
	"testing"
	"time"

	"objectweaver/orchestration/jos/domain"
)

func TestNewProgressiveObjectAssembler(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(500)

	if assembler == nil {
		t.Fatal("Expected non-nil assembler")
	}

	if assembler.currentMap == nil {
		t.Error("Expected currentMap to be initialized")
	}

	if assembler.progressiveFields == nil {
		t.Error("Expected progressiveFields to be initialized")
	}

	if assembler.emitInterval != 500*time.Millisecond {
		t.Errorf("Expected emitInterval to be 500ms, got %v", assembler.emitInterval)
	}
}

func TestProgressiveObjectAssembler_Assemble(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(500)

	results := []*domain.TaskResult{
		domain.NewTaskResult("task1", "field1", "value1", domain.NewResultMetadata()),
	}

	result, err := assembler.Assemble(results)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Since it delegates to default assembler, we can't test much here without mocking
	// But we can check that it doesn't panic
}

func TestUpdateProgressiveValue(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(500)

	token := &domain.TokenStreamChunk{
		Key:      "testKey",
		Path:     []string{"root", "field"},
		Token:    "test",
		Complete: false,
	}

	assembler.updateProgressiveValue(token)

	// Check that progressive field was created
	pv, exists := assembler.progressiveFields["testKey"]
	if !exists {
		t.Error("Expected progressive field to be created")
	}

	if pv.Key() != "testKey" {
		t.Errorf("Expected key 'testKey', got '%s'", pv.Key())
	}

	if pv.CurrentValue() != "test" {
		t.Errorf("Expected current value 'test', got '%s'", pv.CurrentValue())
	}

	if pv.IsComplete() {
		t.Error("Expected field to not be complete")
	}

	// Check nested value in currentMap
	root, exists := assembler.currentMap["root"]
	if !exists {
		t.Error("Expected root to exist in currentMap")
	}

	rootMap, ok := root.(map[string]interface{})
	if !ok {
		t.Error("Expected root to be a map")
	}

	field, exists := rootMap["field"]
	if !exists {
		t.Error("Expected field to exist in root map")
	}

	if field != "test" {
		t.Errorf("Expected field value 'test', got '%v'", field)
	}
}

func TestUpdateProgressiveValueComplete(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(500)

	token := &domain.TokenStreamChunk{
		Key:      "testKey",
		Path:     []string{"field"},
		Token:    "complete",
		Complete: true,
	}

	assembler.updateProgressiveValue(token)

	pv := assembler.progressiveFields["testKey"]
	if !pv.IsComplete() {
		t.Error("Expected field to be complete")
	}

	if assembler.currentMap["field"] != "complete" {
		t.Errorf("Expected currentMap field to be 'complete', got '%v'", assembler.currentMap["field"])
	}
}

func TestCalculateProgress(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(500)

	// No fields
	progress := assembler.calculateProgress()
	if progress != 0.0 {
		t.Errorf("Expected progress 0.0, got %f", progress)
	}

	// Add incomplete field
	token1 := &domain.TokenStreamChunk{Key: "key1", Path: []string{"field1"}, Token: "val1", Complete: false}
	assembler.updateProgressiveValue(token1)

	progress = assembler.calculateProgress()
	if progress != 0.0 {
		t.Errorf("Expected progress 0.0, got %f", progress)
	}

	// Add complete field
	token2 := &domain.TokenStreamChunk{Key: "key2", Path: []string{"field2"}, Token: "val2", Complete: true}
	assembler.updateProgressiveValue(token2)

	progress = assembler.calculateProgress()
	if progress != 0.5 {
		t.Errorf("Expected progress 0.5, got %f", progress)
	}

	// Complete first field
	token3 := &domain.TokenStreamChunk{Key: "key1", Path: []string{"field1"}, Token: "val1", Complete: true}
	assembler.updateProgressiveValue(token3)

	progress = assembler.calculateProgress()
	if progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", progress)
	}
}

func TestProgressiveObjectAssembler_SetNestedValue(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(500)

	obj := make(map[string]interface{})

	// Simple path
	assembler.setNestedValue(obj, []string{"key"}, "value")
	if obj["key"] != "value" {
		t.Errorf("Expected 'value', got '%v'", obj["key"])
	}

	// Nested path
	assembler.setNestedValue(obj, []string{"parent", "child"}, "nestedValue")
	parent, ok := obj["parent"].(map[string]interface{})
	if !ok {
		t.Error("Expected parent to be a map")
	}
	if parent["child"] != "nestedValue" {
		t.Errorf("Expected 'nestedValue', got '%v'", parent["child"])
	}

	// Empty path
	assembler.setNestedValue(obj, []string{}, "empty")
	// Should not change anything
}

func TestProgressiveObjectAssembler_DeepCopy(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(500)

	original := map[string]interface{}{
		"string": "value",
		"number": 42,
		"nested": map[string]interface{}{
			"inner": "innerValue",
		},
		"slice": []interface{}{"item1", "item2"},
	}

	copy := assembler.deepCopy(original)

	// Check values
	if copy["string"] != "value" {
		t.Error("String not copied correctly")
	}

	if copy["number"] != 42 {
		t.Error("Number not copied correctly")
	}

	nested, ok := copy["nested"].(map[string]interface{})
	if !ok {
		t.Error("Nested map not copied")
	}
	if nested["inner"] != "innerValue" {
		t.Error("Nested value not copied")
	}

	slice, ok := copy["slice"].([]interface{})
	if !ok {
		t.Error("Slice not copied")
	}
	if len(slice) != 2 || slice[0] != "item1" || slice[1] != "item2" {
		t.Error("Slice not copied correctly")
	}

	// Modify original and check copy is independent
	original["string"] = "modified"
	if copy["string"] != "value" {
		t.Error("Copy not independent")
	}
}

func TestAssembleProgressive(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(100) // Short interval for testing

	tokenStream := make(chan *domain.TokenStreamChunk, 10)

	// Send some tokens
	tokenStream <- &domain.TokenStreamChunk{Key: "key1", Path: []string{"field1"}, Token: "part1", Complete: false}
	tokenStream <- &domain.TokenStreamChunk{Key: "key1", Path: []string{"field1"}, Token: "part2", Complete: true}
	tokenStream <- &domain.TokenStreamChunk{Key: "key2", Path: []string{"field2"}, Token: "value2", Complete: true}
	close(tokenStream)

	out, err := assembler.AssembleProgressive(tokenStream)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	var chunks []*domain.AccumulatedStreamChunk
	for chunk := range out {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Check final chunk
	finalChunk := chunks[len(chunks)-1]
	if !finalChunk.IsFinal {
		t.Error("Expected final chunk to be marked as final")
	}

	if finalChunk.Progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", finalChunk.Progress)
	}

	// Check assembled data
	if finalChunk.CurrentMap["field1"] != "part1part2" {
		t.Errorf("Expected field1 to be 'part1part2', got '%v'", finalChunk.CurrentMap["field1"])
	}

	if finalChunk.CurrentMap["field2"] != "value2" {
		t.Errorf("Expected field2 to be 'value2', got '%v'", finalChunk.CurrentMap["field2"])
	}
}

func TestEmitCurrentState(t *testing.T) {
	assembler := NewProgressiveObjectAssembler(500)

	// Add some data
	token := &domain.TokenStreamChunk{Key: "key1", Path: []string{"field1"}, Token: "value1", Complete: true}
	assembler.updateProgressiveValue(token)

	out := make(chan *domain.AccumulatedStreamChunk, 1)

	assembler.emitCurrentState(out, token, false)

	select {
	case chunk := <-out:
		if chunk.IsFinal {
			t.Error("Expected not final")
		}
		if chunk.NewToken != token {
			t.Error("Expected new token to match")
		}
		if chunk.Progress != 1.0 {
			t.Errorf("Expected progress 1.0, got %f", chunk.Progress)
		}
		if chunk.CurrentMap["field1"] != "value1" {
			t.Errorf("Expected field1 'value1', got '%v'", chunk.CurrentMap["field1"])
		}
	default:
		t.Error("Expected chunk to be emitted")
	}
}

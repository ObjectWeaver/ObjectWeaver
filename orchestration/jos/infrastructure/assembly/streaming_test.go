package assembly

import (
	"errors"
	"testing"

	"objectweaver/orchestration/jos/domain"
)

func TestNewStreamingAssembler(t *testing.T) {
	assembler := NewStreamingAssembler()
	if assembler == nil {
		t.Error("NewStreamingAssembler returned nil")
	}
}

func TestStreamingAssembler_Assemble(t *testing.T) {
	assembler := NewStreamingAssembler()

	// Create some task results
	results := []*domain.TaskResult{
		domain.NewTaskResult("task1", "key1", "value1", domain.NewResultMetadata()),
		domain.NewTaskResult("task2", "key2", "value2", domain.NewResultMetadata()),
	}

	result, err := assembler.Assemble(results)
	if err != nil {
		t.Errorf("Assemble returned error: %v", err)
	}
	if result == nil {
		t.Error("Assemble returned nil result")
	}
	// Since it delegates to DefaultAssembler, we expect it to work as per default tests
}

func TestStreamingAssembler_AssembleStreaming(t *testing.T) {
	assembler := NewStreamingAssembler()

	results := make(chan *domain.TaskResult, 10)

	// Send successful results
	result1 := domain.NewTaskResult("task1", "name", "John", domain.NewResultMetadata()).WithPath([]string{"user", "name"})
	result2 := domain.NewTaskResult("task2", "age", 30, domain.NewResultMetadata()).WithPath([]string{"user", "age"})
	result3 := domain.NewTaskResultWithError("task3", "email", errors.New("failed"))

	results <- result1
	results <- result2
	results <- result3
	close(results)

	stream, err := assembler.AssembleStreaming(results)
	if err != nil {
		t.Errorf("AssembleStreaming returned error: %v", err)
	}

	var chunks []*domain.StreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	// Should have 3 chunks: 2 data + 1 final
	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	// Check first chunk
	if chunks[0].Key != "name" {
		t.Errorf("Expected key 'name', got %s", chunks[0].Key)
	}
	if chunks[0].Value != "John" {
		t.Errorf("Expected value 'John', got %v", chunks[0].Value)
	}
	if len(chunks[0].Path) != 2 || chunks[0].Path[0] != "user" || chunks[0].Path[1] != "name" {
		t.Errorf("Expected path ['user', 'name'], got %v", chunks[0].Path)
	}
	if chunks[0].IsFinal {
		t.Error("First chunk should not be final")
	}

	// Check second chunk
	if chunks[1].Key != "age" {
		t.Errorf("Expected key 'age', got %s", chunks[1].Key)
	}
	if chunks[1].Value != 30 {
		t.Errorf("Expected value 30, got %v", chunks[1].Value)
	}
	if len(chunks[1].Path) != 2 || chunks[1].Path[0] != "user" || chunks[1].Path[1] != "age" {
		t.Errorf("Expected path ['user', 'age'], got %v", chunks[1].Path)
	}
	if chunks[1].IsFinal {
		t.Error("Second chunk should not be final")
	}

	// Check final chunk
	if chunks[2].Key != "" {
		t.Errorf("Expected final key '', got %s", chunks[2].Key)
	}
	if chunks[2].Value != nil {
		t.Errorf("Expected final value nil, got %v", chunks[2].Value)
	}
	if !chunks[2].IsFinal {
		t.Error("Final chunk should be final")
	}
}

func TestStreamingAssembler_AssembleStreaming_NoResults(t *testing.T) {
	assembler := NewStreamingAssembler()

	results := make(chan *domain.TaskResult)
	close(results)

	stream, err := assembler.AssembleStreaming(results)
	if err != nil {
		t.Errorf("AssembleStreaming returned error: %v", err)
	}

	var chunks []*domain.StreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	// Should have 1 chunk: final
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	if !chunks[0].IsFinal {
		t.Error("Chunk should be final")
	}
}

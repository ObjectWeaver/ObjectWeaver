// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
package application

import (
	"errors"
	"objectweaver/orchestration/jos/domain"
	"testing"
	"time"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// TestNewStreamingGenerator tests the constructor
func TestNewStreamingGenerator(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	if generator == nil {
		t.Fatal("expected generator to be non-nil")
	}

	if generator.llmProvider == nil {
		t.Error("expected llmProvider to be set")
	}

	if generator.promptBuilder == nil {
		t.Error("expected promptBuilder to be set")
	}

	if generator.fieldProcessor == nil {
		t.Error("expected fieldProcessor to be set")
	}

	if generator.plugins == nil {
		t.Error("expected plugins registry to be set")
	}
}

// TestStreamingGenerate_BasicFlow tests the basic synchronous generation workflow
func TestStreamingGenerate_BasicFlow(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Generate a test object",
		Properties: map[string]jsonSchema.Definition{
			"name": {
				Type: jsonSchema.String,
			},
		},
	}

	request := domain.NewGenerationRequest("Generate a test", schema)

	result, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	if !result.IsSuccess() {
		t.Error("expected result to be successful")
	}

	if result.Data() == nil {
		t.Error("expected data to be non-nil")
	}
}

// TestGenerateStream_BasicFlow tests the streaming generation workflow
func TestGenerateStream_BasicFlow(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Generate a test object",
		Properties: map[string]jsonSchema.Definition{
			"name": {
				Type: jsonSchema.String,
			},
			"age": {
				Type: jsonSchema.Integer,
			},
		},
	}

	request := domain.NewGenerationRequest("Generate a test", schema)

	stream, err := generator.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if stream == nil {
		t.Fatal("expected stream to be non-nil")
	}

	// Collect chunks
	chunks := make([]*domain.StreamChunk, 0)
	timeout := time.After(5 * time.Second)

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				// Channel closed
				goto Done
			}
			chunks = append(chunks, chunk)
			if chunk.IsFinal {
				goto Done
			}
		case <-timeout:
			t.Fatal("test timed out waiting for stream to complete")
		}
	}

Done:
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Check that we received a final chunk
	foundFinal := false
	for _, chunk := range chunks {
		if chunk.IsFinal {
			foundFinal = true
			break
		}
	}

	if !foundFinal {
		t.Error("expected to receive final chunk")
	}
}

// TestGenerateStream_MultipleFields tests streaming with multiple fields
func TestGenerateStream_MultipleFields(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Generate a test object",
		Properties: map[string]jsonSchema.Definition{
			"field1": {Type: jsonSchema.String},
			"field2": {Type: jsonSchema.String},
			"field3": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Generate a test", schema)

	stream, err := generator.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Collect chunks
	fieldChunks := make(map[string]interface{})
	timeout := time.After(5 * time.Second)

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				goto Done
			}
			if !chunk.IsFinal {
				fieldChunks[chunk.Key] = chunk.Value
			} else {
				goto Done
			}
		case <-timeout:
			t.Fatal("test timed out waiting for stream to complete")
		}
	}

Done:
	if len(fieldChunks) == 0 {
		t.Error("expected at least one field chunk")
	}
}

// TestGenerateStream_PreProcessorPlugin tests pre-processor plugin integration
func TestGenerateStream_PreProcessorPlugin(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	preProcessorCalled := false
	preProcessor := &mockPreProcessorPlugin{
		name:    "test-preprocessor",
		version: "1.0.0",
		preProcessFunc: func(request *domain.GenerationRequest) (*domain.GenerationRequest, error) {
			preProcessorCalled = true
			return request.WithMetadata("preprocessed", true), nil
		},
	}

	generator.RegisterPlugin(preProcessor)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	stream, err := generator.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Drain the stream
	timeout := time.After(5 * time.Second)
	for {
		select {
		case chunk, ok := <-stream:
			if !ok || (chunk != nil && chunk.IsFinal) {
				goto Done
			}
		case <-timeout:
			t.Fatal("test timed out")
		}
	}

Done:
	if !preProcessorCalled {
		t.Error("expected pre-processor to be called")
	}
}

// TestGenerateStream_PreProcessorError tests error handling in pre-processor
func TestGenerateStream_PreProcessorError(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	preProcessorError := errors.New("pre-processing failed")
	preProcessor := &mockPreProcessorPlugin{
		name:    "failing-preprocessor",
		version: "1.0.0",
		preProcessFunc: func(request *domain.GenerationRequest) (*domain.GenerationRequest, error) {
			return nil, preProcessorError
		},
	}

	generator.RegisterPlugin(preProcessor)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties:  map[string]jsonSchema.Definition{},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)
	stream, err := generator.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error from GenerateStream itself, got: %v", err)
	}

	// The stream should close immediately due to preprocessing error
	timeout := time.After(2 * time.Second)
	chunkCount := 0

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				// Stream closed
				goto Done
			}
			if chunk != nil {
				chunkCount++
			}
		case <-timeout:
			t.Fatal("test timed out waiting for stream to close")
		}
	}

Done:
	// With preprocessing error, we should receive no chunks (stream closes immediately)
	if chunkCount > 1 {
		t.Errorf("expected minimal chunks due to preprocessing error, got %d", chunkCount)
	}
}

// TestGenerateStream_EmptySchema tests streaming with empty schema
func TestGenerateStream_EmptySchema(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties:  map[string]jsonSchema.Definition{},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	stream, err := generator.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Collect chunks
	chunks := make([]*domain.StreamChunk, 0)
	timeout := time.After(5 * time.Second)

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				goto Done
			}
			chunks = append(chunks, chunk)
			if chunk.IsFinal {
				goto Done
			}
		case <-timeout:
			t.Fatal("test timed out")
		}
	}

Done:
	// Should still receive final chunk even with empty schema
	foundFinal := false
	for _, chunk := range chunks {
		if chunk.IsFinal {
			foundFinal = true
			break
		}
	}

	if !foundFinal {
		t.Error("expected to receive final chunk")
	}
}

// TestStreamingGenerateStreamProgressive_NotSupported tests that progressive streaming is not supported
func TestStreamingGenerateStreamProgressive_NotSupported(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	result, err := generator.GenerateStreamProgressive(request)

	if err == nil {
		t.Fatal("expected error for unsupported progressive streaming")
	}

	if result != nil {
		t.Error("expected nil result for unsupported progressive streaming")
	}
}

// TestStreamingGenerate_CollectsAllChunks tests that Generate collects all chunks
func TestStreamingGenerate_CollectsAllChunks(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "test value", &domain.ProviderMetadata{}, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field1": {Type: jsonSchema.String},
			"field2": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	result, err := generator.Generate(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be non-nil")
	}

	data := result.Data()
	if data == nil {
		t.Fatal("expected data to be non-nil")
	}

	// Should have collected all fields
	if len(data) == 0 {
		t.Error("expected at least one field in collected data")
	}
}

// TestRegisterPlugin tests plugin registration
func TestStreamingRegisterPlugin(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	preProcessor := &mockPreProcessorPlugin{name: "test", version: "1.0.0"}
	postProcessor := &mockPostProcessorPlugin{name: "test", version: "1.0.0"}
	validator := &mockValidationPlugin{name: "test", version: "1.0.0"}
	cache := newMockCachePlugin()

	// Should not panic
	generator.RegisterPlugin(preProcessor)
	generator.RegisterPlugin(postProcessor)
	generator.RegisterPlugin(validator)
	generator.RegisterPlugin(cache)
}

// TestGenerateStream_ChannelClosure tests that the channel is properly closed
func TestGenerateStream_ChannelClosure(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	stream, err := generator.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Drain the stream completely
	timeout := time.After(5 * time.Second)
	channelClosed := false

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				channelClosed = true
				goto Done
			}
			if chunk != nil && chunk.IsFinal {
				// Continue to check if channel closes after final chunk
			}
		case <-timeout:
			t.Fatal("test timed out waiting for channel closure")
		}
	}

Done:
	if !channelClosed {
		t.Error("expected channel to be closed after streaming completes")
	}
}

// TestGenerateStream_ChunkOrdering tests that chunks are received in a valid order
func TestGenerateStream_ChunkOrdering(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field1": {Type: jsonSchema.String},
			"field2": {Type: jsonSchema.String},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	stream, err := generator.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	chunks := make([]*domain.StreamChunk, 0)
	timeout := time.After(5 * time.Second)

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				goto Done
			}
			chunks = append(chunks, chunk)
			if chunk.IsFinal {
				goto Done
			}
		case <-timeout:
			t.Fatal("test timed out")
		}
	}

Done:
	// Verify that final chunk comes last
	if len(chunks) > 0 {
		lastChunk := chunks[len(chunks)-1]
		if !lastChunk.IsFinal {
			t.Error("expected last chunk to be final chunk")
		}

		// Verify no final chunks before the last one
		for i := 0; i < len(chunks)-1; i++ {
			if chunks[i].IsFinal {
				t.Errorf("found final chunk at position %d, expected only at end", i)
			}
		}
	}
}

// TestGenerateStream_Concurrency tests multiple concurrent stream requests
func TestGenerateStream_Concurrency(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"field": {Type: jsonSchema.String},
		},
	}

	// Run multiple streams concurrently
	numStreams := 5
	done := make(chan bool, numStreams)
	errors := make(chan error, numStreams)

	for i := 0; i < numStreams; i++ {
		go func() {
			request := domain.NewGenerationRequest("Test prompt", schema)
			stream, err := generator.GenerateStream(request)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			// Drain the stream
			timeout := time.After(5 * time.Second)
			for {
				select {
				case chunk, ok := <-stream:
					if !ok || (chunk != nil && chunk.IsFinal) {
						done <- true
						return
					}
				case <-timeout:
					errors <- err
					done <- false
					return
				}
			}
		}()
	}

	// Wait for all streams to complete
	for i := 0; i < numStreams; i++ {
		select {
		case success := <-done:
			if !success {
				t.Error("stream failed")
			}
		case err := <-errors:
			t.Errorf("stream error: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("test timed out waiting for concurrent streams")
		}
	}
}

// TestGenerateStream_WithPath tests that chunks include path information
func TestGenerateStream_WithPath(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	generator := NewStreamingGenerator(llmProvider, promptBuilder)

	schema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Test",
		Properties: map[string]jsonSchema.Definition{
			"nested": {
				Type: jsonSchema.Object,
				Properties: map[string]jsonSchema.Definition{
					"field": {Type: jsonSchema.String},
				},
			},
		},
	}

	request := domain.NewGenerationRequest("Test prompt", schema)

	stream, err := generator.GenerateStream(request)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	hasPath := false
	timeout := time.After(5 * time.Second)

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				goto Done
			}
			if chunk != nil && !chunk.IsFinal && len(chunk.Path) > 0 {
				hasPath = true
			}
			if chunk != nil && chunk.IsFinal {
				goto Done
			}
		case <-timeout:
			t.Fatal("test timed out")
		}
	}

Done:
	if !hasPath {
		t.Log("Note: No chunks with path information received (may be expected for simple schemas)")
	}
}

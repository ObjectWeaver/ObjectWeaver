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
// <https://objectweaver.dev/licensing/server-side-public-license>.
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

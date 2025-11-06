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
	"fmt"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure/llm"
	"objectweaver/orchestration/jos/infrastructure/prompt"
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// BenchmarkProcessFields tests the concurrent processing of fields
func BenchmarkProcessFields(b *testing.B) {
	// Setup mock provider and processor
	mockProvider := llm.NewMockProvider()
	promptBuilder := prompt.NewDefaultPromptBuilder()
	processor := NewFieldProcessor(mockProvider, promptBuilder)
	processor.SetGenerator(nil) // We don't need recursive generation for this benchmark

	tests := []struct {
		name   string
		schema *jsonSchema.Definition
	}{
		{
			name: "SmallObject_5Fields",
			schema: &jsonSchema.Definition{
				Type: jsonSchema.Object,
				Properties: map[string]jsonSchema.Definition{
					"field1": {Type: jsonSchema.String},
					"field2": {Type: jsonSchema.Integer},
					"field3": {Type: jsonSchema.Boolean},
					"field4": {Type: jsonSchema.String},
					"field5": {Type: jsonSchema.Number},
				},
			},
		},
		{
			name:   "MediumObject_20Fields",
			schema: createSchemaWithFields(20),
		},
		{
			name:   "LargeObject_50Fields",
			schema: createSchemaWithFields(50),
		},
		{
			name:   "VeryLargeObject_100Fields",
			schema: createSchemaWithFields(100),
		},
		{
			name: "MixedTypes_WithDependencies",
			schema: &jsonSchema.Definition{
				Type: jsonSchema.Object,
				Properties: map[string]jsonSchema.Definition{
					"name":     {Type: jsonSchema.String},
					"age":      {Type: jsonSchema.Integer},
					"email":    {Type: jsonSchema.String},
					"active":   {Type: jsonSchema.Boolean},
					"score":    {Type: jsonSchema.Number},
					"tags":     {Type: jsonSchema.Array},
					"metadata": {Type: jsonSchema.Object},
					"status":   {Type: jsonSchema.String},
				},
				ProcessingOrder: []string{"name", "age"}, // First two must be sequential
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			request := domain.NewGenerationRequest("test prompt", tt.schema)
			context := domain.NewExecutionContext(request)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Process fields and collect results
				resultsCh := processor.ProcessFields(tt.schema, nil, context)
				count := 0
				for range resultsCh {
					count++
				}

				// Ensure we got results
				if count == 0 && len(tt.schema.Properties) > 0 {
					b.Fatal("Expected results but got none")
				}
			}
		})
	}
}

// BenchmarkProcessConcurrentFields specifically tests parallel processing
func BenchmarkProcessConcurrentFields(b *testing.B) {
	mockProvider := llm.NewMockProvider()
	promptBuilder := prompt.NewDefaultPromptBuilder()
	processor := NewFieldProcessor(mockProvider, promptBuilder)
	processor.SetGenerator(nil)

	concurrencyLevels := []int{1, 5, 10, 25, 50, 100}

	for _, numFields := range concurrencyLevels {
		b.Run(fmt.Sprintf("Fields_%d", numFields), func(b *testing.B) {
			schema := createSchemaWithFields(numFields)
			request := domain.NewGenerationRequest("test", schema)
			context := domain.NewExecutionContext(request)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				resultsCh := processor.ProcessFields(schema, nil, context)
				for range resultsCh {
					// Consume results
				}
			}
		})
	}
}

// BenchmarkSequentialVsParallel compares sequential and parallel processing
func BenchmarkSequentialVsParallel(b *testing.B) {
	mockProvider := llm.NewMockProvider()
	promptBuilder := prompt.NewDefaultPromptBuilder()
	processor := NewFieldProcessor(mockProvider, promptBuilder)
	processor.SetGenerator(nil)

	numFields := 50

	b.Run("AllSequential", func(b *testing.B) {
		// Force sequential by putting all fields in ProcessingOrder
		schema := createSchemaWithFields(numFields)
		orderedKeys := make([]string, 0, numFields)
		for key := range schema.Properties {
			orderedKeys = append(orderedKeys, key)
		}
		schema.ProcessingOrder = orderedKeys

		request := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(request)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resultsCh := processor.ProcessFields(schema, nil, context)
			for range resultsCh {
			}
		}
	})

	b.Run("AllParallel", func(b *testing.B) {
		// All fields can be parallel (no ProcessingOrder)
		schema := createSchemaWithFields(numFields)
		schema.ProcessingOrder = nil

		request := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(request)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resultsCh := processor.ProcessFields(schema, nil, context)
			for range resultsCh {
			}
		}
	})

	b.Run("Mixed_10Sequential_40Parallel", func(b *testing.B) {
		schema := createSchemaWithFields(numFields)
		orderedKeys := make([]string, 0, 10)
		count := 0
		for key := range schema.Properties {
			if count < 10 {
				orderedKeys = append(orderedKeys, key)
				count++
			}
		}
		schema.ProcessingOrder = orderedKeys

		request := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(request)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resultsCh := processor.ProcessFields(schema, nil, context)
			for range resultsCh {
			}
		}
	})
}

// BenchmarkNestedObjectProcessing tests nested object field processing
func BenchmarkNestedObjectProcessing(b *testing.B) {
	mockProvider := llm.NewMockProvider()
	promptBuilder := prompt.NewDefaultPromptBuilder()
	processor := NewFieldProcessor(mockProvider, promptBuilder)
	processor.SetGenerator(nil)

	b.Run("SingleLevel", func(b *testing.B) {
		schema := createSchemaWithFields(10)
		request := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(request)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resultsCh := processor.ProcessFields(schema, nil, context)
			for range resultsCh {
			}
		}
	})

	b.Run("TwoLevels", func(b *testing.B) {
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"nested1": {
					Type:       jsonSchema.Object,
					Properties: createSchemaWithFields(5).Properties,
				},
				"nested2": {
					Type:       jsonSchema.Object,
					Properties: createSchemaWithFields(5).Properties,
				},
			},
		}
		request := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(request)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resultsCh := processor.ProcessFields(schema, nil, context)
			for range resultsCh {
			}
		}
	})
}

// Helper function to create a schema with N fields
func createSchemaWithFields(n int) *jsonSchema.Definition {
	schema := &jsonSchema.Definition{
		Type:       jsonSchema.Object,
		Properties: make(map[string]jsonSchema.Definition),
	}

	types := []jsonSchema.DataType{
		jsonSchema.String,
		jsonSchema.Integer,
		jsonSchema.Number,
		jsonSchema.Boolean,
	}

	for i := 0; i < n; i++ {
		fieldName := fmt.Sprintf("field%d", i)
		schema.Properties[fieldName] = jsonSchema.Definition{
			Type: types[i%len(types)],
		}
	}

	return schema
}

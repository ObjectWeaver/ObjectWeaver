package execution

import (
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

// TestAnalyzeFieldDependencies_WithSelectFields tests that SelectFields dependencies are respected
func TestAnalyzeFieldDependencies_WithSelectFields(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	fp := NewFieldProcessor(llmProvider, promptBuilder)

	t.Run("SimpleSelectFieldsDependency", func(t *testing.T) {
		// Schema where "summary" depends on "content"
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"content": {Type: jsonSchema.String, Instruction: "Generate content"},
				"summary": {Type: jsonSchema.String, Instruction: "Summarize", SelectFields: []string{"content"}},
			},
		}

		orderedKeys := []string{"content", "summary"}
		batches := fp.analyzeFieldDependencies(orderedKeys, schema)

		// Should be 2 batches: content first, then summary
		if len(batches) != 2 {
			t.Errorf("Expected 2 batches, got %d", len(batches))
		}

		// First batch should contain only "content"
		if len(batches[0]) != 1 || batches[0][0] != "content" {
			t.Errorf("Expected first batch to be ['content'], got %v", batches[0])
		}

		// Second batch should contain only "summary"
		if len(batches[1]) != 1 || batches[1][0] != "summary" {
			t.Errorf("Expected second batch to be ['summary'], got %v", batches[1])
		}
	})

	t.Run("NestedPathDependency", func(t *testing.T) {
		// Schema where "analysis" depends on "car.color" (nested path)
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"car":      {Type: jsonSchema.Object, Properties: map[string]jsonSchema.Definition{"color": {Type: jsonSchema.String}}},
				"analysis": {Type: jsonSchema.String, Instruction: "Analyze", SelectFields: []string{"car.color"}},
			},
		}

		orderedKeys := []string{"car", "analysis"}
		batches := fp.analyzeFieldDependencies(orderedKeys, schema)

		// Car must be processed before analysis
		if len(batches) < 2 {
			t.Errorf("Expected at least 2 batches, got %d", len(batches))
		}

		// Find which batch contains "car" and which contains "analysis"
		carBatch := -1
		analysisBatch := -1
		for i, batch := range batches {
			for _, key := range batch {
				if key == "car" {
					carBatch = i
				}
				if key == "analysis" {
					analysisBatch = i
				}
			}
		}

		if carBatch == -1 || analysisBatch == -1 {
			t.Error("Expected both 'car' and 'analysis' to be in batches")
		}

		if carBatch >= analysisBatch {
			t.Errorf("Expected 'car' (batch %d) to be processed before 'analysis' (batch %d)", carBatch, analysisBatch)
		}
	})

	t.Run("MultipleIndependentFields", func(t *testing.T) {
		// Fields without SelectFields can be batched together
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"name":  {Type: jsonSchema.String, Instruction: "Generate name"},
				"age":   {Type: jsonSchema.Number, Instruction: "Generate age"},
				"email": {Type: jsonSchema.String, Instruction: "Generate email"},
			},
		}

		orderedKeys := []string{"name", "age", "email"}
		batches := fp.analyzeFieldDependencies(orderedKeys, schema)

		// All three can be in a single batch since they're independent
		if len(batches) != 1 {
			t.Errorf("Expected 1 batch for independent fields, got %d", len(batches))
		}

		if len(batches[0]) != 3 {
			t.Errorf("Expected all 3 fields in one batch, got %d fields", len(batches[0]))
		}
	})

	t.Run("MixedDependentAndIndependent", func(t *testing.T) {
		// Some fields are independent, some depend on others
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"name":    {Type: jsonSchema.String, Instruction: "Generate name"},
				"age":     {Type: jsonSchema.Number, Instruction: "Generate age"},
				"summary": {Type: jsonSchema.String, Instruction: "Summarize", SelectFields: []string{"name", "age"}},
			},
		}

		orderedKeys := []string{"name", "age", "summary"}
		batches := fp.analyzeFieldDependencies(orderedKeys, schema)

		// Should be 2 batches: [name, age] then [summary]
		if len(batches) != 2 {
			t.Errorf("Expected 2 batches, got %d", len(batches))
		}

		// First batch should contain name and age
		if len(batches[0]) != 2 {
			t.Errorf("Expected 2 fields in first batch, got %d", len(batches[0]))
		}

		// Second batch should contain only summary
		if len(batches[1]) != 1 || batches[1][0] != "summary" {
			t.Errorf("Expected second batch to be ['summary'], got %v", batches[1])
		}
	})

	t.Run("ArrayFieldDependency", func(t *testing.T) {
		// Field depends on array field using path notation
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"cars":        {Type: jsonSchema.Array},
				"colorReport": {Type: jsonSchema.String, Instruction: "Report colors", SelectFields: []string{"cars.color"}},
			},
		}

		orderedKeys := []string{"cars", "colorReport"}
		batches := fp.analyzeFieldDependencies(orderedKeys, schema)

		// Should be 2 batches
		if len(batches) != 2 {
			t.Errorf("Expected 2 batches, got %d", len(batches))
		}

		// Cars must come before colorReport
		if batches[0][0] != "cars" {
			t.Errorf("Expected first batch to contain 'cars', got %v", batches[0])
		}

		if batches[1][0] != "colorReport" {
			t.Errorf("Expected second batch to contain 'colorReport', got %v", batches[1])
		}
	})

	t.Run("ChainedDependencies", func(t *testing.T) {
		// A -> B -> C dependency chain
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"base":    {Type: jsonSchema.String, Instruction: "Generate base"},
				"derived": {Type: jsonSchema.String, Instruction: "Derive", SelectFields: []string{"base"}},
				"final":   {Type: jsonSchema.String, Instruction: "Finalize", SelectFields: []string{"derived"}},
			},
		}

		orderedKeys := []string{"base", "derived", "final"}
		batches := fp.analyzeFieldDependencies(orderedKeys, schema)

		// Should be 3 batches, one for each field
		if len(batches) != 3 {
			t.Errorf("Expected 3 batches for chained dependencies, got %d", len(batches))
		}

		// Verify order
		if batches[0][0] != "base" || batches[1][0] != "derived" || batches[2][0] != "final" {
			t.Errorf("Expected order [base, derived, final], got %v", batches)
		}
	})

	t.Run("WrongOrderInProcessingOrder", func(t *testing.T) {
		// If ProcessingOrder specifies wrong order, our function should still respect dependencies
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"base":    {Type: jsonSchema.String, Instruction: "Generate base"},
				"summary": {Type: jsonSchema.String, Instruction: "Summarize", SelectFields: []string{"base"}},
			},
		}

		// Wrong order: summary before base
		orderedKeys := []string{"summary", "base"}
		batches := fp.analyzeFieldDependencies(orderedKeys, schema)

		// Should still put base before summary
		if len(batches) != 2 {
			t.Errorf("Expected 2 batches, got %d", len(batches))
		}

		// Base should come first despite wrong input order
		foundBase := false
		foundSummary := false
		for i, batch := range batches {
			for _, key := range batch {
				if key == "base" {
					foundBase = true
					if foundSummary {
						t.Error("Found 'summary' before 'base', dependencies not respected")
					}
				}
				if key == "summary" {
					foundSummary = true
					if !foundBase {
						t.Errorf("Found 'summary' in batch %d before 'base', dependencies not respected", i)
					}
				}
			}
		}
	})

	t.Run("SelfReference", func(t *testing.T) {
		// Field should not depend on itself
		schema := &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"field": {Type: jsonSchema.String, Instruction: "Generate", SelectFields: []string{"field"}},
			},
		}

		orderedKeys := []string{"field"}
		batches := fp.analyzeFieldDependencies(orderedKeys, schema)

		// Should process normally (self-reference ignored)
		if len(batches) != 1 || len(batches[0]) != 1 {
			t.Error("Self-reference should be ignored")
		}
	})
}

// TestExtractRootFieldName tests the helper function
func TestExtractRootFieldName(t *testing.T) {
	tests := []struct {
		name      string
		fieldPath string
		expected  string
	}{
		{"Simple field", "name", "name"},
		{"Nested field", "car.color", "car"},
		{"Deeply nested", "user.profile.avatar.url", "user"},
		{"Empty string", "", ""},
		{"Single dot", ".", ""},
		{"Trailing dot", "field.", "field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRootFieldName(tt.fieldPath)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

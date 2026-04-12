package execution

import (
	"context"
	"objectweaver/orchestration/jos/domain"
	"testing"

	"objectweaver/jsonSchema"
)

// TestSelectFields_NestedObjectAccess tests that SelectFields can access nested object fields
func TestSelectFields_NestedObjectAccess(t *testing.T) {
	// Mock LLM provider that captures the prompt
	var capturedPrompt string
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			capturedPrompt = prompt
			return "summary based on nested data", &domain.ProviderMetadata{}, nil
		},
	}

	processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})

	// Set up context with nested object
	ctx := context.Background()
	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(request)

	// Add nested object to context
	execContext.SetGeneratedValue("car", map[string]interface{}{
		"color": "red",
		"brand": "Toyota",
		"specs": map[string]interface{}{
			"engine": "V6",
			"hp":     280,
		},
	})

	// Create task with SelectFields pointing to nested fields
	taskDef := &jsonSchema.Definition{
		Type:         jsonSchema.String,
		Instruction:  "Summarize",
		SelectFields: []string{"car.color", "car.specs.engine"},
	}
	task := domain.NewFieldTask("summary", taskDef, nil)

	// Process the task
	_, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify the prompt includes the nested values
	if capturedPrompt == "" {
		t.Fatal("Expected prompt to be captured")
	}

	// Check for the nested color value
	if !containsString(capturedPrompt, "red") {
		t.Errorf("Expected prompt to contain 'red' (car.color), got: %s", capturedPrompt)
	}

	// Check for the deeply nested engine value
	if !containsString(capturedPrompt, "V6") {
		t.Errorf("Expected prompt to contain 'V6' (car.specs.engine), got: %s", capturedPrompt)
	}
}

// TestSelectFields_ArrayFieldExtraction tests that SelectFields can extract fields from arrays
func TestSelectFields_ArrayFieldExtraction(t *testing.T) {
	// Mock LLM provider that captures the prompt
	var capturedPrompt string
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			capturedPrompt = prompt
			return "summary of all colors", &domain.ProviderMetadata{}, nil
		},
	}

	processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})

	// Set up context with array
	ctx := context.Background()
	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(request)

	// Add array of objects to context
	execContext.SetGeneratedValue("cars", []interface{}{
		map[string]interface{}{"color": "red", "brand": "Toyota"},
		map[string]interface{}{"color": "blue", "brand": "Honda"},
		map[string]interface{}{"color": "green", "brand": "Ford"},
	})

	// Create task with SelectFields pointing to array field
	taskDef := &jsonSchema.Definition{
		Type:         jsonSchema.String,
		Instruction:  "List all colors",
		SelectFields: []string{"cars.color"},
	}
	task := domain.NewFieldTask("colorSummary", taskDef, nil)

	// Process the task
	_, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify the prompt includes all colors
	if capturedPrompt == "" {
		t.Fatal("Expected prompt to be captured")
	}

	// Check for all color values
	expectedColors := []string{"red", "blue", "green"}
	for _, color := range expectedColors {
		if !containsString(capturedPrompt, color) {
			t.Errorf("Expected prompt to contain '%s', got: %s", color, capturedPrompt)
		}
	}
}

// TestSelectFields_MixedPaths tests using both simple and nested paths together
func TestSelectFields_MixedPaths(t *testing.T) {
	// Mock LLM provider that captures the prompt
	var capturedPrompt string
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			capturedPrompt = prompt
			return "comprehensive summary", &domain.ProviderMetadata{}, nil
		},
	}

	processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})

	// Set up context with mixed data
	ctx := context.Background()
	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(request)

	// Add various types of data
	execContext.SetGeneratedValue("title", "Product Review")
	execContext.SetGeneratedValue("product", map[string]interface{}{
		"name":  "Widget",
		"price": 99.99,
	})
	execContext.SetGeneratedValue("reviews", []interface{}{
		map[string]interface{}{"rating": 5, "comment": "Great!"},
		map[string]interface{}{"rating": 4, "comment": "Good"},
	})

	// Create task with SelectFields using mixed paths
	taskDef := &jsonSchema.Definition{
		Type:         jsonSchema.String,
		Instruction:  "Create summary",
		SelectFields: []string{"title", "product.name", "reviews.rating"},
	}
	task := domain.NewFieldTask("summary", taskDef, nil)

	// Process the task
	_, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify the prompt includes all selected values
	if capturedPrompt == "" {
		t.Fatal("Expected prompt to be captured")
	}

	// Check for simple field
	if !containsString(capturedPrompt, "Product Review") {
		t.Errorf("Expected prompt to contain 'Product Review', got: %s", capturedPrompt)
	}

	// Check for nested field
	if !containsString(capturedPrompt, "Widget") {
		t.Errorf("Expected prompt to contain 'Widget', got: %s", capturedPrompt)
	}

	// Check for array field extraction (should have both ratings)
	if !containsString(capturedPrompt, "5") || !containsString(capturedPrompt, "4") {
		t.Errorf("Expected prompt to contain ratings '5' and '4', got: %s", capturedPrompt)
	}
}

// TestSelectFields_NonExistentPath tests that non-existent paths are handled gracefully
func TestSelectFields_NonExistentPath(t *testing.T) {
	// Mock LLM provider that captures the prompt
	var capturedPrompt string
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			capturedPrompt = prompt
			return "fallback response", &domain.ProviderMetadata{}, nil
		},
	}

	processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})

	// Set up context with limited data
	ctx := context.Background()
	fieldDef := &jsonSchema.Definition{Type: jsonSchema.String}
	request := domain.NewGenerationRequest("test", fieldDef)
	execContext := domain.NewExecutionContext(request)

	execContext.SetGeneratedValue("car", map[string]interface{}{
		"color": "red",
	})

	// Create task with SelectFields including non-existent path
	taskDef := &jsonSchema.Definition{
		Type:         jsonSchema.String,
		Instruction:  "Describe",
		SelectFields: []string{"car.color", "car.missing.field", "nonexistent"},
	}
	task := domain.NewFieldTask("description", taskDef, nil)

	// Process should not fail due to missing fields
	_, err := processor.Process(ctx, task, execContext)
	if err != nil {
		t.Fatalf("Process should not fail with missing SelectFields: %v", err)
	}

	// Verify existing field was included
	if !containsString(capturedPrompt, "red") {
		t.Errorf("Expected prompt to contain existing field 'red', got: %s", capturedPrompt)
	}
}

// Helper function to check if a string contains a substring (case-insensitive search could be added)
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

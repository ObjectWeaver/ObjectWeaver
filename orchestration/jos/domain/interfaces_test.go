package domain

import (
	"reflect"
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

func TestNewExecutionContext(t *testing.T) {
	req := NewGenerationRequest("test prompt", &jsonSchema.Definition{Type: jsonSchema.String})
	ctx := NewExecutionContext(req)

	if ctx.request != req {
		t.Errorf("Expected request to be set, got %v", ctx.request)
	}
	if ctx.generatedValues == nil {
		t.Error("Expected generatedValues to be initialized")
	}
	if ctx.metadata == nil {
		t.Error("Expected metadata to be initialized")
	}
	if ctx.promptContext == nil {
		t.Error("Expected promptContext to be initialized")
	}
	if ctx.generationConfig == nil {
		t.Error("Expected generationConfig to be initialized")
	}
}

func TestExecutionContext_WithParent(t *testing.T) {
	req := NewGenerationRequest("test prompt", &jsonSchema.Definition{Type: jsonSchema.String})
	parentCtx := NewExecutionContext(req)
	parentCtx.SetGeneratedValue("key1", "value1")
	parentCtx.metadata["meta1"] = "metaValue1"

	parentTask := NewFieldTask("parent", &jsonSchema.Definition{Type: jsonSchema.String}, nil)
	childCtx := parentCtx.WithParent(parentTask)

	if childCtx.request != req {
		t.Errorf("Expected request to be the same")
	}
	if childCtx.parentContext != parentCtx {
		t.Errorf("Expected parentContext to be set")
	}
	if childCtx.generatedValues["key1"] != "value1" {
		t.Errorf("Expected generatedValues to be copied")
	}
	if childCtx.metadata["meta1"] != "metaValue1" {
		t.Errorf("Expected metadata to be copied")
	}
	if childCtx.promptContext != parentCtx.promptContext {
		t.Errorf("Expected promptContext to be shared")
	}
	if childCtx.generationConfig != parentCtx.generationConfig {
		t.Errorf("Expected generationConfig to be shared")
	}
}

func TestExecutionContext_Getters(t *testing.T) {
	req := NewGenerationRequest("test prompt", &jsonSchema.Definition{Type: jsonSchema.String})
	ctx := NewExecutionContext(req)

	if ctx.Request() != req {
		t.Errorf("Request getter failed")
	}
	if ctx.GeneratedValues() == nil {
		t.Error("GeneratedValues getter failed")
	}
	if ctx.PromptContext() == nil {
		t.Error("PromptContext getter failed")
	}
	if ctx.GenerationConfig() == nil {
		t.Error("GenerationConfig getter failed")
	}
}

func TestExecutionContext_SetGeneratedValue(t *testing.T) {
	req := NewGenerationRequest("test prompt", &jsonSchema.Definition{Type: jsonSchema.String})
	ctx := NewExecutionContext(req)

	ctx.SetGeneratedValue("key", "value")
	if ctx.generatedValues["key"] != "value" {
		t.Errorf("SetGeneratedValue failed")
	}
}

func TestExecutionContext_copyGeneratedValues(t *testing.T) {
	req := NewGenerationRequest("test prompt", &jsonSchema.Definition{Type: jsonSchema.String})
	ctx := NewExecutionContext(req)
	ctx.SetGeneratedValue("key", "value")

	copied := ctx.copyGeneratedValues()
	if !reflect.DeepEqual(ctx.generatedValues, copied) {
		t.Errorf("copyGeneratedValues failed")
	}
	if &ctx.generatedValues == &copied {
		t.Error("copyGeneratedValues should create a new map")
	}
}

func TestExecutionContext_copyMetadata(t *testing.T) {
	req := NewGenerationRequest("test prompt", &jsonSchema.Definition{Type: jsonSchema.String})
	ctx := NewExecutionContext(req)
	ctx.metadata["key"] = "value"

	copied := ctx.copyMetadata()
	if !reflect.DeepEqual(ctx.metadata, copied) {
		t.Errorf("copyMetadata failed")
	}
	if &ctx.metadata == &copied {
		t.Error("copyMetadata should create a new map")
	}
}

func TestNewPromptContext(t *testing.T) {
	pc := NewPromptContext()

	if pc.Prompts == nil {
		t.Error("Expected Prompts to be initialized")
	}
	if pc.ExistingSubLists == nil {
		t.Error("Expected ExistingSubLists to be initialized")
	}
	if pc.CurrentGen != "" {
		t.Errorf("Expected CurrentGen to be empty, got %s", pc.CurrentGen)
	}
	if pc.ParentGen != "" {
		t.Errorf("Expected ParentGen to be empty, got %s", pc.ParentGen)
	}
}

func TestPromptContext_AddPrompt(t *testing.T) {
	pc := NewPromptContext()
	pc.AddPrompt("test prompt")

	if len(pc.Prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(pc.Prompts))
	}
	if pc.Prompts[0] != "test prompt" {
		t.Errorf("Expected prompt to be 'test prompt', got %s", pc.Prompts[0])
	}
}

func TestPromptContext_FirstPrompt(t *testing.T) {
	pc := NewPromptContext()

	// Empty
	if pc.FirstPrompt() != "" {
		t.Errorf("Expected empty string, got %s", pc.FirstPrompt())
	}

	pc.AddPrompt("first")
	pc.AddPrompt("second")

	if pc.FirstPrompt() != "first" {
		t.Errorf("Expected 'first', got %s", pc.FirstPrompt())
	}
}

func TestDefaultGenerationConfig(t *testing.T) {
	config := DefaultGenerationConfig()

	if config.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", config.Temperature)
	}
	if config.MaxTokens != 2000 {
		t.Errorf("Expected MaxTokens 2000, got %d", config.MaxTokens)
	}
	if config.Granularity != GranularityField {
		t.Errorf("Expected Granularity GranularityField, got %v", config.Granularity)
	}
	if config.BufferSize != 10 {
		t.Errorf("Expected BufferSize 10, got %d", config.BufferSize)
	}
	if len(config.StopSequences) != 0 {
		t.Errorf("Expected empty StopSequences, got %v", config.StopSequences)
	}
	if config.SystemPrompt != "" {
		t.Errorf("Expected empty SystemPrompt, got %s", config.SystemPrompt)
	}
	if config.Model != "" {
		t.Errorf("Expected empty Model, got %v", config.Model)
	}
	if config.Definition != nil {
		t.Errorf("Expected nil Definition, got %v", config.Definition)
	}
}

// Additional tests for other structs if needed, but since they are mostly interfaces, this covers the concrete ones.

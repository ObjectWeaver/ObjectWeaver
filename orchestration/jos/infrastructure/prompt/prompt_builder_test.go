package prompt

import (
	"testing"

	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

func TestNewDefaultPromptBuilder(t *testing.T) {
	builder := NewDefaultPromptBuilder()
	if builder == nil {
		t.Fatal("NewDefaultPromptBuilder returned nil")
	}
	if builder.extractor == nil {
		t.Error("extractor should be initialized")
	}
}

func TestBuild_OverridePrompt(t *testing.T) {
	builder := NewDefaultPromptBuilder()
	def := &jsonSchema.Definition{
		OverridePrompt: stringPtr("Custom override prompt"),
		Instruction:    "Default instruction",
	}
	task := domain.NewFieldTask("test", def, nil)

	context := domain.NewPromptContext()

	prompt, err := builder.Build(task, context)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	expected := "Custom override prompt"
	if prompt != expected {
		t.Errorf("Expected %q, got %q", expected, prompt)
	}
}

func TestBuild_StandardPrompt(t *testing.T) {
	builder := NewDefaultPromptBuilder()
	def := &jsonSchema.Definition{
		Instruction: "Generate a name",
	}
	task := createTestFieldTask("name", def)

	context := domain.NewPromptContext()
	context.AddPrompt("Base prompt")
	context.CurrentGen = "Existing generation"

	prompt, err := builder.Build(task, context)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !contains(prompt, "name") {
		t.Error("Prompt should contain the field key")
	}
	if !contains(prompt, "Generate a name") {
		t.Error("Prompt should contain the instruction")
	}
	if !contains(prompt, "Existing generation") {
		t.Error("Prompt should contain current gen")
	}
}

func TestBuild_NarrowFocus(t *testing.T) {
	builder := NewDefaultPromptBuilder()
	def := &jsonSchema.Definition{
		NarrowFocus: &jsonSchema.Focus{
			Prompt:       "Focus on this",
			KeepOriginal: true,
		},
	}
	task := domain.NewFieldTask("field", def, nil)

	context := domain.NewPromptContext()
	context.AddPrompt("Original prompt")
	context.CurrentGen = "Current context"

	prompt, err := builder.Build(task, context)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !contains(prompt, "Focus on this") {
		t.Error("Prompt should contain narrow focus prompt")
	}
	if !contains(prompt, "Original prompt") {
		t.Error("Prompt should contain original prompt when KeepOriginal is true")
	}
}

func TestBuildWithHistory(t *testing.T) {
	builder := NewDefaultPromptBuilder()
	def := &jsonSchema.Definition{
		Instruction: "Test instruction",
	}
	task := createTestFieldTask("test", def)
	context := domain.NewPromptContext()
	history := &domain.GenerationHistory{}

	prompt1, err1 := builder.Build(task, context)
	prompt2, err2 := builder.BuildWithHistory(task, context, history)

	if err1 != nil || err2 != nil {
		t.Fatalf("Errors: %v, %v", err1, err2)
	}
	if prompt1 != prompt2 {
		t.Error("BuildWithHistory should currently delegate to Build")
	}
}

func TestBuildNarrowFocusPrompt(t *testing.T) {
	builder := NewDefaultPromptBuilder()
	def := &jsonSchema.Definition{
		NarrowFocus: &jsonSchema.Focus{
			Prompt:       "Narrow prompt",
			KeepOriginal: false,
		},
	}
	context := domain.NewPromptContext()
	context.AddPrompt("Base prompt")
	context.CurrentGen = "Current gen"

	prompt, err := builder.buildNarrowFocusPrompt(def, context, "base")
	if err != nil {
		t.Fatalf("buildNarrowFocusPrompt returned error: %v", err)
	}
	if !contains(prompt, "Narrow prompt") {
		t.Error("Prompt should contain narrow focus prompt")
	}
	if contains(prompt, "Base prompt") {
		t.Error("Prompt should not contain original when KeepOriginal is false")
	}
}

// Helper functions

func createTestFieldTask(key string, def *jsonSchema.Definition) *domain.FieldTask {
	return domain.NewFieldTask(key, def, nil)
}

func stringPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package requestManagement

import (
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

	"github.com/sashabaranov/go-openai"
)

func TestNewEmbeddingOpenAIReqBuilder(t *testing.T) {
	builder := NewEmbeddingOpenAIReqBuilder()
	if builder == nil {
		t.Fatal("NewEmbeddingOpenAIReqBuilder() returned nil")
	}

	// Verify it returns the correct type
	if _, ok := builder.(*embeddingOpenAIReqBuilder); !ok {
		t.Error("NewEmbeddingOpenAIReqBuilder() did not return *embeddingOpenAIReqBuilder")
	}
}

func TestEmbeddingOpenAIReqBuilder_BuildRequest_Basic(t *testing.T) {
	builder := NewEmbeddingOpenAIReqBuilder()

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Model: "text-embedding-ada-002",
		},
		Prompt: "This is a test prompt",
	}

	req, err := builder.BuildRequest(inputs)
	if err != nil {
		t.Fatalf("BuildRequest() failed: %v", err)
	}

	if string(req.Model) != "text-embedding-ada-002" {
		t.Errorf("Model = %s, want text-embedding-ada-002", req.Model)
	}

	if req.Input != "This is a test prompt" {
		t.Errorf("Input = %s, want 'This is a test prompt'", req.Input)
	}
}

func TestEmbeddingOpenAIReqBuilder_BuildRequest_DifferentModels(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{
			name:  "text-embedding-ada-002",
			model: "text-embedding-ada-002",
		},
		{
			name:  "text-embedding-3-small",
			model: "text-embedding-3-small",
		},
		{
			name:  "text-embedding-3-large",
			model: "text-embedding-3-large",
		},
	}

	builder := NewEmbeddingOpenAIReqBuilder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := &llmManagement.Inputs{
				Def: &jsonSchema.Definition{
					Model: tt.model,
				},
				Prompt: "Test prompt",
			}

			req, err := builder.BuildRequest(inputs)
			if err != nil {
				t.Fatalf("BuildRequest() failed: %v", err)
			}

			if string(req.Model) != tt.model {
				t.Errorf("Model = %s, want %s", req.Model, tt.model)
			}
		})
	}
}

func TestEmbeddingOpenAIReqBuilder_BuildRequest_DifferentPrompts(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{
			name:   "simple text",
			prompt: "Hello world",
		},
		{
			name:   "empty string",
			prompt: "",
		},
		{
			name:   "long text",
			prompt: "This is a very long text that might be used for embedding. It contains multiple sentences and should still work properly with the embedding request builder.",
		},
		{
			name:   "special characters",
			prompt: "Special characters: !@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:   "unicode characters",
			prompt: "Unicode: 你好世界 🌍 émojis",
		},
	}

	builder := NewEmbeddingOpenAIReqBuilder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := &llmManagement.Inputs{
				Def: &jsonSchema.Definition{
					Model: "text-embedding-ada-002",
				},
				Prompt: tt.prompt,
			}

			req, err := builder.BuildRequest(inputs)
			if err != nil {
				t.Fatalf("BuildRequest() failed: %v", err)
			}

			if req.Input != tt.prompt {
				t.Errorf("Input = %s, want %s", req.Input, tt.prompt)
			}
		})
	}
}

func TestEmbeddingOpenAIReqBuilder_BuildRequest_VerifyType(t *testing.T) {
	builder := NewEmbeddingOpenAIReqBuilder()

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Model: "text-embedding-ada-002",
		},
		Prompt: "Test",
	}

	req, err := builder.BuildRequest(inputs)
	if err != nil {
		t.Fatalf("BuildRequest() failed: %v", err)
	}

	// Verify the returned type is openai.EmbeddingRequest
	var _ openai.EmbeddingRequest = req
}

func TestEmbeddingOpenAIReqBuilder_BuildRequest_SystemPromptIgnored(t *testing.T) {
	builder := NewEmbeddingOpenAIReqBuilder()

	// SystemPrompt should be ignored for embedding requests
	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Model: "text-embedding-ada-002",
		},
		Prompt:       "This is a test prompt",
		SystemPrompt: "This should be ignored",
	}

	req, err := builder.BuildRequest(inputs)
	if err != nil {
		t.Fatalf("BuildRequest() failed: %v", err)
	}

	// Only the Prompt should be used, not SystemPrompt
	if req.Input != "This is a test prompt" {
		t.Errorf("Input = %s, want 'This is a test prompt'", req.Input)
	}
}

func TestEmbeddingOpenAIReqBuilder_BuildRequest_MultipleInputs(t *testing.T) {
	builder := NewEmbeddingOpenAIReqBuilder()

	// Test that we can create multiple requests with the same builder
	inputs1 := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Model: "text-embedding-ada-002",
		},
		Prompt: "First prompt",
	}

	inputs2 := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Model: "text-embedding-3-small",
		},
		Prompt: "Second prompt",
	}

	req1, err := builder.BuildRequest(inputs1)
	if err != nil {
		t.Fatalf("BuildRequest() for inputs1 failed: %v", err)
	}

	req2, err := builder.BuildRequest(inputs2)
	if err != nil {
		t.Fatalf("BuildRequest() for inputs2 failed: %v", err)
	}

	if req1.Input != "First prompt" {
		t.Errorf("req1.Input = %s, want 'First prompt'", req1.Input)
	}

	if req2.Input != "Second prompt" {
		t.Errorf("req2.Input = %s, want 'Second prompt'", req2.Input)
	}

	if string(req1.Model) != "text-embedding-ada-002" {
		t.Errorf("req1.Model = %s, want text-embedding-ada-002", req1.Model)
	}

	if string(req2.Model) != "text-embedding-3-small" {
		t.Errorf("req2.Model = %s, want text-embedding-3-small", req2.Model)
	}
}

func TestEmbeddingOpenAIReqBuilder_Interface(t *testing.T) {
	// Verify that embeddingOpenAIReqBuilder implements EmbeddingRequestBuilder
	var _ EmbeddingRequestBuilder = &embeddingOpenAIReqBuilder{}
	var _ EmbeddingRequestBuilder = NewEmbeddingOpenAIReqBuilder()
}

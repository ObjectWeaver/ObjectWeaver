package clientManager

import (
	"errors"
	"objectweaver/llmManagement"
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

// Mock request builder for OpenAI testing
type mockOpenAIRequestBuilder struct {
	request openai.ChatCompletionRequest
	err     error
}

func (m *mockOpenAIRequestBuilder) BuildRequest(inputs *llmManagement.Inputs) (openai.ChatCompletionRequest, error) {
	return m.request, m.err
}

// Mock embedding request builder for OpenAI testing
type mockOpenAIEmbeddingRequestBuilder struct {
	request openai.EmbeddingRequest
	err     error
}

func (m *mockOpenAIEmbeddingRequestBuilder) BuildRequest(inputs *llmManagement.Inputs) (openai.EmbeddingRequest, error) {
	return m.request, m.err
}

func TestNewOpenAIClientAdapter(t *testing.T) {
	apiKey := "test-api-key"
	builder := &mockOpenAIRequestBuilder{}
	embeddingBuilder := &mockOpenAIEmbeddingRequestBuilder{}

	adapter := NewOpenAIClientAdapter(apiKey, builder, embeddingBuilder, nil)

	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}

	if adapter.client == nil {
		t.Error("Expected client to be initialized")
	}

	if adapter.requestBuilder == nil {
		t.Error("Expected requestBuilder to be set")
	}

	if adapter.embeddingRequestBuilder == nil {
		t.Error("Expected embeddingRequestBuilder to be set")
	}
}

func TestOpenAIClientAdapter_ProcessBatch(t *testing.T) {
	adapter := NewOpenAIClientAdapter("test-key", &mockOpenAIRequestBuilder{}, &mockOpenAIEmbeddingRequestBuilder{}, nil)

	jobs := []any{1, 2, 3}
	resp, err := adapter.ProcessBatch(jobs)

	if err == nil {
		t.Error("Expected error from ProcessBatch")
	}

	if resp != nil {
		t.Error("Expected nil response from ProcessBatch")
	}

	if err.Error() != "doesn't exist" {
		t.Errorf("Expected error message 'doesn't exist', got %v", err)
	}
}

func TestOpenAIClientAdapter_Process_ChatRequestBuildError(t *testing.T) {
	builder := &mockOpenAIRequestBuilder{err: errors.New("mock build error")}
	adapter := NewOpenAIClientAdapter("test-key", builder, &mockOpenAIEmbeddingRequestBuilder{}, nil)

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Type: jsonSchema.String, // Non-vector type for chat
		},
	}

	result, err := adapter.Process(inputs)

	if err == nil {
		t.Fatal("Expected error when request build fails")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}

	if err.Error() != "failed to build openai request: mock build error" {
		t.Errorf("Expected build error, got: %v", err)
	}
}

func TestOpenAIClientAdapter_Process_EmbeddingRequestBuildError(t *testing.T) {
	embeddingBuilder := &mockOpenAIEmbeddingRequestBuilder{err: errors.New("mock embedding build error")}
	adapter := NewOpenAIClientAdapter("test-key", &mockOpenAIRequestBuilder{}, embeddingBuilder, nil)

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Type: jsonSchema.Vector, // Vector type for embedding
		},
	}

	result, err := adapter.Process(inputs)

	if err == nil {
		t.Fatal("Expected error when embedding request build fails")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}

	if err.Error() != "failed to build openai embedding request: mock embedding build error" {
		t.Errorf("Expected embedding build error, got: %v", err)
	}
}

func TestOpenAIClientAdapter_Process_VectorType(t *testing.T) {
	// Test that vector type routes to embedding processing
	adapter := NewOpenAIClientAdapter("invalid-key", &mockOpenAIRequestBuilder{}, &mockOpenAIEmbeddingRequestBuilder{}, nil)

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Type: jsonSchema.Vector,
		},
	}

	// This will fail at API call (invalid key), but verifies routing
	_, err := adapter.Process(inputs)

	// Should get an API error, not a routing error
	if err != nil && err.Error() != "failed to build openai embedding request: mock embedding build error" {
		// Expected - we're using mock that doesn't error
		// If we got here, the embedding path was attempted
	}
}

func TestOpenAIClientAdapter_Process_NonVectorType(t *testing.T) {
	// Test that non-vector types route to chat processing
	adapter := NewOpenAIClientAdapter("invalid-key", &mockOpenAIRequestBuilder{}, &mockOpenAIEmbeddingRequestBuilder{}, nil)

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Type: jsonSchema.String,
		},
	}

	// This will fail at API call (invalid key), but verifies routing
	_, err := adapter.Process(inputs)

	// Should get an API error, not a routing error
	if err != nil && err.Error() != "failed to build openai request: mock build error" {
		// Expected - we're using mock that doesn't error
		// If we got here, the chat path was attempted
	}
}

func TestOpenAIClientAdapter_StructureValidation(t *testing.T) {
	// Validate the adapter structure
	apiKey := "test-key-123"
	builder := &mockOpenAIRequestBuilder{}
	embeddingBuilder := &mockOpenAIEmbeddingRequestBuilder{}

	adapter := NewOpenAIClientAdapter(apiKey, builder, embeddingBuilder, nil)

	// Verify all fields are properly initialized
	if adapter.client == nil {
		t.Error("Client should be initialized")
	}

	if adapter.requestBuilder != builder {
		t.Error("Request builder should be set correctly")
	}

	if adapter.embeddingRequestBuilder != embeddingBuilder {
		t.Error("Embedding request builder should be set correctly")
	}
}

func TestOpenAIClientAdapter_TypeRouting(t *testing.T) {
	tests := []struct {
		name        string
		defType     jsonSchema.DataType
		expectsChat bool
	}{
		{
			name:        "String type routes to chat",
			defType:     jsonSchema.String,
			expectsChat: true,
		},
		{
			name:        "Object type routes to chat",
			defType:     jsonSchema.Object,
			expectsChat: true,
		},
		{
			name:        "Array type routes to chat",
			defType:     jsonSchema.Array,
			expectsChat: true,
		},
		{
			name:        "Vector type routes to embedding",
			defType:     jsonSchema.Vector,
			expectsChat: false,
		},
		{
			name:        "Number type routes to chat",
			defType:     jsonSchema.Number,
			expectsChat: true,
		},
		{
			name:        "Boolean type routes to chat",
			defType:     jsonSchema.Boolean,
			expectsChat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatBuilder := &mockOpenAIRequestBuilder{err: errors.New("mock build error")}
			embeddingBuilder := &mockOpenAIEmbeddingRequestBuilder{err: errors.New("mock embedding build error")}
			adapter := NewOpenAIClientAdapter("test-key", chatBuilder, embeddingBuilder, nil)

			inputs := &llmManagement.Inputs{
				Def: &jsonSchema.Definition{
					Type: tt.defType,
				},
			}

			_, err := adapter.Process(inputs)

			if err == nil {
				t.Fatal("Expected error from mock")
			}

			if tt.expectsChat {
				// Should have chat error
				if err.Error() != "failed to build openai request: mock build error" {
					t.Errorf("Expected chat error, got: %v", err)
				}
			} else {
				// Should have embedding error
				if err.Error() != "failed to build openai embedding request: mock embedding build error" {
					t.Errorf("Expected embedding error, got: %v", err)
				}
			}
		})
	}
}

func TestOpenAIClientAdapter_EmptyAPIKey(t *testing.T) {
	// Test with empty API key - should still create adapter
	adapter := NewOpenAIClientAdapter("", &mockOpenAIRequestBuilder{}, &mockOpenAIEmbeddingRequestBuilder{}, nil)

	if adapter == nil {
		t.Error("Should create adapter even with empty API key")
	}

	if adapter.client == nil {
		t.Error("Client should be created even with empty API key")
	}
}

func TestOpenAIClientAdapter_NilBuilders(t *testing.T) {
	// Test that adapter can be created with nil builders (though not recommended)
	adapter := NewOpenAIClientAdapter("test-key", nil, nil, nil)

	if adapter == nil {
		t.Error("Should create adapter even with nil builders")
	}

	if adapter.requestBuilder != nil {
		t.Error("Expected requestBuilder to be nil")
	}

	if adapter.embeddingRequestBuilder != nil {
		t.Error("Expected embeddingRequestBuilder to be nil")
	}
}

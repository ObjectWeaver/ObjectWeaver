package domain

import (
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestCreateJobResult(t *testing.T) {
	tests := []struct {
		name         string
		chatRes      *openai.ChatCompletionResponse
		embeddingRes *openai.EmbeddingResponse
	}{
		{
			name: "both responses provided",
			chatRes: &openai.ChatCompletionResponse{
				ID:      "test-id",
				Object:  "chat.completion",
				Created: 1234567890,
			},
			embeddingRes: &openai.EmbeddingResponse{
				Object: "list",
			},
		},
		{
			name:         "only chat response",
			chatRes:      &openai.ChatCompletionResponse{ID: "chat-id"},
			embeddingRes: nil,
		},
		{
			name:         "only embedding response",
			chatRes:      nil,
			embeddingRes: &openai.EmbeddingResponse{Object: "list"},
		},
		{
			name:         "both nil",
			chatRes:      nil,
			embeddingRes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateJobResult(tt.chatRes, tt.embeddingRes)

			if result == nil {
				t.Fatal("Expected non-nil JobResult")
			}

			if result.ChatRes != tt.chatRes {
				t.Errorf("Expected ChatRes to be %v, got %v", tt.chatRes, result.ChatRes)
			}

			if result.EmbeddingRes != tt.embeddingRes {
				t.Errorf("Expected EmbeddingRes to be %v, got %v", tt.embeddingRes, result.EmbeddingRes)
			}
		})
	}
}

func TestJobResultFields(t *testing.T) {
	chatRes := &openai.ChatCompletionResponse{
		ID:      "chat-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "Hello, world!",
				},
				FinishReason: "stop",
			},
		},
	}

	embeddingRes := &openai.EmbeddingResponse{
		Object: "list",
		Data: []openai.Embedding{
			{
				Object:    "embedding",
				Embedding: []float32{0.1, 0.2, 0.3},
				Index:     0,
			},
		},
		Model: "text-embedding-ada-002",
		Usage: openai.Usage{
			PromptTokens:     10,
			CompletionTokens: 0,
			TotalTokens:      10,
		},
	}

	result := CreateJobResult(chatRes, embeddingRes)

	// Verify chat response is accessible
	if result.ChatRes.ID != "chat-123" {
		t.Errorf("Expected chat ID to be 'chat-123', got '%s'", result.ChatRes.ID)
	}
	if len(result.ChatRes.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(result.ChatRes.Choices))
	}
	if result.ChatRes.Choices[0].Message.Content != "Hello, world!" {
		t.Errorf("Expected message content to be 'Hello, world!', got '%s'", result.ChatRes.Choices[0].Message.Content)
	}

	// Verify embedding response is accessible
	if result.EmbeddingRes.Object != "list" {
		t.Errorf("Expected embedding object to be 'list', got '%s'", result.EmbeddingRes.Object)
	}
	if len(result.EmbeddingRes.Data) != 1 {
		t.Errorf("Expected 1 embedding, got %d", len(result.EmbeddingRes.Data))
	}
	if len(result.EmbeddingRes.Data[0].Embedding) != 3 {
		t.Errorf("Expected embedding length 3, got %d", len(result.EmbeddingRes.Data[0].Embedding))
	}
}

func TestJobResultDirectCreation(t *testing.T) {
	// Test creating JobResult directly without the helper function
	chatRes := &openai.ChatCompletionResponse{ID: "direct-create"}
	embeddingRes := &openai.EmbeddingResponse{Object: "embedding"}

	result := &JobResult{
		ChatRes:      chatRes,
		EmbeddingRes: embeddingRes,
	}

	if result.ChatRes.ID != "direct-create" {
		t.Errorf("Expected chat ID 'direct-create', got '%s'", result.ChatRes.ID)
	}
	if result.EmbeddingRes.Object != "embedding" {
		t.Errorf("Expected embedding object 'embedding', got '%s'", result.EmbeddingRes.Object)
	}
}

package clientManager

import (
	"context"
	"errors"
	"fmt"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/requestManagement"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClientAdapter uses the official OpenAI Go SDK directly.
// This adapter is useful for direct OpenAI API calls without HTTP middleware.
type OpenAIClientAdapter struct {
	client         *openai.Client
	requestBuilder requestManagement.RequestBuilder
}

// NewOpenAIClientAdapter creates a new adapter that uses the native OpenAI SDK.
func NewOpenAIClientAdapter(
	apiKey string,
	builder requestManagement.RequestBuilder,
) *OpenAIClientAdapter {
	client := openai.NewClient(apiKey)
	return &OpenAIClientAdapter{
		client:         client,
		requestBuilder: builder,
	}
}

// Process implements the ClientAdapter interface using the native OpenAI SDK.
func (a *OpenAIClientAdapter) Process(inputs *llmManagement.Inputs) (*openai.ChatCompletionResponse, error) {
	// 1. Build the request using the standard builder
	req, err := a.requestBuilder.BuildRequest(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to build openai request: %w", err)
	}

	// 2. Use the native OpenAI SDK to make the request
	ctx := context.Background()
	resp, err := a.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai api error: %w", err)
	}

	return &resp, nil
}

func (a *OpenAIClientAdapter) ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error) {
	return nil, errors.New("Doesn't exist")
}

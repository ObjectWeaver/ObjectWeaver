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
package clientManager

import (
	"context"
	"errors"
	"fmt"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/domain"
	"objectweaver/llmManagement/requestManagement"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

// OpenAIClientAdapter uses the official OpenAI Go SDK directly.
// This adapter is useful for direct OpenAI API calls without HTTP middleware.
type OpenAIClientAdapter struct {
	client                  *openai.Client
	requestBuilder          requestManagement.RequestBuilder
	embeddingRequestBuilder requestManagement.EmbeddingRequestBuilder
}

// NewOpenAIClientAdapter creates a new adapter that uses the native OpenAI SDK.
func NewOpenAIClientAdapter(
	apiKey string,
	builder requestManagement.RequestBuilder,
	embeddingBuilder requestManagement.EmbeddingRequestBuilder,
) *OpenAIClientAdapter {
	client := openai.NewClient(apiKey)
	return &OpenAIClientAdapter{
		client:                  client,
		requestBuilder:          builder,
		embeddingRequestBuilder: embeddingBuilder,
	}
}

// Process implements the ClientAdapter interface using the native OpenAI SDK.
func (a *OpenAIClientAdapter) Process(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	if inputs.Def.Type == jsonSchema.Vector {
		return a.processEmbedding(inputs)
	}
	return a.processChat(inputs)
}

func (a *OpenAIClientAdapter) ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error) {
	return nil, errors.New("doesn't exist")
}

func (a *OpenAIClientAdapter) processChat(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
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

	return domain.CreateJobResult(&resp, nil), nil
}

func (a *OpenAIClientAdapter) processEmbedding(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	// 1. Build the embedding request using the standard builder
	req, err := a.embeddingRequestBuilder.BuildRequest(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to build openai embedding request: %w", err)
	}

	// 2. Use the native OpenAI SDK to make the request
	ctx := context.Background()
	resp, err := a.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai api error: %w", err)
	}

	return domain.CreateJobResult(nil, &resp), nil
}

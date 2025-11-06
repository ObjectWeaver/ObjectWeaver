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
package requestManagement

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

// CustomAnalysisResponse represents the structure of the custom response format
type CustomAnalysisResponse struct {
	ModelUsed string `json:"modelUsed"`
	Response  struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"response"`
}

type LocalResponse struct {
	Result   string               `json:"result"`
	Completion CustomAnalysisResponse `json:"completion"`
}

// standardConverter is the default implementation of the RequestConverter interface.
// It performs a direct JSON unmarshal from the response body to the target type.
type LocalConverter struct{}

// NewRequestConverter creates and returns a new instance of the LocalConverter,
// which adheres to the RequestConverter interface.
func NewLocalConverter() RequestConverter {
	return &LocalConverter{}
}

// ToChatCompletionResponse implements the conversion logic for the LocalConverter.
func (c *LocalConverter) ToChatCompletionResponse(resp *http.Response) (openai.ChatCompletionResponse, error) {
	var chatResponse openai.ChatCompletionResponse

	// Ensure the response is not nil to prevent panics.
	if resp == nil || resp.Body == nil {
		return chatResponse, fmt.Errorf("cannot convert a nil http.Response or response with nil body")
	}
	defer resp.Body.Close()

	// Read the entire response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return chatResponse, fmt.Errorf("failed to read response body: %w", err)
	}

	LocalRes := LocalResponse{}
	err = json.Unmarshal(body, &LocalRes)
	if err != nil {
		return chatResponse, fmt.Errorf("failed to unmarshal response body to LocalResponse: %w", err)
	}

	chatResponse.Model = LocalRes.Completion.ModelUsed
	chatResponse.Choices = []openai.ChatCompletionChoice{
		{
			Index: 0,
			Message: openai.ChatCompletionMessage{
				Role:    LocalRes.Completion.Response.Role,
				Content: LocalRes.Completion.Response.Content,
			},
			FinishReason: "stop",
		},
	}

	return chatResponse, nil
}

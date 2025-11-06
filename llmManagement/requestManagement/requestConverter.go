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


// RequestConverter defines an interface for converting a standard http.Response
// into a structured OpenAI ChatCompletionResponse.
type RequestConverter interface {
	// ToChatCompletionResponse reads the body of an http.Response, expecting a JSON
	// payload, and unmarshals it into a openai.ChatCompletionResponse.
	ToChatCompletionResponse(resp *http.Response) (openai.ChatCompletionResponse, error)
}

// standardConverter is the default implementation of the RequestConverter interface.
// It performs a direct JSON unmarshal from the response body to the target type.
type standardConverter struct{}

// NewRequestConverter creates and returns a new instance of the standardConverter,
// which adheres to the RequestConverter interface.
func NewRequestConverter() RequestConverter {
	return &standardConverter{}
}

// ToChatCompletionResponse implements the conversion logic for the standardConverter.
func (c *standardConverter) ToChatCompletionResponse(resp *http.Response) (openai.ChatCompletionResponse, error) {
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
	// Check HTTP status code before attempting to unmarshal
	if resp.StatusCode != http.StatusOK {
		// Try to parse as OpenAI error format
		var openaiError struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Param   string `json:"param"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if jsonErr := json.Unmarshal(body, &openaiError); jsonErr == nil && openaiError.Error.Message != "" {
			return chatResponse, fmt.Errorf("OpenAI API error (status %d): %s [type: %s]",
				resp.StatusCode, openaiError.Error.Message, openaiError.Error.Type)
		}

		// Fallback to generic error message
		return chatResponse, fmt.Errorf("HTTP request failed with status %d: %s. Response body: %s",
			resp.StatusCode, resp.Status, string(body))
	}

	// Unmarshal the JSON from the body into the ChatCompletionResponse struct.
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return chatResponse, fmt.Errorf("failed to unmarshal json into ChatCompletionResponse: %w. Response body: %s", err, string(body))
	}

	return chatResponse, nil
}

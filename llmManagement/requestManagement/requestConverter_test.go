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
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
package requestManagement

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestStandardConverter_ToChatCompletionResponse(t *testing.T) {
	converter := NewRequestConverter()

	t.Run("nil response", func(t *testing.T) {
		_, err := converter.ToChatCompletionResponse(nil)
		if err == nil {
			t.Error("expected error for nil response")
		}
		if !strings.Contains(err.Error(), "cannot convert a nil http.Response") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("nil body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       nil,
		}
		_, err := converter.ToChatCompletionResponse(resp)
		if err == nil {
			t.Error("expected error for nil body")
		}
		if !strings.Contains(err.Error(), "response with nil body") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("read body error", func(t *testing.T) {
		// Create a reader that errors
		body := &errorReader{err: io.ErrUnexpectedEOF}
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
		}
		_, err := converter.ToChatCompletionResponse(resp)
		if err == nil {
			t.Error("expected error when reading body fails")
		}
		if !strings.Contains(err.Error(), "failed to read response body") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("status 200 valid JSON", func(t *testing.T) {
		validResponse := openai.ChatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
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
			Usage: openai.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}
		bodyBytes, _ := json.Marshal(validResponse)
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(bodyBytes))),
		}
		result, err := converter.ToChatCompletionResponse(resp)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.ID != "test-id" {
			t.Errorf("expected ID 'test-id', got %s", result.ID)
		}
		if result.Choices[0].Message.Content != "Hello, world!" {
			t.Errorf("expected content 'Hello, world!', got %s", result.Choices[0].Message.Content)
		}
	})

	t.Run("status 200 invalid JSON", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("invalid json")),
		}
		_, err := converter.ToChatCompletionResponse(resp)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "failed to unmarshal json") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("status 400 with OpenAI error", func(t *testing.T) {
		openaiError := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid API key",
				"type":    "authentication_error",
				"param":   "api_key",
				"code":    "invalid_api_key",
			},
		}
		bodyBytes, _ := json.Marshal(openaiError)
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       io.NopCloser(strings.NewReader(string(bodyBytes))),
		}
		_, err := converter.ToChatCompletionResponse(resp)
		if err == nil {
			t.Error("expected error for non-200 status")
		}
		if !strings.Contains(err.Error(), "OpenAI API error") {
			t.Errorf("unexpected error message: %v", err)
		}
		if !strings.Contains(err.Error(), "Invalid API key") {
			t.Errorf("error should contain OpenAI message: %v", err)
		}
	})

	t.Run("status 500 without OpenAI error", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Status:     "500 Internal Server Error",
			Body:       io.NopCloser(strings.NewReader("Server error")),
		}
		_, err := converter.ToChatCompletionResponse(resp)
		if err == nil {
			t.Error("expected error for non-200 status")
		}
		if !strings.Contains(err.Error(), "HTTP request failed with status 500") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("status 400 with malformed OpenAI error", func(t *testing.T) {
		// Malformed error JSON, should fall back to generic error
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       io.NopCloser(strings.NewReader(`{"error": "not an object"}`)),
		}
		_, err := converter.ToChatCompletionResponse(resp)
		if err == nil {
			t.Error("expected error for non-200 status")
		}
		if !strings.Contains(err.Error(), "HTTP request failed with status 400") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

// errorReader is a helper to simulate a reader that returns an error
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errorReader) Close() error {
	return nil
}

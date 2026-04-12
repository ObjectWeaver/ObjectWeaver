package clientManager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"

	"github.com/sashabaranov/go-openai"
)

// Local interface definitions for testing
type RequestBuilder interface {
	BuildRequest(inputs *llmManagement.Inputs) (openai.ChatCompletionRequest, error)
}

type ModelConverter interface {
	Convert(model string) string
}

// Mock implementations for testing

type mockRequestBuilder struct {
	request openai.ChatCompletionRequest
	err     error
}

func (m *mockRequestBuilder) BuildRequest(inputs *llmManagement.Inputs) (openai.ChatCompletionRequest, error) {
	return m.request, m.err
}

type mockEmbeddingRequestBuilder struct{}

func (m *mockEmbeddingRequestBuilder) BuildRequest(inputs *llmManagement.Inputs) (openai.EmbeddingRequest, error) {
	return openai.EmbeddingRequest{}, nil
}

type mockModelConverter struct{}

func (m *mockModelConverter) Convert(model string) string {
	return string(model)
}

func TestNewGeminiClientAdapter(t *testing.T) {
	apiKey := "test-key"
	builder := &mockRequestBuilder{}
	embeddingBuilder := &mockEmbeddingRequestBuilder{}
	converter := &mockModelConverter{}
	httpClient := &http.Client{}

	adapter := NewGeminiClientAdapter(apiKey, builder, embeddingBuilder, converter, httpClient)

	if adapter.apiKey != apiKey {
		t.Errorf("expected apiKey %s, got %s", apiKey, adapter.apiKey)
	}
	if adapter.requestBuilder != builder {
		t.Errorf("expected requestBuilder to be set")
	}
	if adapter.httpClient != httpClient {
		t.Errorf("expected httpClient to be set")
	}
	if adapter.baseURL != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("expected baseURL to be set correctly")
	}
}

func TestNewGeminiClientAdapter_NilHttpClient(t *testing.T) {
	apiKey := "test-key"
	builder := &mockRequestBuilder{}
	embeddingBuilder := &mockEmbeddingRequestBuilder{}
	converter := &mockModelConverter{}

	adapter := NewGeminiClientAdapter(apiKey, builder, embeddingBuilder, converter, nil)

	if adapter.httpClient == nil {
		t.Errorf("expected httpClient to be initialized")
	}
}

func TestConvertToGeminiFormat_TextMessage(t *testing.T) {
	adapter := &GeminiClientAdapter{}

	req := openai.ChatCompletionRequest{
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Hello world",
			},
		},
	}

	result := adapter.convertToGeminiFormat(req)

	contents, ok := result["contents"].([]map[string]interface{})
	if !ok || len(contents) != 1 {
		t.Errorf("expected 1 content")
	}

	content := contents[0]
	if content["role"] != "user" {
		t.Errorf("expected role user")
	}

	parts, ok := content["parts"].([]map[string]interface{})
	if !ok || len(parts) != 1 {
		t.Errorf("expected 1 part")
	}

	if parts[0]["text"] != "Hello world" {
		t.Errorf("expected text 'Hello world'")
	}
}

func TestConvertToGeminiFormat_AssistantMessage(t *testing.T) {
	adapter := &GeminiClientAdapter{}

	req := openai.ChatCompletionRequest{
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleAssistant,
				Content: "Response",
			},
		},
	}

	result := adapter.convertToGeminiFormat(req)

	contents := result["contents"].([]map[string]interface{})
	content := contents[0]
	if content["role"] != "model" {
		t.Errorf("expected role model")
	}
}

func TestConvertToGeminiFormat_ImageMessage(t *testing.T) {
	adapter := &GeminiClientAdapter{}

	req := openai.ChatCompletionRequest{
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: "data:image/jpeg;base64,SGVsbG8=", // base64 for "Hello"
						},
					},
				},
			},
		},
	}

	result := adapter.convertToGeminiFormat(req)

	contents := result["contents"].([]map[string]interface{})
	content := contents[0]
	parts := content["parts"].([]map[string]interface{})

	part := parts[0]
	inlineData, ok := part["inline_data"].(map[string]interface{})
	if !ok {
		t.Errorf("expected inline_data")
	}

	if inlineData["mime_type"] != "image/jpeg" {
		t.Errorf("expected mime_type image/jpeg")
	}
	if inlineData["data"] != "SGVsbG8=" {
		t.Errorf("expected data SGVsbG8=")
	}
}

func TestConvertToGeminiFormat_GenerationConfig(t *testing.T) {
	adapter := &GeminiClientAdapter{}

	req := openai.ChatCompletionRequest{
		Temperature: 0.5,
		TopP:        0.9,
		MaxTokens:   100,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Test",
			},
		},
	}

	result := adapter.convertToGeminiFormat(req)

	config, ok := result["generationConfig"].(map[string]interface{})
	if !ok {
		t.Errorf("expected generationConfig")
	}

	if temp, ok := config["temperature"].(float32); !ok || temp != 0.5 {
		t.Errorf("expected temperature 0.5")
	}
	if topP, ok := config["topP"].(float32); !ok || topP != 0.9 {
		t.Errorf("expected topP 0.9")
	}
	if maxTokens, ok := config["maxOutputTokens"].(int); !ok || maxTokens != 100 {
		t.Errorf("expected maxOutputTokens 100")
	}
}

func TestConvertFromGeminiFormat(t *testing.T) {
	adapter := &GeminiClientAdapter{}

	geminiResponse := map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"parts": []map[string]interface{}{
						{"text": "Hello"},
						{"text": " World"},
					},
					"role": "model",
				},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]interface{}{
			"promptTokenCount":     10,
			"candidatesTokenCount": 5,
			"totalTokenCount":      15,
		},
	}

	jsonData, _ := json.Marshal(geminiResponse)
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(string(jsonData))),
	}

	result, err := adapter.convertFromGeminiFormat(resp)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Choices[0].Message.Content != "Hello World" {
		t.Errorf("expected content 'Hello World', got '%s'", result.Choices[0].Message.Content)
	}
	if result.Choices[0].FinishReason != openai.FinishReasonStop {
		t.Errorf("expected finish reason stop")
	}
	if result.Usage.PromptTokens != 10 {
		t.Errorf("expected prompt tokens 10")
	}
	if result.Usage.CompletionTokens != 5 {
		t.Errorf("expected completion tokens 5")
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("expected total tokens 15")
	}
}

func TestConvertFromGeminiFormat_FinishReasons(t *testing.T) {
	adapter := &GeminiClientAdapter{}

	testCases := []struct {
		geminiReason string
		openaiReason openai.FinishReason
	}{
		{"STOP", openai.FinishReasonStop},
		{"MAX_TOKENS", openai.FinishReasonLength},
		{"SAFETY", openai.FinishReasonContentFilter},
		{"UNKNOWN", openai.FinishReasonStop}, // default
	}

	for _, tc := range testCases {
		geminiResponse := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "Test"},
						},
					},
					"finishReason": tc.geminiReason,
				},
			},
		}

		jsonData, _ := json.Marshal(geminiResponse)
		resp := &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(string(jsonData))),
		}

		result, _ := adapter.convertFromGeminiFormat(resp)
		if result.Choices[0].FinishReason != tc.openaiReason {
			t.Errorf("expected finish reason %s for gemini %s", tc.openaiReason, tc.geminiReason)
		}
	}
}

func TestProcess_Success(t *testing.T) {
	// Mock request builder
	mockReq := openai.ChatCompletionRequest{
		Model: "gemini-2.5-flash-lite",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: "Test"},
		},
	}
	builder := &mockRequestBuilder{request: mockReq}
	embeddingBuilder := &mockEmbeddingRequestBuilder{}
	converter := &mockModelConverter{}

	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/models/") {
			w.WriteHeader(404)
			return
		}
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "Mocked response"},
						},
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     5,
				"candidatesTokenCount": 3,
				"totalTokenCount":      8,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create adapter with test server client
	httpClient := server.Client()
	adapter := NewGeminiClientAdapter("test-key", builder, embeddingBuilder, converter, httpClient)
	adapter.baseURL = server.URL

	inputs := &llmManagement.Inputs{} // Assuming empty inputs for mock

	result, err := adapter.Process(inputs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ChatRes.Choices[0].Message.Content != "Mocked response" {
		t.Errorf("expected 'Mocked response', got '%s'", result.ChatRes.Choices[0].Message.Content)
	}
}

func TestProcess_BuildRequestError(t *testing.T) {
	builder := &mockRequestBuilder{err: fmt.Errorf("build error")}
	embeddingBuilder := &mockEmbeddingRequestBuilder{}
	converter := &mockModelConverter{}
	adapter := NewGeminiClientAdapter("key", builder, embeddingBuilder, converter, nil)

	inputs := &llmManagement.Inputs{}
	_, err := adapter.Process(inputs)
	if err == nil || !strings.Contains(err.Error(), "build error") {
		t.Errorf("expected build error")
	}
}

func TestProcess_ApiError(t *testing.T) {
	mockReq := openai.ChatCompletionRequest{Model: "gemini-test"}
	builder := &mockRequestBuilder{request: mockReq}
	embeddingBuilder := &mockEmbeddingRequestBuilder{}
	converter := &mockModelConverter{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/models/") {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("API error"))
	}))
	defer server.Close()

	httpClient := server.Client()
	adapter := NewGeminiClientAdapter("test-key", builder, embeddingBuilder, converter, httpClient)
	adapter.baseURL = server.URL

	inputs := &llmManagement.Inputs{}
	_, err := adapter.Process(inputs)
	if err == nil || !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected API error")
	}
}

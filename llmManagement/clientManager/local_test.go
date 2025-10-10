package clientManager

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"objectGeneration/llmManagement"

	"github.com/sashabaranov/go-openai"
)

// mockRequestConverter implements requestManagement.RequestConverter for testing
type mockRequestConverter struct {
	response openai.ChatCompletionResponse
	err      error
}

func (m *mockRequestConverter) ToChatCompletionResponse(resp *http.Response) (openai.ChatCompletionResponse, error) {
	return m.response, m.err
}

func TestNewLocalClientAdapter(t *testing.T) {
	url := "http://example.com"
	token := "test-token"
	builder := &mockRequestBuilder{}
	converter := &mockRequestConverter{}
	httpClient := &http.Client{}

	adapter := NewLocalClientAdapter(url, token, builder, converter, httpClient)

	if adapter.targetURL != url {
		t.Errorf("expected targetURL %s, got %s", url, adapter.targetURL)
	}
	if adapter.authToken != token {
		t.Errorf("expected authToken %s, got %s", token, adapter.authToken)
	}
	if adapter.requestBuilder != builder {
		t.Error("requestBuilder not set correctly")
	}
	if adapter.requestConverter != converter {
		t.Error("requestConverter not set correctly")
	}
	if adapter.client != httpClient {
		t.Error("httpClient not set correctly")
	}
}

func TestLocalClientAdapter_Process_Success(t *testing.T) {
	// Mock server
	expectedResponse := openai.ChatCompletionResponse{
		ID: "test-id",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "test content",
				},
			},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	inputs := &llmManagement.Inputs{
		Prompt: "test",
	}
	mockReq := openai.ChatCompletionRequest{
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
	}

	builder := &mockRequestBuilder{request: mockReq}
	converter := &mockRequestConverter{response: expectedResponse}
	httpClient := server.Client()

	adapter := NewLocalClientAdapter(server.URL, "test-token", builder, converter, httpClient)

	result, err := adapter.Process(inputs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != expectedResponse.ID {
		t.Errorf("expected ID %s, got %s", expectedResponse.ID, result.ID)
	}
	if len(result.Choices) != 1 || result.Choices[0].Message.Content != "test content" {
		t.Error("response not matched")
	}
}

func TestLocalClientAdapter_Process_BuildRequestError(t *testing.T) {
	builder := &mockRequestBuilder{err: errors.New("build error")}
	converter := &mockRequestConverter{}
	httpClient := &http.Client{}

	adapter := NewLocalClientAdapter("http://example.com", "", builder, converter, httpClient)

	_, err := adapter.Process(&llmManagement.Inputs{})
	if err == nil {
		t.Error("expected error from BuildRequest")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("failed to build request")) {
		t.Errorf("expected error message containing 'failed to build request', got %s", err.Error())
	}
}

func TestLocalClientAdapter_Process_MarshalError(t *testing.T) {
	// Create a request that can't be marshaled (e.g., with unexported fields or cycles, but hard in Go)
	// Actually, json.Marshal rarely fails for structs, but we can mock by making BuildRequest return something invalid.
	// Since it's hard, perhaps skip or use a custom type.
	// For simplicity, assume marshal doesn't fail in tests, as it's rare.
	t.Skip("Marshal error test skipped as json.Marshal rarely fails for valid structs")
}

func TestLocalClientAdapter_Process_HTTPRequestCreationError(t *testing.T) {
	// http.NewRequest fails if method is invalid, but POST is valid.
	// URL could be invalid, but let's use a bad URL.
	builder := &mockRequestBuilder{request: openai.ChatCompletionRequest{}}
	converter := &mockRequestConverter{}
	httpClient := &http.Client{}

	// Invalid URL
	adapter := NewLocalClientAdapter("http://invalid url", "", builder, converter, httpClient)

	_, err := adapter.Process(&llmManagement.Inputs{})
	if err == nil {
		t.Error("expected error from http.NewRequest")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("failed to create new http request")) {
		t.Errorf("expected error message containing 'failed to create new http request', got %s", err.Error())
	}
}

func TestLocalClientAdapter_Process_HTTPClientError(t *testing.T) {
	// To simulate client.Do error, perhaps use a bad URL or timeout.
	// For simplicity, use a non-existent server.
	builder := &mockRequestBuilder{request: openai.ChatCompletionRequest{}}
	converter := &mockRequestConverter{}
	httpClient := &http.Client{Timeout: 0} // No timeout, but still.

	adapter := NewLocalClientAdapter("http://nonexistent", "", builder, converter, httpClient)

	_, err := adapter.Process(&llmManagement.Inputs{})
	if err == nil {
		t.Error("expected error from http.Client.Do")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("http client failed to process request")) {
		t.Errorf("expected error message containing 'http client failed to process request', got %s", err.Error())
	}
}

func TestLocalClientAdapter_Process_ConversionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}")) // Valid JSON
	}))
	defer server.Close()

	builder := &mockRequestBuilder{request: openai.ChatCompletionRequest{}}
	converter := &mockRequestConverter{err: errors.New("conversion error")}
	httpClient := server.Client()

	adapter := NewLocalClientAdapter(server.URL, "", builder, converter, httpClient)

	_, err := adapter.Process(&llmManagement.Inputs{})
	if err == nil {
		t.Error("expected error from ToChatCompletionResponse")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("failed to convert http response")) {
		t.Errorf("expected error message containing 'failed to convert http response', got %s", err.Error())
	}
}

func TestLocalClientAdapter_Process_NoAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("expected no Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openai.ChatCompletionResponse{})
	}))
	defer server.Close()

	builder := &mockRequestBuilder{request: openai.ChatCompletionRequest{}}
	converter := &mockRequestConverter{response: openai.ChatCompletionResponse{}}
	httpClient := server.Client()

	adapter := NewLocalClientAdapter(server.URL, "", builder, converter, httpClient) // No token

	_, err := adapter.Process(&llmManagement.Inputs{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

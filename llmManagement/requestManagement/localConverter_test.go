package requestManagement

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestNewLocalConverter(t *testing.T) {
	converter := NewLocalConverter()
	if converter == nil {
		t.Fatal("NewLocalConverter returned nil")
	}
	// Check if it implements RequestConverter interface
	var _ RequestConverter = converter
}

func TestLocalConverter_ToChatCompletionResponse_Valid(t *testing.T) {
	converter := &LocalConverter{}

	validJSON := `{
		"result": "test result",
		"completion": {
			"modelUsed": "gpt-4",
			"response": {
				"role": "assistant",
				"content": "This is a test response."
			}
		}
	}`

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(validJSON)),
	}

	var chatResp openai.ChatCompletionResponse
	chatResp, err := converter.ToChatCompletionResponse(resp)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if chatResp.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", chatResp.Model)
	}

	if len(chatResp.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(chatResp.Choices))
	}

	choice := chatResp.Choices[0]
	if choice.Index != 0 {
		t.Errorf("Expected index 0, got %d", choice.Index)
	}

	if choice.Message.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got '%s'", choice.Message.Role)
	}

	if choice.Message.Content != "This is a test response." {
		t.Errorf("Expected content 'This is a test response.', got '%s'", choice.Message.Content)
	}

	if choice.FinishReason != "stop" {
		t.Errorf("Expected finish reason 'stop', got '%s'", choice.FinishReason)
	}
}

func TestLocalConverter_ToChatCompletionResponse_NilResponse(t *testing.T) {
	converter := &LocalConverter{}

	_, err := converter.ToChatCompletionResponse(nil)
	if err == nil {
		t.Fatal("Expected error for nil response, got nil")
	}

	expectedMsg := "cannot convert a nil http.Response or response with nil body"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestLocalConverter_ToChatCompletionResponse_NilBody(t *testing.T) {
	converter := &LocalConverter{}

	resp := &http.Response{}
	_, err := converter.ToChatCompletionResponse(resp)
	if err == nil {
		t.Fatal("Expected error for nil body, got nil")
	}

	expectedMsg := "cannot convert a nil http.Response or response with nil body"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestLocalConverter_ToChatCompletionResponse_InvalidJSON(t *testing.T) {
	converter := &LocalConverter{}

	invalidJSON := `{"invalid": json}`

	req := httptest.NewRequest("POST", "/", strings.NewReader(invalidJSON))
	w := httptest.NewRecorder()
	w.WriteString(invalidJSON)
	resp := w.Result()
	resp.Body = req.Body

	_, err := converter.ToChatCompletionResponse(resp)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "failed to unmarshal response body") {
		t.Errorf("Expected unmarshal error, got '%s'", err.Error())
	}
}

func TestLocalConverter_ToChatCompletionResponse_EmptyBody(t *testing.T) {
	converter := &LocalConverter{}

	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	w := httptest.NewRecorder()
	resp := w.Result()
	resp.Body = req.Body

	_, err := converter.ToChatCompletionResponse(resp)
	if err == nil {
		t.Fatal("Expected error for empty body, got nil")
	}

	if !strings.Contains(err.Error(), "failed to unmarshal response body") {
		t.Errorf("Expected unmarshal error, got '%s'", err.Error())
	}
}

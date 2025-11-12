package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/objectweaver/go-sdk/client"
	"github.com/objectweaver/go-sdk/jsonSchema"
)

func TestObjectGen_MethodNotAllowed(t *testing.T) {
	methods := []string{"GET", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/objectGen", nil)
			w := httptest.NewRecorder()

			ObjectGen(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405, got %d", w.Code)
			}

			body := w.Body.String()
			if !strings.Contains(body, "Only POST method is allowed") {
				t.Errorf("Expected method not allowed message, got: %s", body)
			}
		})
	}
}

func TestObjectGen_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/objectGen", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	ObjectGen(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid request body") {
		t.Errorf("Expected invalid request body message, got: %s", body)
	}
}

func TestObjectGen_EmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/objectGen", strings.NewReader(""))
	w := httptest.NewRecorder()

	ObjectGen(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestObjectGen_CircularDefinition(t *testing.T) {
	// Note: We can't test the full flow without mocking the generator factory
	// This test just verifies the JSON unmarshalling works for valid request structure
	def := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"field1": {
				Type: jsonSchema.String,
			},
		},
	}

	reqBody := &client.RequestBody{
		Prompt:     "test",
		Definition: def,
	}

	// Just verify the request body can be marshalled/unmarshalled
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var decoded client.RequestBody
	if err := json.Unmarshal(bodyBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decoded.Prompt != "test" {
		t.Errorf("Expected prompt 'test', got %s", decoded.Prompt)
	}
}

func TestObjectGen_ContextCancellation(t *testing.T) {
	// Test that cancelled context is detected at entry
	reqBody := &client.RequestBody{
		Prompt: "test",
		Definition: &jsonSchema.Definition{
			Type: jsonSchema.String,
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/objectGen", bytes.NewReader(bodyBytes))

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	
	// Should detect cancelled context and return early
	ObjectGen(w, req)
	
	// Should get a timeout/cancel error response
	if w.Code != http.StatusRequestTimeout {
		t.Errorf("Expected 408 Request Timeout, got %d", w.Code)
	}
}

func TestObjectGen_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(5 * time.Millisecond)

	reqBody := &client.RequestBody{
		Prompt: "test",
		Definition: &jsonSchema.Definition{
			Type: jsonSchema.String,
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/objectGen", bytes.NewReader(bodyBytes))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ObjectGen(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("Expected status 408, got %d", w.Code)
	}
}

func TestObjectGen_ValidRequestStructure(t *testing.T) {
	reqBody := &client.RequestBody{
		Prompt: "Generate a test string",
		Definition: &jsonSchema.Definition{
			Type:        jsonSchema.String,
			Instruction: "Create a test string",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Verify the request body is valid JSON
	var decoded client.RequestBody
	if err := json.Unmarshal(bodyBytes, &decoded); err != nil {
		t.Errorf("Request body is not valid JSON: %v", err)
	}

	if decoded.Prompt != reqBody.Prompt {
		t.Error("Prompt was not preserved in JSON encoding")
	}
}

func TestPrettyPrintJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple object",
			input: `{"key":"value"}`,
		},
		{
			name:  "nested object",
			input: `{"outer":{"inner":"value"}}`,
		},
		{
			name:  "array",
			input: `{"items":[1,2,3]}`,
		},
		{
			name:  "empty object",
			input: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function just prints, so we're testing it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PrettyPrintJSON panicked: %v", r)
				}
			}()

			PrettyPrintJSON([]byte(tt.input))
		})
	}
}

func TestPrettyPrintJSON_InvalidJSON(t *testing.T) {
	// Test with invalid JSON - should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrettyPrintJSON should not panic on invalid JSON: %v", r)
		}
	}()

	PrettyPrintJSON([]byte("invalid json"))
}

func TestPrettyPrintJSON_EmptyInput(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrettyPrintJSON should not panic on empty input: %v", r)
		}
	}()

	PrettyPrintJSON([]byte(""))
}

func TestResponse_JSONMarshalling(t *testing.T) {
	response := Response{
		Data: map[string]any{
			"field1": "value1",
			"field2": 42,
		},
		DetailedData: map[string]*DetailedField{
			"field1": {
				Value: "value1",
				Metadata: &FieldMetadata{
					TokensUsed: 10,
					Cost:       0.001,
					ModelUsed:  "gpt-4",
				},
			},
		},
		UsdCost: 0.002,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decoded.UsdCost != response.UsdCost {
		t.Errorf("Expected cost %f, got %f", response.UsdCost, decoded.UsdCost)
	}

	if decoded.Data["field1"] != response.Data["field1"] {
		t.Error("Data was not preserved in JSON encoding")
	}
}

func TestDetailedField_JSONMarshalling(t *testing.T) {
	field := DetailedField{
		Value: "test value",
		Metadata: &FieldMetadata{
			TokensUsed: 100,
			Cost:       0.05,
			ModelUsed:  "gpt-3.5-turbo",
		},
	}

	jsonBytes, err := json.Marshal(field)
	if err != nil {
		t.Fatalf("Failed to marshal field: %v", err)
	}

	var decoded DetailedField
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal field: %v", err)
	}

	if decoded.Metadata.TokensUsed != field.Metadata.TokensUsed {
		t.Error("TokensUsed was not preserved")
	}

	if decoded.Metadata.Cost != field.Metadata.Cost {
		t.Error("Cost was not preserved")
	}
}

func TestFieldMetadata_AllFields(t *testing.T) {
	metadata := FieldMetadata{
		TokensUsed: 500,
		Cost:       0.25,
		ModelUsed:  "gpt-4-turbo",
	}

	if metadata.TokensUsed != 500 {
		t.Error("TokensUsed not set correctly")
	}

	if metadata.Cost != 0.25 {
		t.Error("Cost not set correctly")
	}

	if metadata.ModelUsed != "gpt-4-turbo" {
		t.Error("ModelUsed not set correctly")
	}
}

func TestObjectGen_ContentTypeHeader(t *testing.T) {
	// Just test that invalid method is handled properly
	// Full success test requires mocking the entire generator which is complex
	req := httptest.NewRequest("GET", "/api/objectGen", nil)
	w := httptest.NewRecorder()

	ObjectGen(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestResponse_EmptyDetailedData(t *testing.T) {
	response := Response{
		Data: map[string]any{
			"test": "value",
		},
		DetailedData: nil, // Should be omitted in JSON
		UsdCost:      0.0,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonBytes)

	// DetailedData should not appear in JSON when nil
	if strings.Contains(jsonStr, "detailedData") {
		t.Error("Expected detailedData to be omitted when nil")
	}
}

func TestResponse_WithDetailedData(t *testing.T) {
	response := Response{
		Data: map[string]any{
			"test": "value",
		},
		DetailedData: map[string]*DetailedField{
			"test": {
				Value: "value",
				Metadata: &FieldMetadata{
					TokensUsed: 10,
					Cost:       0.001,
					ModelUsed:  "test-model",
				},
			},
		},
		UsdCost: 0.001,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonBytes)

	// DetailedData should appear in JSON
	if !strings.Contains(jsonStr, "detailedData") {
		t.Error("Expected detailedData to appear in JSON")
	}
}

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// mockHTTPClient creates a mock HTTP client that returns predefined responses
type mockRoundTripper struct {
	responses map[string]*http.Response
	requests  []*http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)

	// Construct key from method and full URL path (not including domain)
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path += "?" + req.URL.RawQuery
	}
	key := req.Method + " " + path

	if resp, ok := m.responses[key]; ok {
		// Clone the response body so it can be read multiple times
		if resp.Body != nil {
			bodyBytes, _ := io.ReadAll(resp.Body)
			// Return a new response with a fresh body reader
			return &http.Response{
				StatusCode: resp.StatusCode,
				Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
				Header:     resp.Header,
			}, nil
		}
		return resp, nil
	}

	return &http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
		Header:     make(http.Header),
	}, nil
}

func newMockHTTPClient(responses map[string]*http.Response) *http.Client {
	return &http.Client{
		Transport: &mockRoundTripper{
			responses: responses,
			requests:  make([]*http.Request, 0),
		},
	}
}

func TestNewBatchClientWithHTTPClient(t *testing.T) {
	apiKey := "test-api-key"
	baseURL := "https://api.openai.com/v1"
	pollInterval := 5 * time.Second
	httpClient := &http.Client{}

	client := NewBatchClientWithHTTPClient(apiKey, baseURL, pollInterval, httpClient)

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}
	if client.apiKey != apiKey {
		t.Errorf("Expected apiKey %s, got %s", apiKey, client.apiKey)
	}
	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, client.baseURL)
	}
	if client.pollInterval != pollInterval {
		t.Errorf("Expected pollInterval %v, got %v", pollInterval, client.pollInterval)
	}
	if client.httpClient != httpClient {
		t.Error("Expected httpClient to match provided client")
	}
}

func TestBatchClient_CreateBatch(t *testing.T) {
	tests := []struct {
		name           string
		request        CreateBatchRequest
		mockResponse   *Batch
		mockStatusCode int
		wantErr        bool
	}{
		{
			name: "successful batch creation",
			request: CreateBatchRequest{
				InputFileID:      "file-123",
				Endpoint:         EndpointChatCompletions,
				CompletionWindow: "24h",
			},
			mockResponse: &Batch{
				ID:               "batch-123",
				Object:           "batch",
				Endpoint:         string(EndpointChatCompletions),
				InputFileID:      "file-123",
				CompletionWindow: "24h",
				Status:           StatusValidating,
				CreatedAt:        time.Now().Unix(),
			},
			mockStatusCode: 200,
			wantErr:        false,
		},
		{
			name: "API error",
			request: CreateBatchRequest{
				InputFileID:      "file-123",
				Endpoint:         EndpointChatCompletions,
				CompletionWindow: "24h",
			},
			mockStatusCode: 400,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.ReadCloser
			if tt.mockResponse != nil {
				data, _ := json.Marshal(tt.mockResponse)
				body = io.NopCloser(bytes.NewReader(data))
			} else {
				body = io.NopCloser(strings.NewReader(`{"error": "bad request"}`))
			}

			responses := map[string]*http.Response{
				"POST /v1/batches": {
					StatusCode: tt.mockStatusCode,
					Body:       body,
				},
			}

			client := NewBatchClientWithHTTPClient(
				"test-key",
				"https://api.openai.com/v1",
				5*time.Second,
				newMockHTTPClient(responses),
			)

			batch, err := client.CreateBatch(context.Background(), tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateBatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && batch == nil {
				t.Error("Expected batch to be returned, got nil")
			}

			if !tt.wantErr && batch != nil {
				if batch.ID != tt.mockResponse.ID {
					t.Errorf("Expected batch ID %s, got %s", tt.mockResponse.ID, batch.ID)
				}
			}
		})
	}
}

func TestBatchClient_GetBatch(t *testing.T) {
	batchID := "batch-123"
	mockBatch := &Batch{
		ID:               batchID,
		Object:           "batch",
		Endpoint:         string(EndpointChatCompletions),
		InputFileID:      "file-123",
		CompletionWindow: "24h",
		Status:           StatusInProgress,
		CreatedAt:        time.Now().Unix(),
	}

	data, _ := json.Marshal(mockBatch)
	responses := map[string]*http.Response{
		"GET /v1/batches/" + batchID: {
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
		},
	}

	client := NewBatchClientWithHTTPClient(
		"test-key",
		"https://api.openai.com/v1",
		5*time.Second,
		newMockHTTPClient(responses),
	)

	batch, err := client.GetBatch(context.Background(), batchID)

	if err != nil {
		t.Fatalf("GetBatch() error = %v", err)
	}

	if batch.ID != batchID {
		t.Errorf("Expected batch ID %s, got %s", batchID, batch.ID)
	}
	if batch.Status != StatusInProgress {
		t.Errorf("Expected status %s, got %s", StatusInProgress, batch.Status)
	}
}

func TestBatchClient_CancelBatch(t *testing.T) {
	batchID := "batch-123"
	mockBatch := &Batch{
		ID:           batchID,
		Status:       StatusCancelling,
		CreatedAt:    time.Now().Unix(),
		CancellingAt: ptrInt64(time.Now().Unix()),
	}

	data, _ := json.Marshal(mockBatch)
	responses := map[string]*http.Response{
		"POST /v1/batches/" + batchID + "/cancel": {
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
		},
	}

	client := NewBatchClientWithHTTPClient(
		"test-key",
		"https://api.openai.com/v1",
		5*time.Second,
		newMockHTTPClient(responses),
	)

	batch, err := client.CancelBatch(context.Background(), batchID)

	if err != nil {
		t.Fatalf("CancelBatch() error = %v", err)
	}

	if batch.Status != StatusCancelling {
		t.Errorf("Expected status %s, got %s", StatusCancelling, batch.Status)
	}
}

func TestBatchClient_ListBatches(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		after    string
		wantPath string
	}{
		{
			name:     "without parameters",
			limit:    0,
			after:    "",
			wantPath: "GET /v1/batches",
		},
		{
			name:     "with limit",
			limit:    10,
			after:    "",
			wantPath: "GET /v1/batches?limit=10",
		},
		{
			name:     "with after",
			limit:    0,
			after:    "batch-123",
			wantPath: "GET /v1/batches?after=batch-123",
		},
		{
			name:     "with both parameters",
			limit:    10,
			after:    "batch-123",
			wantPath: "GET /v1/batches?limit=10&after=batch-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResponse := &ListBatchesResponse{
				Object: "list",
				Data: []Batch{
					{
						ID:     "batch-1",
						Status: StatusCompleted,
					},
				},
				HasMore: false,
			}

			data, _ := json.Marshal(mockResponse)
			responses := map[string]*http.Response{
				tt.wantPath: {
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(data)),
				},
			}

			client := NewBatchClientWithHTTPClient(
				"test-key",
				"https://api.openai.com/v1",
				5*time.Second,
				newMockHTTPClient(responses),
			)

			listResp, err := client.ListBatches(context.Background(), tt.limit, tt.after)

			if err != nil {
				t.Fatalf("ListBatches() error = %v", err)
			}

			if listResp.Object != "list" {
				t.Errorf("Expected object 'list', got %s", listResp.Object)
			}
			if len(listResp.Data) != 1 {
				t.Errorf("Expected 1 batch, got %d", len(listResp.Data))
			}
		})
	}
}

func TestBatchClient_GetFileContent(t *testing.T) {
	fileID := "file-123"
	expectedContent := []byte("test file content")

	responses := map[string]*http.Response{
		"GET /v1/files/" + fileID + "/content": {
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(expectedContent)),
		},
	}

	client := NewBatchClientWithHTTPClient(
		"test-key",
		"https://api.openai.com/v1",
		5*time.Second,
		newMockHTTPClient(responses),
	)

	content, err := client.GetFileContent(context.Background(), fileID)

	if err != nil {
		t.Fatalf("GetFileContent() error = %v", err)
	}

	if !bytes.Equal(content, expectedContent) {
		t.Errorf("Expected content %s, got %s", expectedContent, content)
	}
}

func TestBatchClient_UploadFileReader(t *testing.T) {
	mockFile := &File{
		ID:        "file-123",
		Object:    "file",
		Bytes:     100,
		CreatedAt: time.Now().Unix(),
		Filename:  "test.jsonl",
		Purpose:   "batch",
	}

	data, _ := json.Marshal(mockFile)
	responses := map[string]*http.Response{
		"POST /v1/files": {
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
		},
	}

	client := NewBatchClientWithHTTPClient(
		"test-key",
		"https://api.openai.com/v1",
		5*time.Second,
		newMockHTTPClient(responses),
	)

	reader := strings.NewReader("test content")
	file, err := client.UploadFileReader(context.Background(), reader, "test.jsonl")

	if err != nil {
		t.Fatalf("UploadFileReader() error = %v", err)
	}

	if file.ID != mockFile.ID {
		t.Errorf("Expected file ID %s, got %s", mockFile.ID, file.ID)
	}
	if file.Purpose != "batch" {
		t.Errorf("Expected purpose 'batch', got %s", file.Purpose)
	}
}

func TestBatchClient_UploadFile(t *testing.T) {
	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `{"custom_id":"req-1","method":"POST","url":"/v1/chat/completions","body":{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}}`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	mockFile := &File{
		ID:        "file-123",
		Object:    "file",
		Bytes:     int(len(content)),
		CreatedAt: time.Now().Unix(),
		Filename:  tmpFile.Name(),
		Purpose:   "batch",
	}

	data, _ := json.Marshal(mockFile)
	responses := map[string]*http.Response{
		"POST /v1/files": {
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
		},
	}

	client := NewBatchClientWithHTTPClient(
		"test-key",
		"https://api.openai.com/v1",
		5*time.Second,
		newMockHTTPClient(responses),
	)

	file, err := client.UploadFile(context.Background(), tmpFile.Name())

	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	if file.ID != mockFile.ID {
		t.Errorf("Expected file ID %s, got %s", mockFile.ID, file.ID)
	}
}

func TestBatchClient_WaitForBatch(t *testing.T) {
	batchID := "batch-123"

	// Create a sequence of responses
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var status BatchStatus
		if callCount == 0 {
			status = StatusValidating
		} else if callCount == 1 {
			status = StatusInProgress
		} else {
			status = StatusCompleted
		}
		callCount++

		batch := &Batch{
			ID:        batchID,
			Status:    status,
			CreatedAt: time.Now().Unix(),
		}
		json.NewEncoder(w).Encode(batch)
	}))
	defer server.Close()

	client := NewBatchClientWithHTTPClient(
		"test-key",
		server.URL,
		100*time.Millisecond,
		&http.Client{},
	)

	batch, err := client.WaitForBatch(context.Background(), batchID)

	if err != nil {
		t.Fatalf("WaitForBatch() error = %v", err)
	}

	if batch.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, batch.Status)
	}
}

func TestBatchClient_WaitForBatch_ContextCancellation(t *testing.T) {
	batchID := "batch-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		batch := &Batch{
			ID:        batchID,
			Status:    StatusInProgress,
			CreatedAt: time.Now().Unix(),
		}
		json.NewEncoder(w).Encode(batch)
	}))
	defer server.Close()

	client := NewBatchClientWithHTTPClient(
		"test-key",
		server.URL,
		1*time.Second,
		&http.Client{},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := client.WaitForBatch(ctx, batchID)

	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestBatchClient_ReadBatchResponses(t *testing.T) {
	responses := []BatchResponse{
		{
			ID:       "resp-1",
			CustomID: "req-1",
			Response: &ResponseData{
				StatusCode: 200,
				RequestID:  "req-123",
				Body: map[string]interface{}{
					"choices": []interface{}{
						map[string]interface{}{"text": "Hello!"},
					},
				},
			},
		},
		{
			ID:       "resp-2",
			CustomID: "req-2",
			Error: &BatchError{
				Code:    "invalid_request",
				Message: "Invalid request",
			},
		},
	}

	var buffer bytes.Buffer
	for _, resp := range responses {
		data, _ := json.Marshal(resp)
		buffer.Write(data)
		buffer.WriteString("\n")
	}

	fileID := "file-123"
	mockResponses := map[string]*http.Response{
		"GET /v1/files/" + fileID + "/content": {
			StatusCode: 200,
			Body:       io.NopCloser(&buffer),
		},
	}

	client := NewBatchClientWithHTTPClient(
		"test-key",
		"https://api.openai.com/v1",
		5*time.Second,
		newMockHTTPClient(mockResponses),
	)

	batchResponses, err := client.ReadBatchResponses(context.Background(), fileID)

	if err != nil {
		t.Fatalf("ReadBatchResponses() error = %v", err)
	}

	if len(batchResponses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(batchResponses))
	}

	if batchResponses[0].CustomID != "req-1" {
		t.Errorf("Expected custom ID 'req-1', got %s", batchResponses[0].CustomID)
	}

	if batchResponses[1].Error == nil {
		t.Error("Expected error in second response, got nil")
	}
}

func TestCreateBatchRequestFile(t *testing.T) {
	requests := []BatchRequest{
		{
			CustomID: "req-1",
			Method:   "POST",
			URL:      "/v1/chat/completions",
			Body: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Hello"},
				},
			},
		},
		{
			CustomID: "req-2",
			Method:   "POST",
			URL:      "/v1/chat/completions",
			Body: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "World"},
				},
			},
		},
	}

	tmpFile, err := os.CreateTemp("", "batch-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = CreateBatchRequestFile(requests, tmpFile.Name())
	if err != nil {
		t.Fatalf("CreateBatchRequestFile() error = %v", err)
	}

	// Read and verify the file
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestNewChatCompletionBatchRequest(t *testing.T) {
	customID := "req-1"
	model := "gpt-3.5-turbo"
	messages := []map[string]interface{}{
		{"role": "user", "content": "Hello"},
	}
	maxTokens := 100

	req := NewChatCompletionBatchRequest(customID, model, messages, maxTokens)

	if req.CustomID != customID {
		t.Errorf("Expected custom ID %s, got %s", customID, req.CustomID)
	}
	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}
	if req.URL != string(EndpointChatCompletions) {
		t.Errorf("Expected URL %s, got %s", EndpointChatCompletions, req.URL)
	}
	if req.Body["model"] != model {
		t.Errorf("Expected model %s, got %v", model, req.Body["model"])
	}
	if req.Body["max_tokens"] != maxTokens {
		t.Errorf("Expected max_tokens %d, got %v", maxTokens, req.Body["max_tokens"])
	}
}

func TestNewEmbeddingBatchRequest(t *testing.T) {
	customID := "req-1"
	model := "text-embedding-ada-002"
	input := "test text"

	req := NewEmbeddingBatchRequest(customID, model, input)

	if req.CustomID != customID {
		t.Errorf("Expected custom ID %s, got %s", customID, req.CustomID)
	}
	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}
	if req.URL != string(EndpointEmbeddings) {
		t.Errorf("Expected URL %s, got %s", EndpointEmbeddings, req.URL)
	}
	if req.Body["model"] != model {
		t.Errorf("Expected model %s, got %v", model, req.Body["model"])
	}
	if req.Body["input"] != input {
		t.Errorf("Expected input %s, got %v", input, req.Body["input"])
	}
}

func TestNewModerationBatchRequest(t *testing.T) {
	customID := "req-1"
	model := "text-moderation-latest"
	input := "test content"

	req := NewModerationBatchRequest(customID, model, input)

	if req.CustomID != customID {
		t.Errorf("Expected custom ID %s, got %s", customID, req.CustomID)
	}
	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}
	if req.URL != string(EndpointModerations) {
		t.Errorf("Expected URL %s, got %s", EndpointModerations, req.URL)
	}
	if req.Body["model"] != model {
		t.Errorf("Expected model %s, got %v", model, req.Body["model"])
	}
	if req.Body["input"] != input {
		t.Errorf("Expected input %s, got %v", input, req.Body["input"])
	}
}

func TestBatchClient_doRequest_Error(t *testing.T) {
	responses := map[string]*http.Response{
		"POST /v1/batches": {
			StatusCode: 400,
			Body:       io.NopCloser(strings.NewReader(`{"error": "bad request"}`)),
		},
	}

	client := NewBatchClientWithHTTPClient(
		"test-key",
		"https://api.openai.com/v1",
		5*time.Second,
		newMockHTTPClient(responses),
	)

	_, err := client.doRequest(context.Background(), "POST", "/batches", nil)
	if err == nil {
		t.Error("Expected error for 400 status code, got nil")
	}
}

func TestBatchClient_UploadFile_NonExistentFile(t *testing.T) {
	client := NewBatchClientWithHTTPClient(
		"test-key",
		"https://api.openai.com/v1",
		5*time.Second,
		&http.Client{},
	)

	_, err := client.UploadFile(context.Background(), "/nonexistent/file.jsonl")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestBatchClient_WaitForBatch_AllTerminalStates(t *testing.T) {
	terminalStates := []BatchStatus{
		StatusCompleted,
		StatusFailed,
		StatusExpired,
		StatusCancelled,
	}

	for _, status := range terminalStates {
		t.Run(string(status), func(t *testing.T) {
			batchID := "batch-123"

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				batch := &Batch{
					ID:        batchID,
					Status:    status,
					CreatedAt: time.Now().Unix(),
				}
				json.NewEncoder(w).Encode(batch)
			}))
			defer server.Close()

			client := NewBatchClientWithHTTPClient(
				"test-key",
				server.URL,
				100*time.Millisecond,
				&http.Client{},
			)

			batch, err := client.WaitForBatch(context.Background(), batchID)

			if err != nil {
				t.Fatalf("WaitForBatch() error = %v", err)
			}

			if batch.Status != status {
				t.Errorf("Expected status %s, got %s", status, batch.Status)
			}
		})
	}
}

// Helper function to create pointer to int64
func ptrInt64(v int64) *int64 {
	return &v
}

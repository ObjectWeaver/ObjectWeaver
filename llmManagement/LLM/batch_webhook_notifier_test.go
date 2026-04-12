package LLM

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/client"
)

func TestNewBatchWebhookNotifierFromEnv_Disabled(t *testing.T) {
	// Clear env vars
	os.Unsetenv("LLM_BATCH_WEBHOOK_URL")
	os.Unsetenv("LLM_BATCH_WEBHOOK_API_KEY")

	notifier := NewBatchWebhookNotifierFromEnv()

	if notifier.IsEnabled() {
		t.Error("Expected notifier to be disabled when no webhook URL is set")
	}
}

func TestNewBatchWebhookNotifierFromEnv_Enabled(t *testing.T) {
	// Set env vars
	os.Setenv("LLM_BATCH_WEBHOOK_URL", "https://example.com/webhook")
	os.Setenv("LLM_BATCH_WEBHOOK_API_KEY", "test-key")
	defer os.Unsetenv("LLM_BATCH_WEBHOOK_URL")
	defer os.Unsetenv("LLM_BATCH_WEBHOOK_API_KEY")

	notifier := NewBatchWebhookNotifierFromEnv()

	if !notifier.IsEnabled() {
		t.Error("Expected notifier to be enabled when webhook URL is set")
	}
}

func TestNewBatchWebhookNotifier(t *testing.T) {
	config := &BatchWebhookConfig{
		WebhookURL:    "https://example.com/webhook",
		APIKey:        "test-key",
		RetryAttempts: 3,
		RetryDelay:    5 * time.Second,
	}

	notifier := NewBatchWebhookNotifier(config)

	if !notifier.IsEnabled() {
		t.Error("Expected notifier to be enabled")
	}

	impl, ok := notifier.(*DefaultBatchWebhookNotifier)
	if !ok {
		t.Fatal("Expected DefaultBatchWebhookNotifier implementation")
	}

	if impl.webhookURL != "https://example.com/webhook" {
		t.Errorf("Expected webhook URL 'https://example.com/webhook', got '%s'", impl.webhookURL)
	}

	if impl.apiKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", impl.apiKey)
	}

	if impl.retryAttempts != 3 {
		t.Errorf("Expected 3 retry attempts, got %d", impl.retryAttempts)
	}
}

func TestNotifyBatchComplete_Success(t *testing.T) {
	var receivedPayload *BatchWebhookPayload
	var receivedHeaders http.Header

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture headers
		receivedHeaders = r.Header.Clone()

		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Verify API key
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected Authorization 'Bearer test-api-key', got '%s'", r.Header.Get("Authorization"))
		}

		// Read and parse body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		var payload BatchWebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("Failed to parse webhook payload: %v", err)
		}

		receivedPayload = &payload

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create notifier
	config := &BatchWebhookConfig{
		WebhookURL:    server.URL,
		APIKey:        "test-api-key",
		RetryAttempts: 2,
		RetryDelay:    100 * time.Millisecond,
	}
	notifier := NewBatchWebhookNotifier(config)

	// Create test data
	completedAt := time.Now().Unix()
	batch := &client.Batch{
		ID:          "batch_123",
		Status:      client.StatusCompleted,
		CompletedAt: &completedAt,
		RequestCounts: client.RequestCounts{
			Total:     5,
			Completed: 5,
			Failed:    0,
		},
	}

	responses := []client.BatchResponse{
		{
			CustomID: "job-1",
			Response: &client.ResponseData{
				StatusCode: 200,
				Body: map[string]interface{}{
					"id": "chatcmpl-123",
					"choices": []interface{}{
						map[string]interface{}{
							"message": map[string]interface{}{
								"content": "Hello",
							},
						},
					},
				},
			},
		},
		{
			CustomID: "job-2",
			Error: &client.BatchError{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded",
			},
		},
	}

	jobs := []*BatchJobEntry{
		{
			Job:      &Job{},
			CustomID: "job-1",
		},
		{
			Job:      &Job{},
			CustomID: "job-2",
		},
	}

	// Send notification
	ctx := context.Background()
	err := notifier.NotifyBatchComplete(ctx, batch, responses, jobs)
	if err != nil {
		t.Fatalf("NotifyBatchComplete failed: %v", err)
	}

	// Verify payload
	if receivedPayload == nil {
		t.Fatal("No payload received")
	}

	if receivedPayload.BatchID != "batch_123" {
		t.Errorf("Expected batch ID 'batch_123', got '%s'", receivedPayload.BatchID)
	}

	if receivedPayload.JobCount != 2 {
		t.Errorf("Expected job count 2, got %d", receivedPayload.JobCount)
	}

	if len(receivedPayload.Responses) != 2 {
		t.Fatalf("Expected 2 responses, got %d", len(receivedPayload.Responses))
	}

	// Verify first response (success)
	if !receivedPayload.Responses[0].Success {
		t.Error("Expected first response to be successful")
	}

	if receivedPayload.Responses[0].CustomID != "job-1" {
		t.Errorf("Expected custom ID 'job-1', got '%s'", receivedPayload.Responses[0].CustomID)
	}

	// Verify second response (error)
	if receivedPayload.Responses[1].Success {
		t.Error("Expected second response to be failed")
	}

	if receivedPayload.Responses[1].Error == nil {
		t.Error("Expected second response to have error")
	} else if receivedPayload.Responses[1].Error.Code != "rate_limit_exceeded" {
		t.Errorf("Expected error code 'rate_limit_exceeded', got '%s'", receivedPayload.Responses[1].Error.Code)
	}

	// Verify custom headers
	if receivedHeaders.Get("X-Batch-ID") != "batch_123" {
		t.Errorf("Expected X-Batch-ID header 'batch_123', got '%s'", receivedHeaders.Get("X-Batch-ID"))
	}

	if receivedHeaders.Get("X-Job-Count") != "2" {
		t.Errorf("Expected X-Job-Count header '2', got '%s'", receivedHeaders.Get("X-Job-Count"))
	}
}

func TestNotifyBatchComplete_Retry(t *testing.T) {
	attemptCount := 0

	// Create test server that fails first 2 attempts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create notifier with retries
	config := &BatchWebhookConfig{
		WebhookURL:    server.URL,
		APIKey:        "test-key",
		RetryAttempts: 3,
		RetryDelay:    50 * time.Millisecond,
	}
	notifier := NewBatchWebhookNotifier(config)

	// Create minimal test data
	batch := &client.Batch{
		ID:     "batch_123",
		Status: client.StatusCompleted,
	}

	ctx := context.Background()
	err := notifier.NotifyBatchComplete(ctx, batch, []client.BatchResponse{}, []*BatchJobEntry{})

	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestNotifyBatchComplete_MaxRetriesExceeded(t *testing.T) {
	attemptCount := 0

	// Create test server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	// Create notifier with limited retries
	config := &BatchWebhookConfig{
		WebhookURL:    server.URL,
		APIKey:        "test-key",
		RetryAttempts: 2,
		RetryDelay:    10 * time.Millisecond,
	}
	notifier := NewBatchWebhookNotifier(config)

	// Create minimal test data
	batch := &client.Batch{
		ID:     "batch_123",
		Status: client.StatusCompleted,
	}

	ctx := context.Background()
	err := notifier.NotifyBatchComplete(ctx, batch, []client.BatchResponse{}, []*BatchJobEntry{})

	if err == nil {
		t.Fatal("Expected error after max retries exceeded")
	}

	expectedAttempts := 3 // Initial attempt + 2 retries
	if attemptCount != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attemptCount)
	}
}

func TestNotifyBatchComplete_Disabled(t *testing.T) {
	// Create disabled notifier
	notifier := &DefaultBatchWebhookNotifier{enabled: false}

	// Create minimal test data
	batch := &client.Batch{
		ID:     "batch_123",
		Status: client.StatusCompleted,
	}

	ctx := context.Background()
	err := notifier.NotifyBatchComplete(ctx, batch, []client.BatchResponse{}, []*BatchJobEntry{})

	if err != nil {
		t.Errorf("Expected no error for disabled notifier, got: %v", err)
	}
}

func TestNotifyBatchComplete_ContextCancellation(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create notifier
	config := &BatchWebhookConfig{
		WebhookURL:    server.URL,
		RetryAttempts: 5,
		RetryDelay:    50 * time.Millisecond,
	}
	notifier := NewBatchWebhookNotifier(config)

	// Create minimal test data
	batch := &client.Batch{
		ID:     "batch_123",
		Status: client.StatusCompleted,
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := notifier.NotifyBatchComplete(ctx, batch, []client.BatchResponse{}, []*BatchJobEntry{})

	if err == nil {
		t.Fatal("Expected error due to context cancellation")
	}
}

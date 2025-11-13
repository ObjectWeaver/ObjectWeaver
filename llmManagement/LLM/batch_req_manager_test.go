package LLM

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"objectweaver/llmManagement"
	"objectweaver/llmManagement/client"
	"objectweaver/llmManagement/domain"
) // mockRoundTripper implements http.RoundTripper for testing
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.roundTripFunc != nil {
		return m.roundTripFunc(req)
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

// createMockBatchClient creates a batch client with mocked HTTP responses
func createMockBatchClient(roundTripFunc func(req *http.Request) (*http.Response, error)) *client.BatchClient {
	httpClient := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: roundTripFunc,
		},
	}
	return client.NewBatchClientWithHTTPClient("test-api-key", "https://api.test.com", 100*time.Millisecond, httpClient)
}

// defaultMockBatchClient creates a batch client that succeeds for all operations
func defaultMockBatchClient() *client.BatchClient {
	return createMockBatchClient(func(req *http.Request) (*http.Response, error) {
		// Upload file
		if strings.HasSuffix(req.URL.Path, "/files") && req.Method == "POST" {
			file := client.File{
				ID:       "file-123",
				Object:   "file",
				Bytes:    1024,
				Filename: "test.jsonl",
				Purpose:  "batch",
			}
			body, _ := json.Marshal(file)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}

		// Create batch
		if strings.HasSuffix(req.URL.Path, "/batches") && req.Method == "POST" {
			batch := client.Batch{
				ID:     "batch-123",
				Status: client.StatusInProgress,
			}
			body, _ := json.Marshal(batch)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}

		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Header:     make(http.Header),
		}, nil
	})
}

func TestNewBatchReqManager(t *testing.T) {
	mockClient := defaultMockBatchClient()
	manager, err := NewBatchReqManager(mockClient)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.PendingJobs != 0 {
		t.Errorf("Expected 0 pending jobs, got %d", stats.PendingJobs)
	}
}

func TestNewBatchReqManager_NilClient(t *testing.T) {
	config := &BatchManagerConfig{
		BatchClient: nil,
	}

	manager, err := NewBatchReqManagerWithConfig(config)

	if err == nil {
		t.Error("Expected error when batch client is nil")
	}

	if manager != nil {
		t.Error("Expected nil manager when error occurs")
	}
}

func TestAddJob(t *testing.T) {
	mockClient := defaultMockBatchClient()
	config := &BatchManagerConfig{
		MaxRequestsPerBatch: 10,
		MaxMemoryBytes:      10 * 1024 * 1024,
		FlushInterval:       1 * time.Minute,
		BatchClient:         mockClient,
	}

	manager, err := NewBatchReqManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	job := &Job{
		Inputs: &llmManagement.Inputs{
			Prompt:       "Test prompt",
			SystemPrompt: "System prompt",
		},
		Result: make(chan *domain.JobResult, 1),
		Error:  make(chan error, 1),
	}

	err = manager.AddJob(job)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	stats := manager.GetStats()
	if stats.PendingJobs != 1 {
		t.Errorf("Expected 1 pending job, got %d", stats.PendingJobs)
	}
}

func TestAddJob_Nil(t *testing.T) {
	mockClient := defaultMockBatchClient()
	manager, _ := NewBatchReqManager(mockClient)

	err := manager.AddJob(nil)
	if err == nil {
		t.Error("Expected error when adding nil job")
	}
}

func TestGetStats(t *testing.T) {
	mockClient := defaultMockBatchClient()
	manager, _ := NewBatchReqManager(mockClient)

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.PendingJobs != 0 {
		t.Errorf("Expected 0 pending jobs, got %d", stats.PendingJobs)
	}
}

func TestEstimateJobMemory(t *testing.T) {
	job := &Job{
		Inputs: &llmManagement.Inputs{
			Prompt:       "Hello",
			SystemPrompt: "System",
		},
	}

	memory := estimateJobMemory(job)
	if memory < 1024 {
		t.Errorf("Expected at least 1024 bytes, got %d", memory)
	}
}

func TestGetIntFromEnv(t *testing.T) {
	os.Setenv("TEST_BATCH_INT", "42")
	defer os.Unsetenv("TEST_BATCH_INT")

	got := getIntFromEnv("TEST_BATCH_INT", 10)
	if got != 42 {
		t.Errorf("Expected 42, got %d", got)
	}

	got = getIntFromEnv("NONEXISTENT_KEY", 10)
	if got != 10 {
		t.Errorf("Expected default 10, got %d", got)
	}
}

func TestBatchManagerStats(t *testing.T) {
	mockClient := defaultMockBatchClient()
	manager, _ := NewBatchReqManager(mockClient)

	for i := 0; i < 3; i++ {
		job := &Job{
			Inputs: &llmManagement.Inputs{
				Prompt:       fmt.Sprintf("Prompt %d", i),
				SystemPrompt: "System",
			},
			Result: make(chan *domain.JobResult, 1),
			Error:  make(chan error, 1),
		}
		manager.AddJob(job)
	}

	stats := manager.GetStats()

	if stats.PendingJobs != 3 {
		t.Errorf("Expected 3 pending jobs, got %d", stats.PendingJobs)
	}

	if stats.TotalJobsQueued != 3 {
		t.Errorf("Expected 3 total jobs queued, got %d", stats.TotalJobsQueued)
	}
}

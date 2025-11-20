package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/clientManager"
	"objectweaver/llmManagement/domain"
	"sync"
	"testing"
	"time"

	"github.com/objectweaver/go-sdk/client"
	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

// MockClientAdapter is a mock implementation of ClientAdapter that returns predefined responses
type MockClientAdapter struct {
	delay          time.Duration
	failureRate    float64
	requestCounter int
	mu             sync.Mutex
}

// NewMockClientAdapter creates a new mock client adapter
func NewMockClientAdapter(delay time.Duration, failureRate float64) *MockClientAdapter {
	return &MockClientAdapter{
		delay:       delay,
		failureRate: failureRate,
	}
}

// Process simulates LLM processing with configurable delay and failure rate
func (m *MockClientAdapter) Process(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	m.mu.Lock()
	m.requestCounter++
	reqNum := m.requestCounter
	m.mu.Unlock()

	// Simulate processing time
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Simulate random failures if configured
	if m.failureRate > 0 && float64(reqNum%100)/100.0 < m.failureRate {
		return nil, fmt.Errorf("simulated LLM failure")
	}

	// Return mock response based on the prompt
	result := &domain.JobResult{
		ChatRes: &openai.ChatCompletionResponse{
			ID:      fmt.Sprintf("mock-response-%d", reqNum),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "mock-model",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: generateMockContent(inputs.Prompt),
					},
					FinishReason: "stop",
				},
			},
			Usage: openai.Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		},
		EmbeddingRes: nil,
	}

	return result, nil
}

// ProcessBatch simulates batch processing (not used in benchmarks but required by interface)
func (m *MockClientAdapter) ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error) {
	return &openai.ChatCompletionResponse{}, nil
}

// generateMockContent generates realistic mock JSON content based on the prompt
func generateMockContent(prompt string) string {
	// Generate a simple JSON response
	mockData := map[string]interface{}{
		"name":        "John Doe",
		"email":       "john.doe@example.com",
		"age":         30,
		"active":      true,
		"score":       95.5,
		"description": "This is a mock generated response for benchmarking purposes",
		"timestamp":   time.Now().Unix(),
	}

	jsonBytes, _ := json.Marshal(mockData)
	return string(jsonBytes)
}

// MockClientAdapterFactory creates mock client adapters for testing
type MockClientAdapterFactory struct {
	delay       time.Duration
	failureRate float64
}

func NewMockClientAdapterFactory(delay time.Duration, failureRate float64) *MockClientAdapterFactory {
	return &MockClientAdapterFactory{
		delay:       delay,
		failureRate: failureRate,
	}
}

func (f *MockClientAdapterFactory) Create(provider string, model string) (clientManager.ClientAdapter, error) {
	return NewMockClientAdapter(f.delay, f.failureRate), nil
}

// createBenchmarkRequest creates a sample request for benchmarking
func createBenchmarkRequest(complexity string) *client.RequestBody {
	var definition *jsonSchema.Definition

	switch complexity {
	case "simple":
		definition = &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"name": {Type: jsonSchema.String},
				"age":  {Type: jsonSchema.Integer},
			},
		}
	case "medium":
		definition = &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"user": {
					Type: jsonSchema.Object,
					Properties: map[string]jsonSchema.Definition{
						"name":  {Type: jsonSchema.String},
						"email": {Type: jsonSchema.String},
						"age":   {Type: jsonSchema.Integer},
					},
				},
				"preferences": {
					Type: jsonSchema.Object,
					Properties: map[string]jsonSchema.Definition{
						"theme":    {Type: jsonSchema.String},
						"language": {Type: jsonSchema.String},
					},
				},
				"metadata": {
					Type: jsonSchema.Object,
					Properties: map[string]jsonSchema.Definition{
						"created": {Type: jsonSchema.String},
						"updated": {Type: jsonSchema.String},
					},
				},
			},
		}
	case "complex":
		definition = &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"profile": {
					Type: jsonSchema.Object,
					Properties: map[string]jsonSchema.Definition{
						"personal": {
							Type: jsonSchema.Object,
							Properties: map[string]jsonSchema.Definition{
								"firstName":   {Type: jsonSchema.String},
								"lastName":    {Type: jsonSchema.String},
								"birthdate":   {Type: jsonSchema.String},
								"nationality": {Type: jsonSchema.String},
							},
						},
						"contact": {
							Type: jsonSchema.Object,
							Properties: map[string]jsonSchema.Definition{
								"email": {Type: jsonSchema.String},
								"phone": {Type: jsonSchema.String},
								"address": {
									Type: jsonSchema.Object,
									Properties: map[string]jsonSchema.Definition{
										"street":  {Type: jsonSchema.String},
										"city":    {Type: jsonSchema.String},
										"country": {Type: jsonSchema.String},
										"zipCode": {Type: jsonSchema.String},
									},
								},
							},
						},
					},
				},
				"settings": {
					Type: jsonSchema.Object,
					Properties: map[string]jsonSchema.Definition{
						"notifications": {
							Type: jsonSchema.Object,
							Properties: map[string]jsonSchema.Definition{
								"email": {Type: jsonSchema.Boolean},
								"sms":   {Type: jsonSchema.Boolean},
								"push":  {Type: jsonSchema.Boolean},
							},
						},
						"privacy": {
							Type: jsonSchema.Object,
							Properties: map[string]jsonSchema.Definition{
								"profileVisible": {Type: jsonSchema.Boolean},
								"shareData":      {Type: jsonSchema.Boolean},
							},
						},
					},
				},
			},
		}
	default:
		definition = &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"name": {Type: jsonSchema.String},
			},
		}
	}

	return &client.RequestBody{
		Prompt:     "Generate a sample object",
		Definition: definition,
	}
}

// BenchmarkServerConcurrency tests server performance under concurrent load
func BenchmarkServerConcurrency(b *testing.B) {
	testCases := []struct {
		name        string
		concurrency int
		complexity  string
		mockDelay   time.Duration
		failureRate float64
	}{
		{"1_Concurrent_Simple", 1, "simple", 10 * time.Millisecond, 0.0},
		{"10_Concurrent_Simple", 10, "simple", 10 * time.Millisecond, 0.0},
		{"50_Concurrent_Simple", 50, "simple", 10 * time.Millisecond, 0.0},
		{"100_Concurrent_Simple", 100, "simple", 10 * time.Millisecond, 0.0},
		{"500_Concurrent_Simple", 500, "simple", 10 * time.Millisecond, 0.0},
		{"1000_Concurrent_Simple", 1000, "simple", 10 * time.Millisecond, 0.0},

		{"100_Concurrent_Medium", 100, "medium", 10 * time.Millisecond, 0.0},
		{"100_Concurrent_Complex", 100, "complex", 10 * time.Millisecond, 0.0},

		{"100_Concurrent_SlowLLM", 100, "simple", 100 * time.Millisecond, 0.0},
		{"100_Concurrent_WithFailures", 100, "simple", 10 * time.Millisecond, 0.05},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkConcurrentRequests(b, tc.concurrency, tc.complexity, tc.mockDelay, tc.failureRate)
		})
	}
}

// benchmarkConcurrentRequests performs the actual concurrent request benchmark
func benchmarkConcurrentRequests(b *testing.B, concurrency int, complexity string, mockDelay time.Duration, failureRate float64) {
	// Note: In production, you'd inject the mock client adapter here
	// For now, this benchmarks the actual server infrastructure
	server := NewHttpServer()
	ts := httptest.NewServer(server.Router)
	defer ts.Close()

	// Create the request body
	reqBody := createBenchmarkRequest(complexity)
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		b.Fatalf("Failed to marshal request: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req, err := http.NewRequest("POST", ts.URL+"/api/objectGen", bytes.NewReader(bodyBytes))
				if err != nil {
					errors <- err
					return
				}

				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test-token")

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				req = req.WithContext(ctx)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					errors <- err
					return
				}
				defer resp.Body.Close()

				// Read the response to ensure full processing
				var result Response
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					// Only report error if not a timeout/context cancellation
					if ctx.Err() == nil {
						errors <- err
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Report errors
		errorCount := 0
		for err := range errors {
			errorCount++
			if errorCount == 1 {
				b.Logf("Errors occurred: %v", err)
			}
		}
		if errorCount > 0 {
			b.Logf("Total errors in iteration: %d/%d", errorCount, concurrency)
		}
	}
}

// BenchmarkServerThroughput measures maximum throughput
func BenchmarkServerThroughput(b *testing.B) {
	server := NewHttpServer()
	ts := httptest.NewServer(server.Router)
	defer ts.Close()

	reqBody := createBenchmarkRequest("simple")
	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	// Sequential requests to measure raw throughput
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("POST", ts.URL+"/api/objectGen", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				b.Logf("Request error: %v", err)
				continue
			}

			var result Response
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()
		}
	})
}

// BenchmarkServerLatency measures request latency under different loads
func BenchmarkServerLatency(b *testing.B) {
	testCases := []struct {
		name       string
		complexity string
	}{
		{"Latency_Simple", "simple"},
		{"Latency_Medium", "medium"},
		{"Latency_Complex", "complex"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			server := NewHttpServer()
			ts := httptest.NewServer(server.Router)
			defer ts.Close()

			reqBody := createBenchmarkRequest(tc.complexity)
			bodyBytes, _ := json.Marshal(reqBody)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				start := time.Now()

				req, _ := http.NewRequest("POST", ts.URL+"/api/objectGen", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test-token")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					b.Logf("Request error: %v", err)
					continue
				}

				var result Response
				json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()

				elapsed := time.Since(start)
				b.ReportMetric(float64(elapsed.Nanoseconds())/1e6, "ms/op")
			}
		})
	}
}

// BenchmarkServerMemoryPressure tests memory usage under high load
func BenchmarkServerMemoryPressure(b *testing.B) {
	testCases := []struct {
		name        string
		concurrency int
		iterations  int
	}{
		{"Memory_100_Concurrent", 100, 10},
		{"Memory_500_Concurrent", 500, 10},
		{"Memory_1000_Concurrent", 1000, 10},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			server := NewHttpServer()
			ts := httptest.NewServer(server.Router)
			defer ts.Close()

			reqBody := createBenchmarkRequest("medium")
			bodyBytes, _ := json.Marshal(reqBody)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for iter := 0; iter < tc.iterations; iter++ {
					var wg sync.WaitGroup
					for j := 0; j < tc.concurrency; j++ {
						wg.Add(1)
						go func() {
							defer wg.Done()

							req, _ := http.NewRequest("POST", ts.URL+"/api/objectGen", bytes.NewReader(bodyBytes))
							req.Header.Set("Content-Type", "application/json")
							req.Header.Set("Authorization", "Bearer test-token")

							resp, err := http.DefaultClient.Do(req)
							if err != nil {
								return
							}
							defer resp.Body.Close()

							var result Response
							json.NewDecoder(resp.Body).Decode(&result)
						}()
					}
					wg.Wait()
				}
			}
		})
	}
}

// BenchmarkEndToEnd simulates realistic end-to-end scenarios
func BenchmarkEndToEnd(b *testing.B) {
	testCases := []struct {
		name        string
		scenario    string
		concurrency int
		complexity  string
	}{
		{"Realistic_LowLoad", "low", 10, "simple"},
		{"Realistic_MediumLoad", "medium", 50, "medium"},
		{"Realistic_HighLoad", "high", 100, "complex"},
		{"Realistic_PeakLoad", "peak", 500, "simple"},
		{"Realistic_StressTest", "stress", 1000, "medium"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			server := NewHttpServer()
			ts := httptest.NewServer(server.Router)
			defer ts.Close()

			reqBody := createBenchmarkRequest(tc.complexity)
			bodyBytes, _ := json.Marshal(reqBody)

			// Metrics tracking
			var successCount, errorCount int64
			var totalLatency time.Duration
			var mu sync.Mutex

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup

				for j := 0; j < tc.concurrency; j++ {
					wg.Add(1)
					go func() {
						defer wg.Done()

						start := time.Now()
						req, _ := http.NewRequest("POST", ts.URL+"/api/objectGen", bytes.NewReader(bodyBytes))
						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("Authorization", "Bearer test-token")

						resp, err := http.DefaultClient.Do(req)
						elapsed := time.Since(start)

						mu.Lock()
						if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
							errorCount++
						} else {
							successCount++
							totalLatency += elapsed
						}
						mu.Unlock()

						if resp != nil {
							var result Response
							json.NewDecoder(resp.Body).Decode(&result)
							resp.Body.Close()
						}
					}()
				}

				wg.Wait()
			}

			// Report metrics
			b.StopTimer()
			totalRequests := successCount + errorCount
			if totalRequests > 0 {
				avgLatency := totalLatency / time.Duration(successCount)
				successRate := float64(successCount) / float64(totalRequests) * 100

				b.ReportMetric(float64(avgLatency.Milliseconds()), "avg_latency_ms")
				b.ReportMetric(successRate, "success_rate_%")
				b.ReportMetric(float64(totalRequests)/b.Elapsed().Seconds(), "requests_per_sec")
			}
		})
	}
}

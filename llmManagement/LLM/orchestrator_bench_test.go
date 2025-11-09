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
package LLM

import (
	"fmt"
	"objectweaver/llmManagement"
	"objectweaver/llmManagement/backoff"
	"objectweaver/llmManagement/domain"
	"sync"
	"testing"
	"time"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

// MockClientAdapter is a test adapter that doesn't hit real APIs
type MockClientAdapter struct {
	processDelay time.Duration
}

func NewMockClientAdapter(delay time.Duration) *MockClientAdapter {
	return &MockClientAdapter{processDelay: delay}
}

func (m *MockClientAdapter) Process(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	if m.processDelay > 0 {
		time.Sleep(m.processDelay)
	}

	chatRes := &openai.ChatCompletionResponse{
		ID:      "mock-response",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "mock response content",
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

	return domain.CreateJobResult(chatRes, nil), nil
}

func (m *MockClientAdapter) ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error) {
	return nil, nil
}

// BenchmarkJobQueue tests the job queue performance
func BenchmarkJobQueue(b *testing.B) {
	tests := []struct {
		name         string
		queueType    QueueType
		numWorkers   int
		numJobs      int
		processDelay time.Duration
	}{
		{"FIFO_1Worker_100Jobs_NoDelay", QueueTypeFIFO, 1, 100, 0},
		{"FIFO_4Workers_100Jobs_NoDelay", QueueTypeFIFO, 4, 100, 0},
		{"FIFO_10Workers_100Jobs_NoDelay", QueueTypeFIFO, 10, 100, 0},
		{"Priority_4Workers_100Jobs_NoDelay", QueueTypePriority, 4, 100, 0},
		{"FIFO_4Workers_100Jobs_1msDelay", QueueTypeFIFO, 4, 100, time.Millisecond},
		{"FIFO_10Workers_1000Jobs_NoDelay", QueueTypeFIFO, 10, 1000, 0},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				b.StopTimer()

				// Setup
				mockAdapter := NewMockClientAdapter(tt.processDelay)
				queue := NewJobQueueByType(tt.queueType)
				queueManager := NewJobQueueManager(tt.numWorkers, 1000, queue)

				var wg sync.WaitGroup
				wg.Add(1)
				go queueManager.StartManager(&wg)

				// Create worker pool
				var workerWg sync.WaitGroup
				for w := 0; w < tt.numWorkers; w++ {
					workerWg.Add(1)
					go func(workerID int) {
						defer workerWg.Done()
						for job := range queueManager.Jobs() {
							resp, err := mockAdapter.Process(job.Inputs)
							if err != nil {
								job.Error <- err
							} else {
								job.Result <- resp
							}
						}
					}(w)
				}

				b.StartTimer()

				// Submit jobs
				for j := 0; j < tt.numJobs; j++ {
					job := &Job{
						Inputs: &llmManagement.Inputs{
							Prompt:       "test prompt",
							SystemPrompt: "test system",
							Def:          &jsonSchema.Definition{Type: jsonSchema.String},
							Priority:     int32(j % 10), // Vary priorities
						},
						Result: make(chan *domain.JobResult, 1),
						Error:  make(chan error, 1),
						Tokens: 30,
					}
					queueManager.Enqueue(job)

					// Consume result
					select {
					case <-job.Result:
					case <-job.Error:
					}
				}

				b.StopTimer()

				// Cleanup
				queueManager.StopManager()
				workerWg.Wait()
				wg.Wait()
			}
		})
	}
}

// BenchmarkOrchestrator tests the full orchestrator with rate limiting
func BenchmarkOrchestrator(b *testing.B) {
	tests := []struct {
		name            string
		concurrency     int
		numJobs         int
		maxTokensPerMin int
		maxReqPerMin    int
		backoffStrategy backoff.BackoffStrategy
	}{
		{"4Workers_100Jobs_NoRateLimit", 4, 100, 1000000, 1000000, backoff.BackoffNone},
		{"10Workers_100Jobs_NoRateLimit", 10, 100, 1000000, 1000000, backoff.BackoffNone},
		{"4Workers_100Jobs_RateLimit", 4, 100, 10000, 100, backoff.BackoffNone},
		{"10Workers_1000Jobs_NoRateLimit", 10, 1000, 1000000, 1000000, backoff.BackoffNone},
		{"10Workers_100Jobs_GlobalBackoff", 10, 100, 1000000, 1000000, backoff.BackoffGlobalExponential},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				b.StopTimer()

				// Setup orchestrator
				mockAdapter := NewMockClientAdapter(0)
				var wg sync.WaitGroup

				config := OrchestratorConfig{
					Concurrency:          tt.concurrency,
					MaxTokensPerMinute:   tt.maxTokensPerMin,
					MaxRequestsPerMinute: tt.maxReqPerMin,
					MaxQueueSize:         1000,
					Verbose:              false,
				}

				jobQueue := NewJobQueueByType(QueueTypePriority)
				queueManager := NewJobQueueManager(tt.concurrency, config.MaxQueueSize, jobQueue)
				errorClassifier := backoff.NewErrorClassifier()
				retryHandler := NewRetryHandler(3, false)

				var backoffManager BackoffManager
				switch tt.backoffStrategy {
				case backoff.BackoffGlobalExponential:
					backoffManager = backoff.NewGlobalExponentialBackoff(1*time.Minute, false)
				case backoff.BackoffPerWorkerExponential:
					backoffManager = backoff.NewPerWorkerExponentialBackoff(1*time.Minute, tt.concurrency, false)
				default:
					backoffManager = &backoff.NoBackoff{Verbose: false}
				}

				orchestrator := NewOrchestrator(
					config,
					mockAdapter,
					queueManager,
					backoffManager,
					retryHandler,
					errorClassifier,
				)

				orchestrator.StartProcessing()

				b.StartTimer()

				// Submit jobs
				for j := 0; j < tt.numJobs; j++ {
					job := &Job{
						Inputs: &llmManagement.Inputs{
							Prompt:       "benchmark prompt",
							SystemPrompt: "system",
							Def:          &jsonSchema.Definition{Type: jsonSchema.String},
							Priority:     int32(j % 10),
						},
						Result: make(chan *domain.JobResult, 1),
						Error:  make(chan error, 1),
						Tokens: 30,
					}

					queueManager.Enqueue(job)

					// Consume result
					select {
					case <-job.Result:
					case <-job.Error:
					}
				}

				b.StopTimer()

				// Cleanup
				orchestrator.Stop()
				wg.Wait()
			}
		})
	}
}

// BenchmarkConcurrentJobSubmission tests concurrent job submissions
func BenchmarkConcurrentJobSubmission(b *testing.B) {
	concurrencyLevels := []int{1, 5, 10, 25, 50}

	for _, numGoroutines := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines_%d", numGoroutines), func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				b.StopTimer()

				mockAdapter := NewMockClientAdapter(0)
				var wg sync.WaitGroup

				config := OrchestratorConfig{
					Concurrency:          10,
					MaxTokensPerMinute:   1000000,
					MaxRequestsPerMinute: 1000000,
					MaxQueueSize:         1000,
					Verbose:              false,
				}

				jobQueue := NewJobQueueByType(QueueTypePriority)
				queueManager := NewJobQueueManager(config.Concurrency, config.MaxQueueSize, jobQueue)
				errorClassifier := backoff.NewErrorClassifier()
				retryHandler := NewRetryHandler(3, false)
				backoffManager := &backoff.NoBackoff{Verbose: false}

				orchestrator := NewOrchestrator(
					config,
					mockAdapter,
					queueManager,
					backoffManager,
					retryHandler,
					errorClassifier,
				)

				orchestrator.StartProcessing()

				jobsPerGoroutine := 100

				b.StartTimer()

				// Submit jobs from multiple goroutines
				var submitWg sync.WaitGroup
				for g := 0; g < numGoroutines; g++ {
					submitWg.Add(1)
					go func(goroutineID int) {
						defer submitWg.Done()

						for j := 0; j < jobsPerGoroutine; j++ {
							job := &Job{
								Inputs: &llmManagement.Inputs{
									Prompt:       "concurrent benchmark",
									SystemPrompt: "system",
									Def:          &jsonSchema.Definition{Type: jsonSchema.String},
									Priority:     int32(j % 10),
								},
								Result: make(chan *domain.JobResult, 1),
								Error:  make(chan error, 1),
								Tokens: 30,
							}

							queueManager.Enqueue(job)

							select {
							case <-job.Result:
							case <-job.Error:
							}
						}
					}(g)
				}

				submitWg.Wait()

				b.StopTimer()

				orchestrator.Stop()
				wg.Wait()
			}
		})
	}
}

// BenchmarkQueueTypes compares different queue implementations
func BenchmarkQueueTypes(b *testing.B) {
	queueTypes := []QueueType{QueueTypeFIFO, QueueTypePriority}

	for _, queueType := range queueTypes {
		b.Run(string(queueType), func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				b.StopTimer()

				queue := NewJobQueueByType(queueType)
				queueManager := NewJobQueueManager(4, 1000, queue)

				var wg sync.WaitGroup
				wg.Add(1)
				go queueManager.StartManager(&wg)

				mockAdapter := NewMockClientAdapter(0)

				var workerWg sync.WaitGroup
				for w := 0; w < 4; w++ {
					workerWg.Add(1)
					go func() {
						defer workerWg.Done()
						for job := range queueManager.Jobs() {
							resp, err := mockAdapter.Process(job.Inputs)
							if err != nil {
								job.Error <- err
							} else {
								job.Result <- resp
							}
						}
					}()
				}

				b.StartTimer()

				numJobs := 200
				for j := 0; j < numJobs; j++ {
					job := &Job{
						Inputs: &llmManagement.Inputs{
							Prompt:   "test",
							Def:      &jsonSchema.Definition{Type: jsonSchema.String},
							Priority: int32(j % 20),
						},
						Result: make(chan *domain.JobResult, 1),
						Error:  make(chan error, 1),
						Tokens: 30,
					}

					queueManager.Enqueue(job)

					select {
					case <-job.Result:
					case <-job.Error:
					}
				}

				b.StopTimer()

				queueManager.StopManager()
				workerWg.Wait()
				wg.Wait()
			}
		})
	}
}

package LLM

import (
	"context"
	"errors"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/backoff"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/domain"
	"sync"
	"testing"
	"time"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

	"github.com/sashabaranov/go-openai"
)

// --- Mock Implementations ---

// mockClientAdapter is a mock implementation of clientManager.ClientAdapter
type mockClientAdapter struct {
	mu                sync.Mutex
	processFunc       func(*llmManagement.Inputs) (*domain.JobResult, error)
	processBatchFunc  func([]any) (*openai.ChatCompletionResponse, error)
	processCalls      int
	processBatchCalls int
}

func newMockClientAdapter() *mockClientAdapter {
	return &mockClientAdapter{
		processFunc: func(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
			// Default: successful response
			return &domain.JobResult{
				ChatRes: &openai.ChatCompletionResponse{
					ID: "test-response",
				},
			}, nil
		},
		processBatchFunc: func(jobs []any) (*openai.ChatCompletionResponse, error) {
			return &openai.ChatCompletionResponse{}, nil
		},
	}
}

func (m *mockClientAdapter) Process(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
	m.mu.Lock()
	m.processCalls++
	m.mu.Unlock()
	return m.processFunc(inputs)
}

func (m *mockClientAdapter) ProcessBatch(jobs []any) (*openai.ChatCompletionResponse, error) {
	m.mu.Lock()
	m.processBatchCalls++
	m.mu.Unlock()
	return m.processBatchFunc(jobs)
}

func (m *mockClientAdapter) getProcessCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.processCalls
}

// mockOrchestratorJobQueue is a mock implementation of IJobQueueManager for orchestrator tests
type mockOrchestratorJobQueue struct {
	mu         sync.Mutex
	jobs       []*Job
	jobChan    chan *Job
	stopChan   chan struct{}
	enqueueLog []*Job
}

func newMockOrchestratorJobQueue() *mockOrchestratorJobQueue {
	return &mockOrchestratorJobQueue{
		jobs:       make([]*Job, 0),
		jobChan:    make(chan *Job, 10),
		stopChan:   make(chan struct{}),
		enqueueLog: make([]*Job, 0),
	}
}

func (m *mockOrchestratorJobQueue) StartManager(wg *sync.WaitGroup) {
	defer wg.Done()
	<-m.stopChan
}

func (m *mockOrchestratorJobQueue) Jobs() <-chan *Job {
	return m.jobChan
}

func (m *mockOrchestratorJobQueue) StopManager() {
	close(m.stopChan)
	close(m.jobChan)
}

func (m *mockOrchestratorJobQueue) Enqueue(job *Job) {
	m.mu.Lock()
	m.jobs = append(m.jobs, job)
	m.enqueueLog = append(m.enqueueLog, job)
	m.mu.Unlock()

	select {
	case m.jobChan <- job:
	default:
	}
}

func (m *mockOrchestratorJobQueue) Dequeue() *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.jobs) == 0 {
		return nil
	}
	job := m.jobs[0]
	m.jobs = m.jobs[1:]
	return job
}

func (m *mockOrchestratorJobQueue) getEnqueueCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.enqueueLog)
}

// mockBackoffManager is a mock implementation of BackoffManager
type mockBackoffManager struct {
	mu                   sync.Mutex
	applyBackoffCalls    map[int]int
	activateBackoffCalls map[int]int
	resetBackoffCalls    map[int]int
	backoffDurations     map[int]time.Duration
}

func newMockBackoffManager() *mockBackoffManager {
	return &mockBackoffManager{
		applyBackoffCalls:    make(map[int]int),
		activateBackoffCalls: make(map[int]int),
		resetBackoffCalls:    make(map[int]int),
		backoffDurations:     make(map[int]time.Duration),
	}
}

func (m *mockBackoffManager) ApplyBackoff(workerID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyBackoffCalls[workerID]++
	if duration, ok := m.backoffDurations[workerID]; ok && duration > 0 {
		time.Sleep(duration)
	}
}

func (m *mockBackoffManager) ActivateBackoff(workerID int, retryAfter time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activateBackoffCalls[workerID]++
	m.backoffDurations[workerID] = retryAfter
}

func (m *mockBackoffManager) ResetBackoff(workerID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetBackoffCalls[workerID]++
	m.backoffDurations[workerID] = 0
}

func (m *mockBackoffManager) getActivateCount(workerID int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activateBackoffCalls[workerID]
}

func (m *mockBackoffManager) getResetCount(workerID int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resetBackoffCalls[workerID]
}

// mockBatchReqManager is a mock implementation of IBatchReqManager
type mockBatchReqManager struct {
	mu              sync.Mutex
	jobs            []*Job
	started         bool
	stopped         bool
	addJobCalls     int
	flushBatchCalls int
	addJobError     error
}

func newMockBatchReqManager() *mockBatchReqManager {
	return &mockBatchReqManager{
		jobs: make([]*Job, 0),
	}
}

func (m *mockBatchReqManager) AddJob(job *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addJobCalls++
	if m.addJobError != nil {
		return m.addJobError
	}
	m.jobs = append(m.jobs, job)
	return nil
}

func (m *mockBatchReqManager) FlushBatch(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushBatchCalls++
	return nil
}

func (m *mockBatchReqManager) Start(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
}

func (m *mockBatchReqManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	return nil
}

func (m *mockBatchReqManager) GetStats() *BatchManagerStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &BatchManagerStats{
		PendingJobs:      len(m.jobs),
		TotalJobsQueued:  int64(m.addJobCalls),
		TotalBatchesSent: int64(m.flushBatchCalls),
	}
}

func (m *mockBatchReqManager) getAddJobCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addJobCalls
}

func (m *mockBatchReqManager) isStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

func (m *mockBatchReqManager) isStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

// --- Test Cases ---

func TestNewOrchestrator(t *testing.T) {
	config := OrchestratorConfig{
		Concurrency:          5,
		MaxTokensPerMinute:   10000,
		MaxRequestsPerMinute: 100,
		MaxQueueSize:         50,
		Verbose:              false,
	}

	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	if orchestrator == nil {
		t.Fatal("Expected non-nil orchestrator")
	}

	if orchestrator.config.Concurrency != config.Concurrency {
		t.Errorf("Expected concurrency %d, got %d", config.Concurrency, orchestrator.config.Concurrency)
	}

	if orchestrator.clientAdapter != clientAdapter {
		t.Error("Client adapter not set correctly")
	}

	if orchestrator.jobQueue != queue {
		t.Error("Job queue not set correctly")
	}

	if orchestrator.backoffManager != backoffManager {
		t.Error("Backoff manager not set correctly")
	}

	if orchestrator.retryHandler != retryHandler {
		t.Error("Retry handler not set correctly")
	}

	if orchestrator.errorClassifier != classifier {
		t.Error("Error classifier not set correctly")
	}

	if orchestrator.requestLimiter == nil {
		t.Error("Request limiter not initialized")
	}

	if orchestrator.tokenLimiter == nil {
		t.Error("Token limiter not initialized")
	}

	if orchestrator.wg == nil {
		t.Error("Wait group not initialized")
	}
}

func TestNewOrchestrator_ZeroRateLimits(t *testing.T) {
	config := OrchestratorConfig{
		Concurrency:          1,
		MaxTokensPerMinute:   0, // Should default to 1
		MaxRequestsPerMinute: 0, // Should default to 1
		MaxQueueSize:         10,
	}

	orchestrator := NewOrchestrator(
		config,
		newMockClientAdapter(),
		newMockOrchestratorJobQueue(),
		newMockBackoffManager(),
		NewRetryHandler(3, false),
		backoff.NewErrorClassifier(),
	)

	if orchestrator.requestLimiter == nil {
		t.Error("Request limiter should be initialized even with zero rate")
	}

	if orchestrator.tokenLimiter == nil {
		t.Error("Token limiter should be initialized even with zero rate")
	}
}

func TestSetBatchManager(t *testing.T) {
	orchestrator := NewOrchestrator(
		OrchestratorConfig{Concurrency: 1},
		newMockClientAdapter(),
		newMockOrchestratorJobQueue(),
		newMockBackoffManager(),
		NewRetryHandler(3, false),
		backoff.NewErrorClassifier(),
	)

	if orchestrator.batchManager != nil {
		t.Error("Batch manager should be nil initially")
	}

	batchManager := newMockBatchReqManager()
	orchestrator.SetBatchManager(batchManager)

	if orchestrator.batchManager != batchManager {
		t.Error("Batch manager not set correctly")
	}
}

func TestOrchestrationJob_Success(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:          1,
		MaxTokensPerMinute:   60000,
		MaxRequestsPerMinute: 6000,
		MaxQueueSize:         10,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	// Create a job
	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{
			Def: &jsonSchema.Definition{},
		},
	}

	// Process the job
	orchestrator.orchestrationJob(job, 0)

	// Verify result received
	select {
	case result := <-job.Result:
		if result == nil {
			t.Error("Expected non-nil result")
		}
		if result.ChatRes == nil {
			t.Error("Expected chat response")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for result")
	}

	// Verify backoff was reset
	if backoffManager.getResetCount(0) != 1 {
		t.Errorf("Expected backoff reset to be called once, got %d", backoffManager.getResetCount(0))
	}

	// Verify client adapter was called
	if clientAdapter.getProcessCalls() != 1 {
		t.Errorf("Expected 1 process call, got %d", clientAdapter.getProcessCalls())
	}
}

func TestOrchestrationJob_RateLimitError(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	clientAdapter.processFunc = func(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
		return nil, &backoff.RateLimitError{RetryAfter: 100 * time.Millisecond}
	}

	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:          1,
		MaxTokensPerMinute:   60000,
		MaxRequestsPerMinute: 6000,
		MaxQueueSize:         10,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{
			Def: &jsonSchema.Definition{},
		},
	}

	// Process the job
	orchestrator.orchestrationJob(job, 0)

	// Verify backoff was activated
	if backoffManager.getActivateCount(0) != 1 {
		t.Errorf("Expected backoff activate to be called once, got %d", backoffManager.getActivateCount(0))
	}

	// Verify job was re-enqueued
	if queue.getEnqueueCount() != 1 {
		t.Errorf("Expected 1 enqueue, got %d", queue.getEnqueueCount())
	}

	// Verify no result was sent
	select {
	case <-job.Result:
		t.Error("Should not receive result for rate limit error")
	default:
		// Expected
	}
}

func TestOrchestrationJob_TransientError(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	clientAdapter.processFunc = func(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
		return nil, errors.New("Service Unavailable")
	}

	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:          1,
		MaxTokensPerMinute:   60000,
		MaxRequestsPerMinute: 6000,
		MaxQueueSize:         10,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	job := &Job{
		Result:  make(chan *domain.JobResult, 1),
		Tokens:  100,
		Inputs:  &llmManagement.Inputs{Def: &jsonSchema.Definition{}},
		Retries: 0,
	}

	// Process the job - should trigger transient error handling
	go orchestrator.orchestrationJob(job, 0)

	// Wait for retry handler to process (it has a sleep)
	time.Sleep(200 * time.Millisecond)

	// Verify job was re-enqueued (transient errors are retried)
	if queue.getEnqueueCount() == 0 {
		t.Error("Expected job to be re-enqueued for transient error")
	}

	// Verify retries were incremented
	if job.Retries != 1 {
		t.Errorf("Expected retries to be 1, got %d", job.Retries)
	}
}

func TestOrchestrationJob_PermanentError(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	clientAdapter.processFunc = func(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
		return nil, errors.New("HTTP request failed with status 400")
	}

	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:          1,
		MaxTokensPerMinute:   60000,
		MaxRequestsPerMinute: 6000,
		MaxQueueSize:         10,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{Def: &jsonSchema.Definition{}},
	}

	// Process the job
	orchestrator.orchestrationJob(job, 0)

	// Verify nil result was sent (permanent errors drop the job)
	select {
	case result := <-job.Result:
		if result != nil {
			t.Error("Expected nil result for permanent error")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for result")
	}

	// Verify job was not re-enqueued
	if queue.getEnqueueCount() != 0 {
		t.Errorf("Expected 0 enqueues for permanent error, got %d", queue.getEnqueueCount())
	}
}

func TestStartProcessing_WithoutBatchManager(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:           2,
		MaxTokensPerMinute:    60000,
		MaxRequestsPerMinute:  6000,
		MaxQueueSize:          10,
		EnableBatchProcessing: false,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	// Start processing
	orchestrator.StartProcessing()

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	// Stop orchestrator
	orchestrator.Stop()

	// Verify queue manager was started and stopped
	// (This is implicit in the test not hanging)
}

func TestStartProcessing_WithBatchManager(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()
	batchManager := newMockBatchReqManager()

	config := OrchestratorConfig{
		Concurrency:           2,
		MaxTokensPerMinute:    60000,
		MaxRequestsPerMinute:  6000,
		MaxQueueSize:          10,
		EnableBatchProcessing: true,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)
	orchestrator.SetBatchManager(batchManager)

	// Start processing
	orchestrator.StartProcessing()

	// Give batch manager time to start
	time.Sleep(50 * time.Millisecond)

	// Verify batch manager was started
	if !batchManager.isStarted() {
		t.Error("Expected batch manager to be started")
	}

	// Stop orchestrator
	orchestrator.Stop()

	// Verify batch manager was stopped
	if !batchManager.isStopped() {
		t.Error("Expected batch manager to be stopped")
	}
}

func TestStop(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:          2,
		MaxTokensPerMinute:   60000,
		MaxRequestsPerMinute: 6000,
		MaxQueueSize:         10,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	orchestrator.StartProcessing()
	time.Sleep(50 * time.Millisecond)

	// Stop should complete without hanging
	done := make(chan bool)
	go func() {
		orchestrator.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Stop() did not complete in time")
	}
}

func TestGetJobQueueManager(t *testing.T) {
	queue := newMockOrchestratorJobQueue()
	orchestrator := NewOrchestrator(
		OrchestratorConfig{Concurrency: 1},
		newMockClientAdapter(),
		queue,
		newMockBackoffManager(),
		NewRetryHandler(3, false),
		backoff.NewErrorClassifier(),
	)

	if orchestrator.GetJobQueueManager() != queue {
		t.Error("GetJobQueueManager did not return the correct queue")
	}
}

func TestSubmitJobWithRouting_RealTime(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:            1,
		MaxTokensPerMinute:     60000,
		MaxRequestsPerMinute:   6000,
		MaxQueueSize:           10,
		EnableBatchProcessing:  true,
		BatchPriorityThreshold: 0,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	batchManager := newMockBatchReqManager()
	orchestrator.SetBatchManager(batchManager)

	// Create a high-priority job (priority >= threshold)
	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{
			Priority: 10, // Higher than threshold
			Def:      &jsonSchema.Definition{},
		},
	}

	err := orchestrator.SubmitJobWithRouting(job)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify job was routed to real-time queue
	if queue.getEnqueueCount() != 1 {
		t.Errorf("Expected 1 job in real-time queue, got %d", queue.getEnqueueCount())
	}

	// Verify job was NOT sent to batch manager
	if batchManager.getAddJobCalls() != 0 {
		t.Errorf("Expected 0 jobs in batch manager, got %d", batchManager.getAddJobCalls())
	}
}

func TestSubmitJobWithRouting_Batch(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:            1,
		MaxTokensPerMinute:     60000,
		MaxRequestsPerMinute:   6000,
		MaxQueueSize:           10,
		EnableBatchProcessing:  true,
		BatchPriorityThreshold: 5,
		Verbose:                true,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	batchManager := newMockBatchReqManager()
	orchestrator.SetBatchManager(batchManager)

	// Create a low-priority job (priority < threshold)
	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{
			Priority: 1, // Lower than threshold
			Def:      &jsonSchema.Definition{},
		},
	}

	err := orchestrator.SubmitJobWithRouting(job)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify job was routed to batch manager
	if batchManager.getAddJobCalls() != 1 {
		t.Errorf("Expected 1 job in batch manager, got %d", batchManager.getAddJobCalls())
	}

	// Verify job was NOT sent to real-time queue
	if queue.getEnqueueCount() != 0 {
		t.Errorf("Expected 0 jobs in real-time queue, got %d", queue.getEnqueueCount())
	}
}

func TestSubmitJobWithRouting_BatchDisabled(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:            1,
		MaxTokensPerMinute:     60000,
		MaxRequestsPerMinute:   6000,
		MaxQueueSize:           10,
		EnableBatchProcessing:  false, // Disabled
		BatchPriorityThreshold: 5,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	// Create a low-priority job
	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{
			Priority: 1,
			Def:      &jsonSchema.Definition{},
		},
	}

	err := orchestrator.SubmitJobWithRouting(job)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify job was routed to real-time queue (batch processing disabled)
	if queue.getEnqueueCount() != 1 {
		t.Errorf("Expected 1 job in real-time queue, got %d", queue.getEnqueueCount())
	}
}

func TestSubmitJobWithRouting_BatchError(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:            1,
		MaxTokensPerMinute:     60000,
		MaxRequestsPerMinute:   6000,
		MaxQueueSize:           10,
		EnableBatchProcessing:  true,
		BatchPriorityThreshold: 5,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	batchManager := newMockBatchReqManager()
	batchManager.addJobError = errors.New("batch manager full")
	orchestrator.SetBatchManager(batchManager)

	// Create a low-priority job
	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{
			Priority: 1,
			Def:      &jsonSchema.Definition{},
		},
	}

	err := orchestrator.SubmitJobWithRouting(job)
	if err == nil {
		t.Error("Expected error from batch manager")
	}
}

func TestGetBatchManager(t *testing.T) {
	orchestrator := NewOrchestrator(
		OrchestratorConfig{Concurrency: 1},
		newMockClientAdapter(),
		newMockOrchestratorJobQueue(),
		newMockBackoffManager(),
		NewRetryHandler(3, false),
		backoff.NewErrorClassifier(),
	)

	if orchestrator.GetBatchManager() != nil {
		t.Error("Expected nil batch manager initially")
	}

	batchManager := newMockBatchReqManager()
	orchestrator.SetBatchManager(batchManager)

	if orchestrator.GetBatchManager() != batchManager {
		t.Error("GetBatchManager did not return the correct batch manager")
	}
}

func TestIsBatchProcessingEnabled(t *testing.T) {
	tests := []struct {
		name            string
		enableBatch     bool
		setBatchManager bool
		expectedEnabled bool
	}{
		{
			name:            "Batch enabled with manager",
			enableBatch:     true,
			setBatchManager: true,
			expectedEnabled: true,
		},
		{
			name:            "Batch enabled without manager",
			enableBatch:     true,
			setBatchManager: false,
			expectedEnabled: false,
		},
		{
			name:            "Batch disabled with manager",
			enableBatch:     false,
			setBatchManager: true,
			expectedEnabled: false,
		},
		{
			name:            "Batch disabled without manager",
			enableBatch:     false,
			setBatchManager: false,
			expectedEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := OrchestratorConfig{
				Concurrency:           1,
				EnableBatchProcessing: tt.enableBatch,
			}

			orchestrator := NewOrchestrator(
				config,
				newMockClientAdapter(),
				newMockOrchestratorJobQueue(),
				newMockBackoffManager(),
				NewRetryHandler(3, false),
				backoff.NewErrorClassifier(),
			)

			if tt.setBatchManager {
				orchestrator.SetBatchManager(newMockBatchReqManager())
			}

			if orchestrator.IsBatchProcessingEnabled() != tt.expectedEnabled {
				t.Errorf("Expected IsBatchProcessingEnabled to be %v, got %v",
					tt.expectedEnabled, orchestrator.IsBatchProcessingEnabled())
			}
		})
	}
}

func TestOrchestrationJob_TokenBurstExceeded(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	// Set very low token limit
	config := OrchestratorConfig{
		Concurrency:          1,
		MaxTokensPerMinute:   60, // Very low
		MaxRequestsPerMinute: 6000,
		MaxQueueSize:         10,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	// Create a job that requires more tokens than the burst limit
	job := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100, // Exceeds burst limit of 60
		Inputs: &llmManagement.Inputs{
			Def: &jsonSchema.Definition{},
		},
	}

	// Process the job
	orchestrator.orchestrationJob(job, 0)

	// Verify nil result was sent (job dropped)
	select {
	case result := <-job.Result:
		if result != nil {
			t.Error("Expected nil result for job exceeding token burst")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for result")
	}
}

func TestOrchestratorConfig_VerboseLogging(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, true) // Verbose enabled
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:            1,
		MaxTokensPerMinute:     60000,
		MaxRequestsPerMinute:   6000,
		MaxQueueSize:           10,
		Verbose:                true,
		EnableBatchProcessing:  true,
		BatchPriorityThreshold: 5,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	batchManager := newMockBatchReqManager()
	orchestrator.SetBatchManager(batchManager)

	// Create jobs with different priorities to test verbose logging
	highPriorityJob := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{
			Priority: 10,
			Def:      &jsonSchema.Definition{Priority: 10},
		},
	}

	lowPriorityJob := &Job{
		Result: make(chan *domain.JobResult, 1),
		Tokens: 100,
		Inputs: &llmManagement.Inputs{
			Priority: 1,
			Def:      &jsonSchema.Definition{Priority: 1},
		},
	}

	// Submit jobs - verbose logging should be active
	_ = orchestrator.SubmitJobWithRouting(highPriorityJob)
	_ = orchestrator.SubmitJobWithRouting(lowPriorityJob)

	// Just verify the routing worked correctly (logs are tested implicitly)
	if queue.getEnqueueCount() != 1 {
		t.Errorf("Expected 1 job in real-time queue, got %d", queue.getEnqueueCount())
	}

	if batchManager.getAddJobCalls() != 1 {
		t.Errorf("Expected 1 job in batch manager, got %d", batchManager.getAddJobCalls())
	}
}

func TestConcurrentJobProcessing(t *testing.T) {
	clientAdapter := newMockClientAdapter()
	processCount := 0
	var mu sync.Mutex

	clientAdapter.processFunc = func(inputs *llmManagement.Inputs) (*domain.JobResult, error) {
		mu.Lock()
		processCount++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond) // Simulate work
		return &domain.JobResult{
			ChatRes: &openai.ChatCompletionResponse{ID: "test"},
		}, nil
	}

	queue := newMockOrchestratorJobQueue()
	backoffManager := newMockBackoffManager()
	retryHandler := NewRetryHandler(3, false)
	classifier := backoff.NewErrorClassifier()

	config := OrchestratorConfig{
		Concurrency:          3, // Multiple workers
		MaxTokensPerMinute:   60000,
		MaxRequestsPerMinute: 6000,
		MaxQueueSize:         10,
	}

	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		queue,
		backoffManager,
		retryHandler,
		classifier,
	)

	orchestrator.StartProcessing()

	// Submit multiple jobs
	numJobs := 5
	results := make(chan *domain.JobResult, numJobs)
	for i := 0; i < numJobs; i++ {
		job := &Job{
			Result: results,
			Tokens: 100,
			Inputs: &llmManagement.Inputs{
				Def: &jsonSchema.Definition{},
			},
		}
		queue.Enqueue(job)
	}

	// Collect results
	receivedCount := 0
	timeout := time.After(2 * time.Second)
	for receivedCount < numJobs {
		select {
		case result := <-results:
			if result != nil {
				receivedCount++
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for results. Received %d/%d", receivedCount, numJobs)
		}
	}

	orchestrator.Stop()

	mu.Lock()
	finalProcessCount := processCount
	mu.Unlock()

	if finalProcessCount != numJobs {
		t.Errorf("Expected %d jobs processed, got %d", numJobs, finalProcessCount)
	}
}

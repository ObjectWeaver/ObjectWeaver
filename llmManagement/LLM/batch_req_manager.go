package LLM

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/client"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/domain"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/modelConverter"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/requestManagement"

	"github.com/sashabaranov/go-openai"
)

const (
	// Default maximum number of requests per batch
	defaultMaxRequestsPerBatch = 100
	// Default maximum memory usage in bytes (190 MB, leaving buffer before 200 MB limit)
	defaultMaxMemoryBytes = 190 * 1024 * 1024
	// Default flush interval - how often to check and potentially send batch
	defaultFlushInterval = 5 * time.Minute
)

// IBatchReqManager defines the interface for managing batch requests
type IBatchReqManager interface {
	// AddJob adds a job to the batch queue
	AddJob(job *Job) error
	// FlushBatch forces the current batch to be sent immediately
	FlushBatch(ctx context.Context) error
	// Start begins the background flush routine
	Start(ctx context.Context)
	// Stop gracefully stops the manager and flushes pending requests
	Stop(ctx context.Context) error
	// GetStats returns statistics about the batch manager
	GetStats() *BatchManagerStats
}

// BatchManagerStats provides statistics about the batch manager
type BatchManagerStats struct {
	PendingJobs      int
	TotalJobsQueued  int64
	TotalBatchesSent int64
	CurrentMemoryMB  float64
	LastFlushTime    time.Time
}

// BatchJobEntry wraps a Job with metadata for batch processing
type BatchJobEntry struct {
	Job          *Job
	CustomID     string
	AddedAt      time.Time
	EstimatedMem int64 // Estimated memory usage in bytes
}

// DefaultBatchReqManager implements batch request management
type DefaultBatchReqManager struct {
	// Configuration
	maxRequestsPerBatch int
	maxMemoryBytes      int64
	flushInterval       time.Duration
	batchClient         *client.BatchClient
	webhookNotifier     IBatchWebhookNotifier

	// State
	mu                sync.RWMutex
	pendingJobs       []*BatchJobEntry
	currentMemoryUsed int64
	totalJobsQueued   int64
	totalBatchesSent  int64
	lastFlushTime     time.Time

	// Control
	stopChan  chan struct{}
	flushChan chan struct{}
	wg        sync.WaitGroup
	isRunning bool

	//Request Builder
	requestBuilder requestManagement.RequestBuilder
}

// BatchManagerConfig holds configuration for the batch manager
type BatchManagerConfig struct {
	MaxRequestsPerBatch int
	MaxMemoryBytes      int64
	FlushInterval       time.Duration
	BatchClient         *client.BatchClient
	WebhookNotifier     IBatchWebhookNotifier // Optional: for batch completion notifications
}

// NewBatchReqManager creates a new batch request manager with default settings
func NewBatchReqManager(batchClient *client.BatchClient) (IBatchReqManager, error) {
	config := &BatchManagerConfig{
		MaxRequestsPerBatch: getIntFromEnv("LLM_BATCH_MAX_REQUESTS", defaultMaxRequestsPerBatch),
		MaxMemoryBytes:      int64(getIntFromEnv("LLM_BATCH_MAX_MEMORY_MB", defaultMaxMemoryBytes/(1024*1024))) * 1024 * 1024,
		FlushInterval:       time.Duration(getIntFromEnv("LLM_BATCH_FLUSH_INTERVAL_SEC", int(defaultFlushInterval.Seconds()))) * time.Second,
		BatchClient:         batchClient,
		WebhookNotifier:     NewBatchWebhookNotifierFromEnv(), // Initialize from env vars
	}

	return NewBatchReqManagerWithConfig(config)
}

// NewBatchReqManagerWithConfig creates a new batch request manager with custom configuration
func NewBatchReqManagerWithConfig(config *BatchManagerConfig) (IBatchReqManager, error) {
	if config.BatchClient == nil {
		return nil, fmt.Errorf("batch client is required")
	}

	if config.MaxRequestsPerBatch <= 0 {
		config.MaxRequestsPerBatch = defaultMaxRequestsPerBatch
	}

	if config.MaxMemoryBytes <= 0 {
		config.MaxMemoryBytes = defaultMaxMemoryBytes
	}

	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultFlushInterval
	}

	// If no webhook notifier provided, create a disabled one
	webhookNotifier := config.WebhookNotifier
	if webhookNotifier == nil {
		webhookNotifier = &DefaultBatchWebhookNotifier{enabled: false}
	}

	modelConv := modelConverter.NewModelConverter()

	return &DefaultBatchReqManager{
		maxRequestsPerBatch: config.MaxRequestsPerBatch,
		maxMemoryBytes:      config.MaxMemoryBytes,
		flushInterval:       config.FlushInterval,
		batchClient:         config.BatchClient,
		webhookNotifier:     webhookNotifier,
		pendingJobs:         make([]*BatchJobEntry, 0, config.MaxRequestsPerBatch),
		stopChan:            make(chan struct{}),
		flushChan:           make(chan struct{}, 1),
		lastFlushTime:       time.Now(),
		requestBuilder:      requestManagement.NewDefaultOpenAIReqBuilder(modelConv),
	}, nil
}

// AddJob adds a job to the batch queue
func (m *DefaultBatchReqManager) AddJob(job *Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	if job.Inputs == nil {
		return fmt.Errorf("job inputs cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create batch job entry
	entry := &BatchJobEntry{
		Job:          job,
		CustomID:     fmt.Sprintf("job-%d-%d", time.Now().UnixNano(), m.totalJobsQueued),
		AddedAt:      time.Now(),
		EstimatedMem: estimateJobMemory(job),
	}

	// Check if adding this job would exceed limits
	newMemoryUsed := m.currentMemoryUsed + entry.EstimatedMem
	newJobCount := len(m.pendingJobs) + 1

	// If adding this job would exceed limits, trigger flush first
	if newJobCount > m.maxRequestsPerBatch || newMemoryUsed > m.maxMemoryBytes {
		// Unlock temporarily to flush (flush will reacquire lock)
		m.mu.Unlock()
		if err := m.FlushBatch(context.Background()); err != nil {
			m.mu.Lock() // Reacquire lock for defer
			return fmt.Errorf("failed to flush batch before adding job: %w", err)
		}
		m.mu.Lock() // Reacquire lock
	}

	// Add job to pending queue
	m.pendingJobs = append(m.pendingJobs, entry)
	m.currentMemoryUsed += entry.EstimatedMem
	m.totalJobsQueued++

	return nil
}

// FlushBatch sends all pending jobs as a batch request
func (m *DefaultBatchReqManager) FlushBatch(ctx context.Context) error {
	m.mu.Lock()

	// Check if there are jobs to flush
	if len(m.pendingJobs) == 0 {
		m.mu.Unlock()
		return nil
	}

	// Take snapshot of pending jobs
	jobsToFlush := make([]*BatchJobEntry, len(m.pendingJobs))
	copy(jobsToFlush, m.pendingJobs)

	// Clear pending state
	m.pendingJobs = make([]*BatchJobEntry, 0, m.maxRequestsPerBatch)
	m.currentMemoryUsed = 0
	m.lastFlushTime = time.Now()

	m.mu.Unlock()

	// Convert jobs to batch requests
	batchRequests, err := m.convertJobsToBatchRequests(jobsToFlush)
	if err != nil {
		// Re-add jobs to queue on error
		m.mu.Lock()
		m.pendingJobs = append(m.pendingJobs, jobsToFlush...)
		m.mu.Unlock()
		return fmt.Errorf("failed to convert jobs to batch requests: %w", err)
	}

	// Create temporary file for batch input
	inputFile, err := m.createBatchInputFile(batchRequests)
	if err != nil {
		return fmt.Errorf("failed to create batch input file: %w", err)
	}
	defer os.Remove(inputFile)

	// Upload file
	file, err := m.batchClient.UploadFile(ctx, inputFile)
	if err != nil {
		return fmt.Errorf("failed to upload batch file: %w", err)
	}

	// Create batch
	batchReq := client.CreateBatchRequest{
		InputFileID:      file.ID,
		Endpoint:         client.EndpointChatCompletions,
		CompletionWindow: "24h",
		Metadata: map[string]interface{}{
			"job_count":  len(jobsToFlush),
			"created_at": time.Now().Unix(),
		},
	}

	batch, err := m.batchClient.CreateBatch(ctx, batchReq)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	m.mu.Lock()
	m.totalBatchesSent++
	m.mu.Unlock()

	// Start goroutine to monitor batch completion and send results back to jobs
	go m.monitorBatchCompletion(ctx, batch.ID, jobsToFlush)

	return nil
}

// Start begins the background flush routine
func (m *DefaultBatchReqManager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.mu.Unlock()

	m.wg.Add(1)
	go m.flushRoutine(ctx)
}

// Stop gracefully stops the manager and flushes pending requests
func (m *DefaultBatchReqManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	// Signal stop
	close(m.stopChan)

	// Wait for background routines
	m.wg.Wait()

	// Final flush
	return m.FlushBatch(ctx)
}

// GetStats returns statistics about the batch manager
func (m *DefaultBatchReqManager) GetStats() *BatchManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &BatchManagerStats{
		PendingJobs:      len(m.pendingJobs),
		TotalJobsQueued:  m.totalJobsQueued,
		TotalBatchesSent: m.totalBatchesSent,
		CurrentMemoryMB:  float64(m.currentMemoryUsed) / (1024 * 1024),
		LastFlushTime:    m.lastFlushTime,
	}
}

// flushRoutine periodically checks and flushes batches
func (m *DefaultBatchReqManager) flushRoutine(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.mu.RLock()
			shouldFlush := len(m.pendingJobs) > 0
			m.mu.RUnlock()

			if shouldFlush {
				if err := m.FlushBatch(ctx); err != nil {
					// Log error but continue
					fmt.Fprintf(os.Stderr, "Error flushing batch: %v\n", err)
				}
			}
		case <-m.flushChan:
			if err := m.FlushBatch(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Error flushing batch: %v\n", err)
			}
		}
	}
}

// convertJobsToBatchRequests converts Job objects to batch API requests
func (m *DefaultBatchReqManager) convertJobsToBatchRequests(jobs []*BatchJobEntry) ([]client.BatchRequest, error) {
	requests := make([]client.BatchRequest, 0, len(jobs))

	for _, entry := range jobs {
		// Build request body from job inputs

		openAiReqFormat, err := m.requestBuilder.BuildRequest(entry.Job.Inputs)
		if err != nil {
			return nil, fmt.Errorf("failed to build request for job %s: %w", entry.CustomID, err)
		}

		bytes, err := json.Marshal(openAiReqFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body for job %s: %w", entry.CustomID, err)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(bytes, &body); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request body for job %s: %w", entry.CustomID, err)
		}

		req := client.BatchRequest{
			CustomID: entry.CustomID,
			Method:   "POST",
			URL:      "/v1/chat/completions",
			Body:     body,
		}

		requests = append(requests, req)
	}

	return requests, nil
}

// createBatchInputFile creates a JSONL file for batch processing
func (m *DefaultBatchReqManager) createBatchInputFile(requests []client.BatchRequest) (string, error) {
	tmpFile, err := os.CreateTemp("", "batch-input-*.jsonl")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	encoder := json.NewEncoder(tmpFile)
	for _, req := range requests {
		if err := encoder.Encode(req); err != nil {
			os.Remove(tmpFile.Name())
			return "", fmt.Errorf("failed to encode request: %w", err)
		}
	}

	return tmpFile.Name(), nil
}

// monitorBatchCompletion monitors a batch and sends results back to job channels
func (m *DefaultBatchReqManager) monitorBatchCompletion(ctx context.Context, batchID string, jobs []*BatchJobEntry) {
	// Wait for batch completion
	batch, err := m.batchClient.WaitForBatch(ctx, batchID)
	if err != nil {
		// Send error to all jobs
		for _, entry := range jobs {
			if entry.Job.Error != nil {
				entry.Job.Error <- fmt.Errorf("batch processing failed: %w", err)
			}
		}
		return
	}

	// Check batch status
	if batch.Status != client.StatusCompleted {
		for _, entry := range jobs {
			if entry.Job.Error != nil {
				entry.Job.Error <- fmt.Errorf("batch ended with status: %s", batch.Status)
			}
		}
		return
	}

	// Retrieve results
	if batch.OutputFileID == nil {
		for _, entry := range jobs {
			if entry.Job.Error != nil {
				entry.Job.Error <- fmt.Errorf("batch completed but no output file available")
			}
		}
		return
	}

	responses, err := m.batchClient.ReadBatchResponses(ctx, *batch.OutputFileID)
	if err != nil {
		for _, entry := range jobs {
			if entry.Job.Error != nil {
				entry.Job.Error <- fmt.Errorf("failed to read batch responses: %w", err)
			}
		}
		return
	}

	// Send webhook notification if enabled
	if m.webhookNotifier.IsEnabled() {
		if err := m.webhookNotifier.NotifyBatchComplete(ctx, batch, responses, jobs); err != nil {
			// Log webhook error but don't fail the batch processing
			fmt.Fprintf(os.Stderr, "Warning: webhook notification failed for batch %s: %v\n", batchID, err)
		}
	}

	// Map responses back to jobs
	responseMap := make(map[string]client.BatchResponse)
	for _, resp := range responses {
		responseMap[resp.CustomID] = resp
	}

	// Send results to job channels
	for _, entry := range jobs {
		resp, ok := responseMap[entry.CustomID]
		if !ok {
			if entry.Job.Error != nil {
				entry.Job.Error <- fmt.Errorf("no response found for job %s", entry.CustomID)
			}
			continue
		}

		if resp.Error != nil {
			if entry.Job.Error != nil {
				entry.Job.Error <- fmt.Errorf("batch request failed: %s", resp.Error.Message)
			}
			continue
		}

		// Convert response to openai.ChatCompletionResponse
		if resp.Response != nil && resp.Response.Body != nil {
			chatResp, err := convertBatchResponseToChatCompletion(resp.Response.Body)
			if err != nil {
				if entry.Job.Error != nil {
					entry.Job.Error <- fmt.Errorf("failed to convert response: %w", err)
				}
				continue
			}

			if entry.Job.Result != nil {
				entry.Job.Result <- domain.CreateJobResult(chatResp, nil)
			}
		}
	}
}

// estimateJobMemory estimates the memory usage of a job in bytes
func estimateJobMemory(job *Job) int64 {
	// Rough estimation based on content size
	var size int64

	// Count prompt and system prompt
	size += int64(len(job.Inputs.Prompt))
	size += int64(len(job.Inputs.SystemPrompt))

	// Account for model name and other metadata (rough estimate)
	size += 1024 // 1KB for overhead

	// Add buffer for JSON serialization overhead
	size = size * 2

	return size
}

// convertBatchResponseToChatCompletion converts a batch response body to ChatCompletionResponse
func convertBatchResponseToChatCompletion(body map[string]interface{}) (*openai.ChatCompletionResponse, error) {
	// Marshal and unmarshal through JSON for type conversion
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var chatResp openai.ChatCompletionResponse
	if err := json.Unmarshal(jsonData, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// getIntFromEnv retrieves an integer value from environment variable with default
func getIntFromEnv(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	if value <= 0 {
		return defaultValue
	}

	return value
}

package LLM

import (
	"log"
	"objectweaver/llmManagement/backoff"
	"objectweaver/llmManagement/client"
	"objectweaver/llmManagement/clientManager"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OrchestrationService defines the public interface for the rate limiting and job processing service.
// This allows us to hide the internal Orchestrator implementation from the consumer.
type OrchestrationService interface {
	Stop()
	GetJobQueueManager() IJobQueueManager
}

// getVerbose checks an environment variable to determine if verbose logging should be enabled.
func getVerbose() bool {
	verbose := os.Getenv("VERBOSE")
	return verbose == "true"
}

// getBoolFromEnv retrieves a boolean value from environment variable
func getBoolFromEnv(key string, defaultValue bool) bool {
	valueStr := strings.ToLower(os.Getenv(key))
	if valueStr == "" {
		return defaultValue
	}
	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}

// getInt32FromEnv retrieves an int32 value from environment variable
func getInt32FromEnv(key string, defaultValue int32) int32 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int32(value)
}

// NewOrchestrationService creates, configures, and starts the entire job processing pipeline.
// It wires together all the necessary components based on the provided configuration.
func NewOrchestrationService(
	wg *sync.WaitGroup,
	maxTokensPerMinute int,
	maxRequestsPerMinute int,
	clientAdapter clientManager.ClientAdapter,
	strategy backoff.BackoffStrategy,
) OrchestrationService {
	verbose := getVerbose()
	maxQueueSize := maxRequestsPerMinute // A reasonable default

	// Determine the number of concurrent workers.
	concurrency := maxRequestsPerMinute / 4
	if concurrency < 1 {
		concurrency = 1 // Ensure at least one worker.
	}

	// 1. Create the core configuration
	config := OrchestratorConfig{
		Concurrency:            concurrency,
		MaxTokensPerMinute:     maxTokensPerMinute,
		MaxRequestsPerMinute:   maxRequestsPerMinute,
		MaxQueueSize:           maxQueueSize,
		Verbose:                verbose,
		EnableBatchProcessing:  getBoolFromEnv("LLM_ENABLE_BATCH", false),
		BatchPriorityThreshold: getInt32FromEnv("LLM_BATCH_PRIORITY_THRESHOLD", 0),
	}

	// 2. Instantiate independent components

	//TODO - add environment variables so that the queue type can be chosen
	jobQueue := NewJobQueueByType(QueueTypePriority)
	jobQueueManager := NewJobQueueManager(concurrency, maxQueueSize, jobQueue)
	errorClassifier := backoff.NewErrorClassifier()
	retryHandler := NewRetryHandler(3, verbose) // Max 3 retries for transient errors

	// 3. Select and instantiate the backoff strategy
	var backoffManager BackoffManager
	maxBackoffDuration := 1 * time.Minute

	switch strategy {
	case backoff.BackoffGlobalExponential:
		backoffManager = backoff.NewGlobalExponentialBackoff(maxBackoffDuration, verbose)
	case backoff.BackoffPerWorkerExponential:
		backoffManager = backoff.NewPerWorkerExponentialBackoff(maxBackoffDuration, config.Concurrency, verbose)
	case backoff.BackoffNone:
		fallthrough
	default:
		backoffManager = &backoff.NoBackoff{Verbose: verbose}
	}

	// 4. Create the main Orchestrator, injecting all dependencies
	orchestrator := NewOrchestrator(
		config,
		clientAdapter,
		jobQueueManager,
		backoffManager,
		retryHandler,
		errorClassifier,
	)

	// 4.5. Initialize batch manager if batch processing is enabled
	if config.EnableBatchProcessing {
		if client.BatchAiClient != nil {
			batchManager, err := NewBatchReqManager(client.BatchAiClient)
			if err != nil {
				log.Printf("[WARNING] Failed to initialize batch manager: %v. Batch processing will be disabled.", err)
			} else {
				orchestrator.SetBatchManager(batchManager)
				if verbose {
					log.Printf("[BatchProcessing] Batch processing enabled with priority threshold: %d", config.BatchPriorityThreshold)
				}
			}
		} else {
			log.Println("[WARNING] Batch processing enabled but BatchAiClient is nil. Batch processing will be disabled.")
		}
	}

	// Add the orchestrator's wait group to the parent wait group
	// This ensures the main application can wait for the orchestrator to shut down cleanly.
	wg.Add(1)
	go func() {
		defer wg.Done()
		orchestrator.wg.Wait()
	}()

	// 5. Start the processing loops in the background
	orchestrator.StartProcessing()

	// 6. Return the orchestrator, satisfying the OrchestrationService interface
	return orchestrator
}

package LLM

import (
	"firechimp/llmManagement/backoff"
	"firechimp/llmManagement/clientManager"
	"os"
	"strconv"
	"sync"
	"time"
)

// OrchestrationService defines the public interface for the rate limiting and job processing service.
// This allows us to hide the internal Orchestrator implementation from the consumer.
type OrchestrationService interface {
	Enqueue(job *Job)
	Stop()
}

// getVerbose checks an environment variable to determine if verbose logging should be enabled.
func getVerbose() bool {
	verbose, _ := strconv.ParseBool(os.Getenv("VERBOSE_LOGS"))
	return verbose
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
		Concurrency:          concurrency,
		MaxTokensPerMinute:   maxTokensPerMinute,
		MaxRequestsPerMinute: maxRequestsPerMinute,
		MaxQueueSize:         maxQueueSize,
		Verbose:              verbose,
	}

	// 2. Instantiate independent components
	jobQueue := NewJobQueue(config.Concurrency, config.MaxQueueSize)
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
		jobQueue,
		backoffManager,
		retryHandler,
		errorClassifier,
	)

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

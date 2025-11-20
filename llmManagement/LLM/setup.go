package LLM

import (
	"log"
	"objectweaver/llmManagement/backoff"
	"objectweaver/llmManagement/clientManager"
	"objectweaver/logger"
	"os"
	"strconv"
	"strings"
	"sync"

	gogpt "github.com/sashabaranov/go-openai"
)

// --- Global WaitGroup ---
// This WaitGroup ensures graceful shutdown. The main application should call
// appWaitGroup.Wait() before exiting to allow all background processing to finish.
var appWaitGroup sync.WaitGroup

var (
	// OpenAI Services & Channels
	Orchestator   OrchestrationService
	WorkerChannel chan *Job
)

// --- API Clients ---
var GptClient *gogpt.Client

// The init function is called on application startup to configure and launch all services.
// Configuration is driven by environment variables for flexibility:
//   - LLM_PROVIDER: "local", "openai", or "gemini"
//   - LLM_API_URL: API endpoint URL (for local provider)
//   - LLM_API_KEY: API key
//   - LLM_USE_GZIP: Enable gzip compression
//   - LLM_CONCURRENCY: Number of worker goroutines (default: 100, max: 500)
//   - LLM_MAX_TOKENS_PER_MINUTE: Rate limit for tokens
//   - LLM_MAX_REQUESTS_PER_MINUTE: Rate limit for requests
//   - LLM_BACKOFF_STRATEGY: "none", "global", or "per-worker"
//   - SKIP_LLM_INIT: Set to "true" to skip initialization (useful for testing or manual initialization)
func init() {
	// Buffer the channel to handle concurrent load without blocking
	// Size: concurrency (100) * avg fields per request (50) = 5000
	WorkerChannel = make(chan *Job, 5000)

	// Skip initialization if explicitly disabled (for testing or manual initialization)
	if strings.ToLower(os.Getenv("SKIP_LLM_INIT")) == "true" {
		logger.Printf("[Init] Skipping LLM initialization (SKIP_LLM_INIT=true)")
		return
	}

	// Set default environment variables if not set
	if os.Getenv("LLM_PROVIDER") == "" {
		os.Setenv("LLM_PROVIDER", "local")
	}
	if os.Getenv("LLM_API_URL") == "" {
		os.Setenv("LLM_API_URL", "http://localhost:8080")
	}
	// LLM_SUBMITTER_TYPE: "default" (uses worker queue) or "direct" (bypasses queue)
	// Direct submitter eliminates 6 layers of indirection, reducing latency from 18s to ~150ms
	if os.Getenv("LLM_SUBMITTER_TYPE") == "" {
		os.Setenv("LLM_SUBMITTER_TYPE", "direct")
	}

	// Create adapters from environment variables using the factory pattern
	// Create multiple adapters to reduce lock contention on HTTP transport
	numAdapters := 32
	adapters := make([]clientManager.ClientAdapter, numAdapters)

	// Create first adapter to use for orchestrator (if needed)
	firstAdapter, err := clientManager.NewClientAdapterFromEnv()
	if err != nil {
		log.Fatalf("Failed to create LLM client adapter: %v", err)
	}
	adapters[0] = firstAdapter

	for i := 1; i < numAdapters; i++ {
		adapter, err := clientManager.NewClientAdapterFromEnv()
		if err != nil {
			log.Fatalf("Failed to create LLM client adapter %d: %v", i, err)
		}
		adapters[i] = adapter
	}

	// Initialize DirectSubmitter singleton eagerly (so it's ready for first request)
	submitterType := os.Getenv("LLM_SUBMITTER_TYPE")
	if submitterType == "direct" || submitterType == "" {
		maxConcurrent := getEnvInt("LLM_CONCURRENCY", 1000)
		verbose := strings.ToLower(os.Getenv("VERBOSE")) == "true"
		logger.Printf("[Init] Initializing DirectSubmitter with maxConcurrent=%d (bypassing worker queue)", maxConcurrent)
		_ = InitDirectSubmitter(adapters, maxConcurrent, verbose)
	}

	// Check if batch processing is enabled
	enableBatch := strings.ToLower(os.Getenv("LLM_ENABLE_BATCH")) == "true"

	// Only initialize orchestrator if using worker queue mode OR batch processing is enabled
	if submitterType == "default" || enableBatch {
		logger.Printf("[Init] Starting orchestrator (submitterType=%s, enableBatch=%v)", submitterType, enableBatch)

		// Rate limiting configuration from environment
		maxTokensPerMinute := getEnvInt("LLM_MAX_TOKENS_PER_MINUTE", 150000000)
		maxRequestsPerMinute := getEnvInt("LLM_MAX_REQUESTS_PER_MINUTE", 100)

		// Backoff strategy from environment
		backoffType := os.Getenv("LLM_BACKOFF_STRATEGY")
		var backoffStrat backoff.BackoffStrategy
		switch backoffType {
		case "global":
			backoffStrat = backoff.BackoffGlobalExponential
		case "per-worker":
			backoffStrat = backoff.BackoffPerWorkerExponential
		default:
			backoffStrat = backoff.BackoffNone
		}

		// Initialize the orchestration service with the configured adapter
		// Note: Orchestrator currently uses a single adapter, which is fine as it's legacy/batch path
		Orchestator = NewOrchestrationService(
			&appWaitGroup,
			maxTokensPerMinute,
			maxRequestsPerMinute,
			firstAdapter,
			backoffStrat,
		)

		createChannelBridge(WorkerChannel, Orchestator)
	} else {
		logger.Printf("[Init] Orchestrator DISABLED - using DirectSubmitter for all requests")
	}
}

// getEnvInt retrieves an integer value from environment variables with a default fallback.
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// createChannelBridge correctly links a legacy worker channel to a modern
// orchestration service. It passes the job without interfering with the result channel.
// It uses SubmitJobWithRouting to enable batch processing based on job priority.
func createChannelBridge(channel <-chan *Job, limiter OrchestrationService) {
	go func() {
		for job := range channel {
			// Try to use SubmitJobWithRouting if available (for batch processing support)
			// Fall back to direct Enqueue if the method is not available
			if orchestrator, ok := limiter.(*Orchestrator); ok {
				if err := orchestrator.SubmitJobWithRouting(job); err != nil {
					logger.Printf("[ChannelBridge ERROR] Failed to route job: %v, falling back to direct enqueue", err)
					limiter.GetJobQueueManager().Enqueue(job)
				}
			} else {
				// Legacy path: direct enqueue
				limiter.GetJobQueueManager().Enqueue(job)
			}
		}
	}()
}

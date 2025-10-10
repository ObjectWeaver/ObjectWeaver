package LLM

import (
	"objectweaver/llmManagement/backoff"
	"objectweaver/llmManagement/clientManager"
	"log"
	"os"
	"strconv"
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
//   - LLM_MAX_TOKENS_PER_MINUTE: Rate limit for tokens
//   - LLM_MAX_REQUESTS_PER_MINUTE: Rate limit for requests
//   - LLM_BACKOFF_STRATEGY: "none", "global", or "per-worker"
func init() {
	WorkerChannel = make(chan *Job)

	// Set default environment variables if not set
	if os.Getenv("LLM_PROVIDER") == "" {
		os.Setenv("LLM_PROVIDER", "local")
	}
	if os.Getenv("LLM_API_URL") == "" {
		os.Setenv("LLM_API_URL", "http://localhost:8080")
	}

	// Create adapter from environment variables using the factory pattern
	adapter, err := clientManager.NewClientAdapterFromEnv()
	if err != nil {
		log.Fatalf("Failed to create LLM client adapter: %v", err)
	}

	// Rate limiting configuration from environment
	maxTokensPerMinute := getEnvInt("LLM_MAX_TOKENS_PER_MINUTE", 150000000)
	maxRequestsPerMinute := getEnvInt("LLM_MAX_REQUESTS_PER_MINUTE", 500)

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
	Orchestator = NewOrchestrationService(
		&appWaitGroup,
		maxTokensPerMinute,
		maxRequestsPerMinute,
		adapter,
		backoffStrat,
	)

	createChannelBridge(WorkerChannel, Orchestator)
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
func createChannelBridge(channel <-chan *Job, limiter OrchestrationService) {
	go func() {
		for job := range channel {
			// Pass the job to the orchestrator. The orchestrator is now
			// responsible for the entire job lifecycle, including writing
			// the final result to job.Result.
			limiter.Enqueue(job)
		}
	}()
}

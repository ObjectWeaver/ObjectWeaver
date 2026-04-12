package LLM

import (
	"log"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/clientManager"
	"os"
	"strconv"
	"strings"
)

// Helper function to read integer from environment variable
func envInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil && intVal > 0 {
			return intVal
		}
	}
	return defaultValue
}

const (
	DefaultSubmitter = "default"
	VariedSubmitter  = "varied"
	DirectSubmitter  = "direct" // NEW: Bypasses worker queue, calls HTTP client directly
)

// directSubmitterInstance is a singleton for the direct submitter
var directSubmitterInstance *DirectJobSubmitter

// InitDirectSubmitter initializes the DirectSubmitter singleton eagerly at startup.
func InitDirectSubmitter(adapters []clientManager.ClientAdapter, maxConcurrent int, verbose bool) *DirectJobSubmitter {
	if directSubmitterInstance == nil {
		directSubmitterInstance = NewDirectJobSubmitter(adapters, maxConcurrent, verbose)
	}
	return directSubmitterInstance
}

func JobSubmitterFactory(submitterType string) JobSumitter {
	switch submitterType {
	case DefaultSubmitter:
		return NewDefaultJobSubmitter()
	case DirectSubmitter:
		return GetDirectSubmitter()
	default:
		return GetDirectSubmitter()
	}
}

// GetDirectSubmitter returns the singleton DirectJobSubmitter instance.
// This submitter bypasses the worker queue and calls the HTTP client directly,
func GetDirectSubmitter() *DirectJobSubmitter {
	if directSubmitterInstance == nil {
		// Create multiple adapters to reduce lock contention on HTTP transport
		// Each adapter has its own http.Client and Transport
		numAdapters := 32
		adapters := make([]clientManager.ClientAdapter, numAdapters)

		for i := 0; i < numAdapters; i++ {
			adapter, err := clientManager.NewClientAdapterFromEnv()
			if err != nil {
				log.Fatalf("Failed to create client adapter %d for DirectSubmitter: %v", i, err)
			}
			adapters[i] = adapter
		}

		// Get concurrency limit from environment (default: 100)
		maxConcurrent := envInt("LLM_CONCURRENCY", 1000)
		verbose := strings.ToLower(os.Getenv("VERBOSE")) == "true"

		directSubmitterInstance = NewDirectJobSubmitter(adapters, maxConcurrent, verbose)
	}
	return directSubmitterInstance
}

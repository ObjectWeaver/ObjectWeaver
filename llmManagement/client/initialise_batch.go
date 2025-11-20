package client

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var BatchAiClient *BatchClient

const (
	// Default timeout for HTTP requests (30 seconds)
	defaultTimeout = 30
	// Default poll interval for checking batch status (10 seconds)
	defaultPollInterval = 10
	// Default OpenAI base URL for batch operations
	openAIBatchBaseURL = "https://api.openai.com/v1"
)

type BatchClientSettings struct {
	APIKey       string
	BaseURL      string
	Timeout      int
	PollInterval int
	UseGzip      bool
}

func init() {
	settings, err := GetSettings()
	if err != nil {
		// Log error but don't panic - allow application to start
		// Application can check if BatchAiClient is nil before using
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize batch client: %v\n", err)
		return
	}

	// Create HTTP client with timeout and optimized transport for high concurrency
	transport := &http.Transport{
		MaxIdleConns:          1000,             // Increased from default 100
		MaxIdleConnsPerHost:   200,              // Increased from default 2
		MaxConnsPerHost:       0,                // 0 = unlimited
		IdleConnTimeout:       90 * time.Second, // Keep connections alive longer
		DisableKeepAlives:     false,            // Enable connection reuse
		DisableCompression:    false,            // Enable compression
		ForceAttemptHTTP2:     true,             // Use HTTP/2 when available
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient := &http.Client{
		Timeout:   time.Duration(settings.Timeout) * time.Second,
		Transport: transport,
	}

	// Apply gzip if enabled
	if settings.UseGzip {
		baseClient := NewStandardClient()
		httpClient = NewGenericGzipClient(baseClient)
		httpClient.Timeout = time.Duration(settings.Timeout) * time.Second

		// Ensure gzip client also has optimized transport
		if httpClient.Transport == nil {
			httpClient.Transport = transport
		}
	}

	// Create batch client with settings
	pollInterval := time.Duration(settings.PollInterval) * time.Second
	BatchAiClient = NewBatchClientWithHTTPClient(
		settings.APIKey,
		settings.BaseURL,
		pollInterval,
		httpClient,
	)
}

// GetSettings retrieves batch client settings from environment variables
// with validation and default values.
//
// Environment variables:
//   - LLM_BATCH_API_KEY: API key for batch operations (falls back to LLM_API_KEY)
//   - LLM_BATCH_BASE_URL: Base URL for batch API (auto-detected based on provider)
//   - LLM_BATCH_PROVIDER: Provider type ("openai", "local", etc.) - falls back to LLM_PROVIDER
//   - LLM_BATCH_API_URL: For local/custom endpoints (falls back to LLM_API_URL)
//   - LLM_BATCH_TIMEOUT: HTTP timeout in seconds (default: 30)
//   - LLM_BATCH_POLL_INTERVAL: Polling interval in seconds (default: 10)
//   - LLM_BATCH_USE_GZIP: Enable gzip compression (default: false)
func GetSettings() (*BatchClientSettings, error) {
	// API Key: Try batch-specific first, then fall back to general LLM key
	batchApiKey := os.Getenv("LLM_BATCH_API_KEY")
	if batchApiKey == "" {
		batchApiKey = os.Getenv("LLM_API_KEY")
	}

	// Get timeout and poll interval with defaults
	batchTimeout := getIntFromEnv("LLM_BATCH_TIMEOUT", defaultTimeout)
	batchPollInterval := getIntFromEnv("LLM_BATCH_POLL_INTERVAL", defaultPollInterval)

	// Get gzip setting
	useGzip := strings.ToLower(os.Getenv("LLM_BATCH_USE_GZIP")) == "true"
	if useGzip {
		// Fall back to general gzip setting if batch-specific not set
		if os.Getenv("LLM_BATCH_USE_GZIP") == "" {
			useGzip = strings.ToLower(os.Getenv("LLM_USE_GZIP")) == "true"
		}
	}

	// Determine base URL based on provider
	baseURL, err := determineBaseURL()
	if err != nil {
		return nil, err
	}

	return &BatchClientSettings{
		APIKey:       batchApiKey,
		BaseURL:      baseURL,
		Timeout:      batchTimeout,
		PollInterval: batchPollInterval,
		UseGzip:      useGzip,
	}, nil
}

// determineBaseURL determines the appropriate base URL for batch operations
// based on environment variables and provider configuration
func determineBaseURL() (string, error) {
	// Check for explicit batch base URL first
	if baseURL := os.Getenv("LLM_BATCH_BASE_URL"); baseURL != "" {
		return strings.TrimSuffix(baseURL, "/"), nil
	}

	// Check for batch-specific API URL (for local/custom providers)
	if apiURL := os.Getenv("LLM_BATCH_API_URL"); apiURL != "" {
		// Extract base URL from full endpoint URL
		return extractBaseURL(apiURL), nil
	}

	// Determine provider
	provider := os.Getenv("LLM_BATCH_PROVIDER")
	if provider == "" {
		provider = os.Getenv("LLM_PROVIDER")
	}

	// Provider-specific base URLs
	switch strings.ToLower(provider) {
	case "openai", "":
		// Default to OpenAI if no provider specified
		return openAIBatchBaseURL, nil

	case "local":
		// For local provider, check for API URL
		if apiURL := os.Getenv("LLM_API_URL"); apiURL != "" {
			return extractBaseURL(apiURL), nil
		}
		// Default local endpoint (Ollama compatible)
		return "http://localhost:11434/v1", nil

	case "gemini":
		// Gemini doesn't support batch API through the same interface
		return "", fmt.Errorf("batch API not supported for Gemini provider - use OpenAI or local provider")

	default:
		return "", fmt.Errorf("unknown batch provider: %s (supported: openai, local)", provider)
	}
}

// extractBaseURL extracts the base URL from a full API endpoint URL
// Example: "https://api.openai.com/v1/chat/completions" -> "https://api.openai.com/v1"
func extractBaseURL(fullURL string) string {
	fullURL = strings.TrimSuffix(fullURL, "/")

	// Common endpoint patterns to remove
	patterns := []string{
		"/chat/completions",
		"/completions",
		"/embeddings",
		"/batches",
		"/files",
	}

	for _, pattern := range patterns {
		if strings.HasSuffix(fullURL, pattern) {
			return strings.TrimSuffix(fullURL, pattern)
		}
	}

	// If no pattern matched, assume it's already a base URL
	return fullURL
}

// getIntFromEnv retrieves an integer value from environment variable
// Returns defaultValue if the env var is not set, empty, or not a valid integer
func getIntFromEnv(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		// If parsing fails, return default value
		return defaultValue
	}

	// Validate that the value is positive
	if value <= 0 {
		return defaultValue
	}

	return value
}

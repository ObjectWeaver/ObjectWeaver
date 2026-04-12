package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

// OpenAI-compatible request structure (simplified)
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI-compatible response structure
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

var (
	// Configurable delay in milliseconds
	minDelay int
	maxDelay int
	// Port to listen on
	port string
	// Maximum concurrent requests
	maxConcurrent int64
	sem           *semaphore.Weighted

	// Random number generator pool to avoid global lock contention
	randPool = sync.Pool{
		New: func() interface{} {
			return rand.New(rand.NewSource(time.Now().UnixNano()))
		},
	}
)

func init() {
	// Read configuration from environment variables
	port = getEnv("MOCK_LLM_PORT", "8080")
	minDelay = getEnvInt("MOCK_LLM_MIN_DELAY_MS", 200)                // Realistic LLM latency
	maxDelay = getEnvInt("MOCK_LLM_MAX_DELAY_MS", 400)                // Realistic LLM latency
	maxConcurrent = int64(getEnvInt("MOCK_LLM_MAX_CONCURRENT", 5000)) // High capacity
	sem = semaphore.NewWeighted(maxConcurrent)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

// Generate mock response that ObjectWeaver expects
func generateMockResponse(req *ChatCompletionRequest) *ChatCompletionResponse {
	// Simulate processing delay
	var delay time.Duration
	if maxDelay > minDelay {
		rng := randPool.Get().(*rand.Rand)
		delay = time.Duration(minDelay+rng.Intn(maxDelay-minDelay)) * time.Millisecond
		randPool.Put(rng)
	} else {
		delay = time.Duration(minDelay) * time.Millisecond
	}
	time.Sleep(delay)

	// Generate a realistic mock response based on the request content
	mockContent := "John Doe"

	// If the request mentions "email", return an email
	if len(req.Messages) > 0 {
		lastMessage := req.Messages[len(req.Messages)-1].Content
		if strings.Contains(strings.ToLower(lastMessage), "email") {
			mockContent = "john.doe@example.com"
		} else if strings.Contains(strings.ToLower(lastMessage), "age") {
			mockContent = "30"
		} else if strings.Contains(strings.ToLower(lastMessage), "street") {
			mockContent = "123 Main St"
		} else if strings.Contains(strings.ToLower(lastMessage), "city") {
			mockContent = "Springfield"
		} else if strings.Contains(strings.ToLower(lastMessage), "zip") {
			mockContent = "12345"
		} else if strings.Contains(strings.ToLower(lastMessage), "order") {
			mockContent = "ORD-123456"
		} else if strings.Contains(strings.ToLower(lastMessage), "amount") {
			mockContent = "99.99"
		} else if strings.Contains(strings.ToLower(lastMessage), "product") {
			mockContent = "PROD-789"
		} else if strings.Contains(strings.ToLower(lastMessage), "quantity") {
			mockContent = "2"
		} else if strings.Contains(strings.ToLower(lastMessage), "boolean") || strings.Contains(strings.ToLower(lastMessage), "newsletter") || strings.Contains(strings.ToLower(lastMessage), "notification") {
			mockContent = "true"
		}
	}

	// Return standard OpenAI-compatible format
	return &ChatCompletionResponse{
		ID:      fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: mockContent,
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}
}

// Handler for /v1/chat/completions (OpenAI-compatible endpoint)
func chatCompletionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Acquire semaphore - limit concurrent processing
	ctx := r.Context()
	if err := sem.Acquire(ctx, 1); err != nil {
		http.Error(w, "Server too busy", http.StatusServiceUnavailable)
		return
	}
	defer sem.Release(1)

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	count := requestCount.Load()
	log.Printf("[REQ #%d] model=%s messages=%d", count, req.Model, len(req.Messages))
	if len(req.Messages) > 0 {
		log.Printf("[REQ #%d] last_message: %.200s", count, req.Messages[len(req.Messages)-1].Content)
	}

	response := generateMockResponse(&req)
	log.Printf("[RES #%d] content=%s", count, truncate(response.Choices[0].Message.Content, 100))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// Helper to truncate strings for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Stats handler to show mock server configuration
func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":               port,
		"min_delay_ms":       minDelay,
		"max_delay_ms":       maxDelay,
		"max_concurrent":     maxConcurrent,
		"uptime_seconds":     time.Since(startTime).Seconds(),
		"requests_processed": requestCount.Load(),
		"average_delay_ms":   float64(minDelay+maxDelay) / 2,
	})
}

var (
	startTime    = time.Now()
	requestCount atomic.Int64
)

// Middleware to count requests
func requestCounterMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		next(w, r)
	}
}

func main() {
	http.HandleFunc("/v1/chat/completions", requestCounterMiddleware(chatCompletionHandler))
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/stats", statsHandler)

	addr := ":" + port
	log.Printf("🚀 Mock LLM Server starting on %s", addr)
	log.Printf("   Min Delay: %dms, Max Delay: %dms", minDelay, maxDelay)
	log.Printf("   Max Concurrent: %d", maxConcurrent)
	log.Printf("   OpenAI-compatible endpoint: /v1/chat/completions")
	log.Printf("   Health check: /health")
	log.Printf("   Stats: /stats")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

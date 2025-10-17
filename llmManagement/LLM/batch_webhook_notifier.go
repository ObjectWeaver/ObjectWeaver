package LLM

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"objectweaver/llmManagement/client"
)

// IBatchWebhookNotifier defines the interface for sending batch completion notifications
type IBatchWebhookNotifier interface {
	// NotifyBatchComplete sends a notification when a batch is completed
	NotifyBatchComplete(ctx context.Context, batch *client.Batch, responses []client.BatchResponse, jobs []*BatchJobEntry) error
	// IsEnabled returns true if webhook notifications are configured
	IsEnabled() bool
}

// BatchWebhookPayload represents the data sent to the webhook endpoint
type BatchWebhookPayload struct {
	BatchID      string                  `json:"batch_id"`
	Status       string                  `json:"status"`
	JobCount     int                     `json:"job_count"`
	CompletedAt  int64                   `json:"completed_at"`
	Responses    []WebhookResponse       `json:"responses"`
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
	RequestCounts *client.RequestCounts  `json:"request_counts,omitempty"`
}

// WebhookResponse represents a single response in the webhook payload
type WebhookResponse struct {
	CustomID  string                 `json:"custom_id"`
	Success   bool                   `json:"success"`
	Response  map[string]interface{} `json:"response,omitempty"`
	Error     *WebhookError          `json:"error,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// WebhookError represents an error in the webhook payload
type WebhookError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DefaultBatchWebhookNotifier implements webhook notifications for batch completions
type DefaultBatchWebhookNotifier struct {
	webhookURL    string
	apiKey        string
	httpClient    *http.Client
	enabled       bool
	retryAttempts int
	retryDelay    time.Duration
}

// BatchWebhookConfig holds configuration for the webhook notifier
type BatchWebhookConfig struct {
	WebhookURL    string
	APIKey        string
	HTTPClient    *http.Client
	RetryAttempts int
	RetryDelay    time.Duration
}

// NewBatchWebhookNotifierFromEnv creates a webhook notifier from environment variables
func NewBatchWebhookNotifierFromEnv() IBatchWebhookNotifier {
	webhookURL := os.Getenv("LLM_BATCH_WEBHOOK_URL")
	apiKey := os.Getenv("LLM_BATCH_WEBHOOK_API_KEY")

	// If no webhook URL is configured, return a disabled notifier
	if webhookURL == "" {
		return &DefaultBatchWebhookNotifier{enabled: false}
	}

	config := &BatchWebhookConfig{
		WebhookURL:    webhookURL,
		APIKey:        apiKey,
		RetryAttempts: getIntFromEnvWithDefault("LLM_BATCH_WEBHOOK_RETRY_ATTEMPTS", 3),
		RetryDelay:    time.Duration(getIntFromEnvWithDefault("LLM_BATCH_WEBHOOK_RETRY_DELAY_SEC", 5)) * time.Second,
	}

	return NewBatchWebhookNotifier(config)
}

// NewBatchWebhookNotifier creates a new webhook notifier with custom configuration
func NewBatchWebhookNotifier(config *BatchWebhookConfig) IBatchWebhookNotifier {
	if config.WebhookURL == "" {
		return &DefaultBatchWebhookNotifier{enabled: false}
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	retryAttempts := config.RetryAttempts
	if retryAttempts <= 0 {
		retryAttempts = 3
	}

	retryDelay := config.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}

	return &DefaultBatchWebhookNotifier{
		webhookURL:    config.WebhookURL,
		apiKey:        config.APIKey,
		httpClient:    httpClient,
		enabled:       true,
		retryAttempts: retryAttempts,
		retryDelay:    retryDelay,
	}
}

// IsEnabled returns true if webhook notifications are configured
func (n *DefaultBatchWebhookNotifier) IsEnabled() bool {
	return n.enabled
}

// NotifyBatchComplete sends a notification when a batch is completed
func (n *DefaultBatchWebhookNotifier) NotifyBatchComplete(
	ctx context.Context,
	batch *client.Batch,
	responses []client.BatchResponse,
	jobs []*BatchJobEntry,
) error {
	if !n.enabled {
		return nil
	}

	// Build webhook payload
	payload := n.buildPayload(batch, responses, jobs)

	// Send with retries
	var lastErr error
	for attempt := 0; attempt <= n.retryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(n.retryDelay):
			}
		}

		err := n.sendWebhook(ctx, payload)
		if err == nil {
			return nil
		}

		lastErr = err

		// Log retry attempt
		if attempt < n.retryAttempts {
			fmt.Fprintf(os.Stderr, "Webhook notification failed (attempt %d/%d): %v\n",
				attempt+1, n.retryAttempts+1, err)
		}
	}

	return fmt.Errorf("webhook notification failed after %d attempts: %w", n.retryAttempts+1, lastErr)
}

// buildPayload constructs the webhook payload from batch data
func (n *DefaultBatchWebhookNotifier) buildPayload(
	batch *client.Batch,
	responses []client.BatchResponse,
	jobs []*BatchJobEntry,
) *BatchWebhookPayload {
	webhookResponses := make([]WebhookResponse, 0, len(responses))

	for _, resp := range responses {
		webhookResp := WebhookResponse{
			CustomID:  resp.CustomID,
			Success:   resp.Error == nil,
			Timestamp: time.Now().Unix(),
		}

		if resp.Error != nil {
			webhookResp.Error = &WebhookError{
				Code:    resp.Error.Code,
				Message: resp.Error.Message,
			}
		} else if resp.Response != nil {
			webhookResp.Response = resp.Response.Body
		}

		webhookResponses = append(webhookResponses, webhookResp)
	}

	completedAt := time.Now().Unix()
	if batch.CompletedAt != nil {
		completedAt = *batch.CompletedAt
	}

	payload := &BatchWebhookPayload{
		BatchID:       batch.ID,
		Status:        string(batch.Status),
		JobCount:      len(jobs),
		CompletedAt:   completedAt,
		Responses:     webhookResponses,
		Metadata:      batch.Metadata,
		RequestCounts: &batch.RequestCounts,
	}

	return payload
}

// sendWebhook sends the webhook payload to the configured endpoint
func (n *DefaultBatchWebhookNotifier) sendWebhook(ctx context.Context, payload *BatchWebhookPayload) error {
	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", n.webhookURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ObjectWeaver-Batch-Notifier/1.0")

	// Add API key if configured
	if n.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+n.apiKey)
		// Also support X-API-Key header for compatibility
		req.Header.Set("X-API-Key", n.apiKey)
	}

	// Add custom headers for batch metadata
	req.Header.Set("X-Batch-ID", payload.BatchID)
	req.Header.Set("X-Batch-Status", payload.Status)
	req.Header.Set("X-Job-Count", fmt.Sprintf("%d", payload.JobCount))

	// Send request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// getIntFromEnvWithDefault retrieves an integer value from environment variable with default
func getIntFromEnvWithDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := fmt.Sscanf(valueStr, "%d", new(int))
	if err != nil || value <= 0 {
		return defaultValue
	}

	return value
}

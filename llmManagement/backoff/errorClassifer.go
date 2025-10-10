package backoff

import (
	"errors"
	"firechimp/logger"
	"fmt"
	"io"
	"strings"
)

// ErrorClassifier categorizes errors to determine the correct handling strategy.
type ErrorClassifier struct{}

func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{}
}

func (ec *ErrorClassifier) Classify(err error) ErrorType {
	if err == nil {
		return -1
	}

	logger.Output.Println(fmt.Sprintf("ErrorClassifier: Classifying error: %v", err))

	if errors.Is(err, io.EOF) {
		return ErrorTypeTransient
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) || strings.Contains(err.Error(), "429") {
		return ErrorTypeRateLimit
	}

	errStr := err.Error()

	// Handle OpenAI-specific error types
	if strings.Contains(errStr, "OpenAI API error") {
		if strings.Contains(errStr, "[type: invalid_request_error]") {
			return ErrorTypePermanent // Invalid requests shouldn't be retried
		}
		if strings.Contains(errStr, "[type: rate_limit_exceeded]") {
			return ErrorTypeRateLimit
		}
		if strings.Contains(errStr, "[type: server_error]") || strings.Contains(errStr, "[type: service_unavailable]") {
			return ErrorTypeTransient // Server errors can be retried
		}
	}

	transientErrors := []string{
		"client connection lost",
		"Service Unavailable",
		"Internal Server Error",
		"context deadline exceeded",
		"connection reset by peer",
		"broken pipe",
		"unexpected EOF",                                       // Added this error to the list
		"HTTP request failed with status 429",                  // Rate limit
		"HTTP request failed with status 502",                  // Bad Gateway
		"HTTP request failed with status 503",                  // Service Unavailable
		"HTTP request failed with status 504",                  // Gateway Timeout
		"failed to unmarshal json into ChatCompletionResponse", // JSON parsing errors (often due to error responses)
	}
	for _, te := range transientErrors {
		if strings.Contains(errStr, te) {
			return ErrorTypeTransient
		}
	}

	// Check for specific HTTP error codes that should be permanent
	permanentHttpErrors := []string{
		"HTTP request failed with status 401", // Unauthorized
		"HTTP request failed with status 403", // Forbidden
		"HTTP request failed with status 404", // Not Found
		"HTTP request failed with status 400", // Bad Request
	}
	for _, pe := range permanentHttpErrors {
		if strings.Contains(errStr, pe) {
			return ErrorTypePermanent
		}
	}

	return ErrorTypePermanent
}

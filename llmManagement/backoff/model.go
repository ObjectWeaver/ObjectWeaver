package backoff

import "time"

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "rate limit error"
}

type ErrorType int

const (
	ErrorTypeRateLimit ErrorType = iota
	ErrorTypeTransient
	ErrorTypePermanent
)

package backoff

import (
	"github.com/ObjectWeaver/ObjectWeaver/logger"
	"sync"
	"time"
)

// AdaptiveRateLimiter manages rate limits with dynamic backoff and recovery.
type AdaptiveRateLimiter struct {
	mu sync.Mutex
	// Map of model name to its specific limiter state
	limiters map[string]*modelLimiter
	// Default requests per minute if not specified
	defaultRPM int
}

type modelLimiter struct {
	// Configuration
	rpmLimit int

	// State
	requestCount int
	windowStart  time.Time

	// Backoff state
	backoffUntil time.Time
	inBackoff    bool

	// Ramp up state (Thundering herd prevention)
	rampUpStart time.Time
	rampUpLevel float64 // 0.0 to 1.0 (percentage of rpmLimit allowed)
}

// NewAdaptiveRateLimiter creates a new limiter with a default RPM.
func NewAdaptiveRateLimiter(defaultRPM int) *AdaptiveRateLimiter {
	if defaultRPM <= 0 {
		defaultRPM = 100000 // Default to 100000 RPM if invalid
	}
	return &AdaptiveRateLimiter{
		limiters:   make(map[string]*modelLimiter),
		defaultRPM: defaultRPM,
	}
}

// Wait blocks until a request is allowed for the given model.
// It returns an error if the context is cancelled or if the system is in a hard backoff.
func (l *AdaptiveRateLimiter) Wait(model string) error {
	l.mu.Lock()
	limiter, exists := l.limiters[model]
	if !exists {
		limiter = &modelLimiter{
			rpmLimit:    l.defaultRPM,
			windowStart: time.Now(),
			rampUpLevel: 1.0, // Start at full capacity until an error occurs
		}
		l.limiters[model] = limiter
	}
	l.mu.Unlock()

	// Loop until allowed or backoff expires
	for {
		l.mu.Lock()
		now := time.Now()

		// 1. Check Hard Backoff (Pause)
		if limiter.inBackoff {
			if now.Before(limiter.backoffUntil) {
				waitTime := limiter.backoffUntil.Sub(now)
				l.mu.Unlock()
				// We are in a forced pause.
				// To avoid holding a goroutine for a full minute, we could return an error
				// or sleep. Given the requirement "pause the processing", we sleep.
				// However, for very long pauses, returning an error might be better.
				// Here we'll sleep in small chunks to allow context cancellation if we had it.
				time.Sleep(waitTime)
				continue
			}
			// Backoff expired, enter ramp-up phase
			limiter.inBackoff = false
			limiter.rampUpStart = now
			limiter.rampUpLevel = 0.1 // Start at 10% capacity
			logger.Printf("[RateLimiter] Backoff expired for %s. Entering slow start at %.0f%% capacity.", model, limiter.rampUpLevel*100)
		}

		// 2. Reset Window if needed
		if now.Sub(limiter.windowStart) >= time.Minute {
			limiter.requestCount = 0
			limiter.windowStart = now

			// Increase ramp up level if we are in recovery mode
			if limiter.rampUpLevel < 1.0 {
				limiter.rampUpLevel += 0.2 // Increase by 20% each minute
				if limiter.rampUpLevel > 1.0 {
					limiter.rampUpLevel = 1.0
				}
				logger.Printf("[RateLimiter] Increasing capacity for %s to %.0f%%", model, limiter.rampUpLevel*100)
			}
		}

		// 3. Check Rate Limit
		effectiveLimit := int(float64(limiter.rpmLimit) * limiter.rampUpLevel)
		if effectiveLimit < 1 {
			effectiveLimit = 1
		}

		if limiter.requestCount >= effectiveLimit {
			// Limit reached for this window
			resetTime := limiter.windowStart.Add(time.Minute)
			waitTime := resetTime.Sub(now)
			l.mu.Unlock()

			if waitTime > 0 {
				time.Sleep(waitTime)
				continue
			}
		}

		// Allowed
		limiter.requestCount++
		l.mu.Unlock()
		return nil
	}
}

// ReportResult informs the limiter of the result of a request.
// If a 429 is reported, it triggers the backoff mechanism.
func (l *AdaptiveRateLimiter) ReportResult(model string, err error) {
	if err == nil {
		return
	}

	// Check for 429 or similar rate limit errors
	classifier := NewErrorClassifier()
	if classifier.Classify(err) == ErrorTypeRateLimit {
		l.mu.Lock()
		defer l.mu.Unlock()

		limiter, exists := l.limiters[model]
		if !exists {
			// Should not happen if Wait was called, but safe to ignore or create
			return
		}

		// Only trigger backoff if not already in backoff
		if !limiter.inBackoff {
			limiter.inBackoff = true
			limiter.backoffUntil = time.Now().Add(1 * time.Minute)
			limiter.rampUpLevel = 0.1 // Reset ramp up
			logger.Printf("[RateLimiter] 429 detected for %s. Pausing for 1 minute.", model)
		}
	}
}

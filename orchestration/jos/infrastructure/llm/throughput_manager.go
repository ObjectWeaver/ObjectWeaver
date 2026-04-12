package llm

import (
	"github.com/ObjectWeaver/ObjectWeaver/logger"
	"sync"
	"time"
)

const (
	// cooldownDuration is how long a model is paused after hitting rate limit
	cooldownDuration = 1 * time.Minute
	// maxWaitForModel is maximum time to wait when all models are rate-limited
	maxWaitForModel = 30 * time.Second
	// waitCheckInterval is how often to check if a model becomes available
	waitCheckInterval = 500 * time.Millisecond
)

type IThroughputManger interface {
	GetModelForRequest() string
	GetModelForRequestWithWait() string
	ReportRateLimitError(model string)
}

type Options struct {
	duration time.Time
}

type DefaultThroughputManager struct {
	models map[string]Options
	mu     sync.Mutex
}

var StandardThroughputManager *DefaultThroughputManager

func NewDefaultThroughputManager(models []string) *DefaultThroughputManager {
	modelMap := make(map[string]Options)
	for _, model := range models {
		modelMap[model] = Options{
			duration: time.Time{},
		}
	}
	return &DefaultThroughputManager{
		models: modelMap,
	}
}

// GetModelForRequest returns an available model or empty string if all are rate-limited
func (tm *DefaultThroughputManager) GetModelForRequest() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	return tm.getAvailableModelLocked()
}

// GetModelForRequestWithWait waits for a model to become available instead of returning empty
// This prevents jobs from failing when all models are temporarily rate-limited
func (tm *DefaultThroughputManager) GetModelForRequestWithWait() string {
	// first check without waiting
	tm.mu.Lock()
	model := tm.getAvailableModelLocked()
	tm.mu.Unlock()

	if model != "" {
		return model
	}

	// all models rate-limited - wait for one to become available
	logger.Printf("[ThroughputManager] All models rate-limited, waiting for cooldown...")

	startTime := time.Now()
	for time.Since(startTime) < maxWaitForModel {
		time.Sleep(waitCheckInterval)

		tm.mu.Lock()
		model = tm.getAvailableModelLocked()
		tm.mu.Unlock()

		if model != "" {
			logger.Printf("[ThroughputManager] Model '%s' now available after waiting %v", model, time.Since(startTime))
			return model
		}
	}

	// last resort - return the model with shortest remaining cooldown
	logger.Printf("[ThroughputManager] Wait timeout exceeded, selecting model with shortest cooldown")
	return tm.getModelWithShortestCooldown()
}

// getAvailableModelLocked returns an available model (caller must hold lock)
func (tm *DefaultThroughputManager) getAvailableModelLocked() string {
	for model, opts := range tm.models {
		if !opts.duration.IsZero() {
			if time.Since(opts.duration) > cooldownDuration {
				// reset after cooldown
				tm.models[model] = Options{
					duration: time.Time{},
				}
				return model
			}
		} else {
			return model
		}
	}
	return ""
}

// getModelWithShortestCooldown returns the model closest to being available
func (tm *DefaultThroughputManager) getModelWithShortestCooldown() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var bestModel string
	var shortestWait time.Duration = cooldownDuration

	for model, opts := range tm.models {
		if opts.duration.IsZero() {
			return model // this one is actually available
		}

		elapsed := time.Since(opts.duration)
		remaining := cooldownDuration - elapsed
		if remaining < shortestWait {
			shortestWait = remaining
			bestModel = model
		}
	}

	if bestModel != "" {
		// reset this model's cooldown so it can be used
		tm.models[bestModel] = Options{duration: time.Time{}}
		logger.Printf("[ThroughputManager] Force-selecting model '%s' (was %v from cooldown end)", bestModel, shortestWait)
	}

	return bestModel
}

func (tm *DefaultThroughputManager) ReportRateLimitError(model string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.models[model]; exists {
		if !tm.models[model].duration.IsZero() {
			return
		}
		tm.models[model] = Options{
			duration: time.Now(),
		}
	}
}

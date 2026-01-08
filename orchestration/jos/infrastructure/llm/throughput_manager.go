package llm

import (
	"sync"
	"time"
)

type IThroughputManger interface {
	GetModelForRequest() string
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

func (tm *DefaultThroughputManager) GetModelForRequest() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for model, opts := range tm.models {
		if !opts.duration.IsZero() {
			if time.Since(opts.duration) > 1*time.Minute {
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

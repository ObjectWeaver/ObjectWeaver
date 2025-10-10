package assembly

import (
	"firechimp/orchestration/jos/domain"
	"sync"
	"time"
)

// ProgressiveObjectAssembler maintains progressive state of entire object with token-level updates
type ProgressiveObjectAssembler struct {
	currentMap        map[string]interface{}
	progressiveFields map[string]*domain.ProgressiveValue
	mu                sync.RWMutex
	emitInterval      time.Duration
}

func NewProgressiveObjectAssembler(emitIntervalMs int) *ProgressiveObjectAssembler {
	return &ProgressiveObjectAssembler{
		currentMap:        make(map[string]interface{}),
		progressiveFields: make(map[string]*domain.ProgressiveValue),
		emitInterval:      time.Duration(emitIntervalMs) * time.Millisecond,
	}
}

func (a *ProgressiveObjectAssembler) Assemble(results []*domain.TaskResult) (*domain.GenerationResult, error) {
	// Fallback to default assembler
	defaultAssembler := NewDefaultAssembler()
	return defaultAssembler.Assemble(results)
}

func (a *ProgressiveObjectAssembler) AssembleProgressive(tokenStream <-chan *domain.TokenStreamChunk) (<-chan *domain.AccumulatedStreamChunk, error) {
	out := make(chan *domain.AccumulatedStreamChunk, 100)

	go func() {
		defer close(out)

		ticker := time.NewTicker(a.emitInterval)
		defer ticker.Stop()

		lastEmit := time.Now()

		for {
			select {
			case token, ok := <-tokenStream:
				if !ok {
					// Stream closed - emit final state
					a.emitCurrentState(out, nil, true)
					return
				}

				// Update progressive value
				a.updateProgressiveValue(token)

				// Emit if complete or interval passed
				if token.Complete || time.Since(lastEmit) >= a.emitInterval {
					a.emitCurrentState(out, token, false)
					lastEmit = time.Now()
				}

			case <-ticker.C:
				// Periodic emission
				a.emitCurrentState(out, nil, false)
				lastEmit = time.Now()
			}
		}
	}()

	return out, nil
}

func (a *ProgressiveObjectAssembler) updateProgressiveValue(token *domain.TokenStreamChunk) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := token.Key

	// Get or create progressive value
	pv, exists := a.progressiveFields[key]
	if !exists {
		pv = domain.NewProgressiveValue(key, token.Path)
		a.progressiveFields[key] = pv
	}

	// Update with new token
	pv.Append(token.Token)

	if token.Complete {
		pv.MarkComplete()
		// Update current map with complete value
		a.setNestedValue(a.currentMap, token.Path, pv.CurrentValue())
	} else {
		// Update current map with partial value
		a.setNestedValue(a.currentMap, token.Path, pv.CurrentValue())
	}
}

func (a *ProgressiveObjectAssembler) emitCurrentState(
	out chan<- *domain.AccumulatedStreamChunk,
	newToken *domain.TokenStreamChunk,
	isFinal bool,
) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create snapshot
	snapshot := a.deepCopy(a.currentMap)
	progressiveSnapshot := a.copyProgressiveFields()

	out <- &domain.AccumulatedStreamChunk{
		CurrentMap:        snapshot,
		ProgressiveFields: progressiveSnapshot,
		NewToken:          newToken,
		Progress:          a.calculateProgress(),
		IsFinal:           isFinal,
	}
}

func (a *ProgressiveObjectAssembler) calculateProgress() float64 {
	if len(a.progressiveFields) == 0 {
		return 0.0
	}

	completed := 0
	for _, pv := range a.progressiveFields {
		if pv.IsComplete() {
			completed++
		}
	}

	return float64(completed) / float64(len(a.progressiveFields))
}

func (a *ProgressiveObjectAssembler) copyProgressiveFields() map[string]*domain.ProgressiveValue {
	copied := make(map[string]*domain.ProgressiveValue)
	for k, v := range a.progressiveFields {
		copied[k] = v
	}
	return copied
}

func (a *ProgressiveObjectAssembler) setNestedValue(obj map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	if len(path) == 1 {
		obj[path[0]] = value
		return
	}

	current := obj
	for i := 0; i < len(path)-1; i++ {
		key := path[i]
		if _, exists := current[key]; !exists {
			current[key] = make(map[string]interface{})
		}
		if nextMap, ok := current[key].(map[string]interface{}); ok {
			current = nextMap
		}
	}

	current[path[len(path)-1]] = value
}

func (a *ProgressiveObjectAssembler) deepCopy(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[k] = a.deepCopy(val)
		case []interface{}:
			dst[k] = a.deepCopySlice(val)
		default:
			dst[k] = val
		}
	}
	return dst
}

func (a *ProgressiveObjectAssembler) deepCopySlice(src []interface{}) []interface{} {
	dst := make([]interface{}, len(src))
	for i, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[i] = a.deepCopy(val)
		case []interface{}:
			dst[i] = a.deepCopySlice(val)
		default:
			dst[i] = val
		}
	}
	return dst
}

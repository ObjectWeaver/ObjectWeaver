package assembly

import (
	"objectGeneration/orchestration/jos/domain"
	"sync"
)

// CompleteStreamingAssembler streams progressive map[string]any
type CompleteStreamingAssembler struct {
	accumulated map[string]interface{}
	mu          sync.RWMutex
}

func NewCompleteStreamingAssembler() *CompleteStreamingAssembler {
	return &CompleteStreamingAssembler{
		accumulated: make(map[string]interface{}),
	}
}

func (a *CompleteStreamingAssembler) Assemble(results []*domain.TaskResult) (*domain.GenerationResult, error) {
	// Fallback to default assembler
	defaultAssembler := NewDefaultAssembler()
	return defaultAssembler.Assemble(results)
}

func (a *CompleteStreamingAssembler) AssembleStreaming(results <-chan *domain.TaskResult) (<-chan *domain.StreamChunk, error) {
	out := make(chan *domain.StreamChunk, 100)

	go func() {
		defer close(out)

		totalFields := 0
		completedFields := 0

		for result := range results {
			if result.IsSuccess() {
				a.mu.Lock()

				// Add to accumulated map
				a.setNestedValue(a.accumulated, result.Path(), result.Value())
				completedFields++

				// Create snapshot of current state
				snapshot := a.deepCopy(a.accumulated)

				a.mu.Unlock()

				// Stream the complete accumulated map so far
				chunk := domain.NewStreamChunk(result.Key(), result.Value())
				chunk.NewKey = result.Key()
				chunk.NewValue = result.Value()
				chunk.WithAccumulatedData(snapshot)
				if totalFields > 0 {
					chunk.WithProgress(float64(completedFields) / float64(totalFields))
				}

				out <- chunk
			}
		}

		// Send final complete map
		a.mu.RLock()
		finalSnapshot := a.deepCopy(a.accumulated)
		a.mu.RUnlock()

		finalChunk := domain.NewStreamChunk("", nil)
		finalChunk.WithAccumulatedData(finalSnapshot)
		finalChunk.WithProgress(1.0)
		finalChunk.MarkFinal()
		out <- finalChunk
	}()

	return out, nil
}

func (a *CompleteStreamingAssembler) setNestedValue(obj map[string]interface{}, path []string, value interface{}) {
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

func (a *CompleteStreamingAssembler) deepCopy(src map[string]interface{}) map[string]interface{} {
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

func (a *CompleteStreamingAssembler) deepCopySlice(src []interface{}) []interface{} {
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
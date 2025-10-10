package assembly

import (
	"log"
	"objectGeneration/orchestration/jos/domain"
	"os"
)

// DefaultAssembler collects all results and returns complete map
type DefaultAssembler struct{}

func NewDefaultAssembler() *DefaultAssembler {
	return &DefaultAssembler{}
}

func (a *DefaultAssembler) Assemble(results []*domain.TaskResult) (*domain.GenerationResult, error) {
	data := make(map[string]interface{})
	metadata := domain.NewResultMetadata()

	// Log the number of results
	verboseLogs := os.Getenv("VERBOSE") == "true"
	if verboseLogs {
		log.Printf("[DefaultAssembler] Assembling %d task results", len(results))
	}

	for i, result := range results {
		if result.IsSuccess() {
			path := result.Path()
			key := result.Key()
			value := result.Value()

			if verboseLogs {
				log.Printf("[DefaultAssembler] Result %d: key=%s, path=%v, value type=%T, value=%v",
					i, key, path, value, value)
			}

			a.setNestedValue(data, result.Path(), result.Value())
			metadata.AddCost(result.Metadata().Cost)
			metadata.AddTokens(result.Metadata().TokensUsed)
			metadata.IncrementFieldCount()
		} else {
			if verboseLogs {
				log.Printf("[DefaultAssembler] Result %d: ERROR - key=%s, error=%v", i, result.Key(), result.Error())
			}
		}
	}

	if verboseLogs {
		log.Printf("[DefaultAssembler] Final data map: %+v", data)
		log.Printf("[DefaultAssembler] Final metadata: fields=%d, cost=%f, tokens=%d",
			metadata.FieldCount, metadata.Cost, metadata.TokensUsed)
	}

	return domain.NewGenerationResult(data, metadata), nil
}

func (a *DefaultAssembler) setNestedValue(obj map[string]interface{}, path []string, value interface{}) {
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
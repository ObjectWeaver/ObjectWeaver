package assembly

import (
	"fmt"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
)

// PathBasedAssembler assembles nested object structures from flat results using path information
// This eliminates the need for intermediate collection goroutines
type PathBasedAssembler struct{}

func NewPathBasedAssembler() *PathBasedAssembler {
	return &PathBasedAssembler{}
}

// Assemble takes flat results with path information and builds nested structure
func (a *PathBasedAssembler) Assemble(results []*domain.TaskResult) (*domain.GenerationResult, error) {
	root := make(map[string]interface{})
	detailedData := make(map[string]*domain.FieldResultWithMetadata)
	metadata := domain.NewResultMetadata()

	for _, result := range results {
		if result == nil || !result.IsSuccess() {
			continue
		}

		// Aggregate metadata
		if result.Metadata() != nil {
			metadata.AddTokens(result.Metadata().TokensUsed)
			metadata.AddPromptTokens(result.Metadata().PromptTokens)
			metadata.AddCompletionTokens(result.Metadata().CompletionTokens)
			metadata.AddCost(result.Metadata().Cost)
			metadata.IncrementFieldCount()
		}

		path := result.Path()
		if len(path) == 0 {
			// Root level field
			root[result.Key()] = result.Value()
			detailedData[result.Key()] = domain.NewFieldResultWithMetadata(
				result.Value(),
				result.Metadata(),
			)
		} else {
			// Nested field - traverse/create path
			if err := a.setNestedValue(root, path, result.Value()); err != nil {
				return nil, fmt.Errorf("failed to set nested value at path %v: %w", path, err)
			}

			// Store detailed data with full path as key
			pathKey := buildPathKey(path)
			detailedData[pathKey] = domain.NewFieldResultWithMetadata(
				result.Value(),
				result.Metadata(),
			)
		}
	}

	return domain.NewGenerationResultWithDetailedData(root, detailedData, metadata), nil
}

// setNestedValue traverses/creates the nested map structure and sets the value
func (a *PathBasedAssembler) setNestedValue(root map[string]interface{}, path []string, value interface{}) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	// Navigate to parent
	current := root
	for i := 0; i < len(path)-1; i++ {
		key := path[i]

		// Get or create nested map
		if existing, exists := current[key]; exists {
			if nestedMap, ok := existing.(map[string]interface{}); ok {
				current = nestedMap
			} else {
				return fmt.Errorf("path conflict: %s already exists as non-map", key)
			}
		} else {
			// Create new nested map
			nestedMap := make(map[string]interface{})
			current[key] = nestedMap
			current = nestedMap
		}
	}

	// Set the final value
	finalKey := path[len(path)-1]
	current[finalKey] = value
	return nil
}

func buildPathKey(path []string) string {
	key := ""
	for i, p := range path {
		if i > 0 {
			key += "."
		}
		key += p
	}
	return key
}

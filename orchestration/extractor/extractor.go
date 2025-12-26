package extractor

import (
	"fmt"
	"strings"
)


// Extractor handles the responsibility of extracting and joining values.
type Extractor interface {
	ExtractAndJoin(currentGen map[string]any, keys []string) string
}

func NewDefaultExtractor() Extractor {
	return &DefaultExtractor{}
}

type DefaultExtractor struct{}

func (de *DefaultExtractor) ExtractAndJoin(currentGen map[string]any, keys []string) string {
	var result []string
	for _, key := range keys {
		if value, exists := currentGen[key]; exists {
			if strValue, ok := value.(string); ok {
				result = append(result, strValue)
			} else {
				v := fmt.Sprintf("%v", value)
				if v == "" || v == "map[]" {
					continue
				}
				result = append(result, v)
			}
		}
	}
	return strings.Join(result, "\n")
}

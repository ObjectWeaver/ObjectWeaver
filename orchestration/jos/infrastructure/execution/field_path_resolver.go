package execution

import (
	"fmt"
	"strings"
)

// ResolveFieldPath resolves a field path like "car.color" or "cars.color" from the generated values map.
// It supports:
// - Simple keys: "color" -> returns generatedValues["color"]
// - Nested objects: "car.color" -> returns generatedValues["car"]["color"]
// - Array fields: "cars.color" -> returns []interface{} of all color values from cars array
func ResolveFieldPath(fieldPath string, generatedValues map[string]interface{}) (interface{}, bool) {
	// Split the path by dots
	parts := strings.Split(fieldPath, ".")

	if len(parts) == 0 {
		return nil, false
	}

	// Start with the root field
	current, exists := generatedValues[parts[0]]
	if !exists {
		return nil, false
	}

	// If it's a simple key (no dots), return immediately
	if len(parts) == 1 {
		return current, true
	}

	// Navigate through nested structure
	for i := 1; i < len(parts); i++ {
		part := parts[i]

		switch val := current.(type) {
		case map[string]interface{}:
			// Navigate into nested object
			next, exists := val[part]
			if !exists {
				return nil, false
			}
			current = next

		case []interface{}:
			// Extract field from all array items
			results := make([]interface{}, 0, len(val))
			for _, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if fieldValue, exists := itemMap[part]; exists {
						results = append(results, fieldValue)
					}
				}
			}
			// If this is the last part, return the array of values
			if i == len(parts)-1 {
				return results, len(results) > 0
			}
			// Otherwise, we can't navigate further into an array of values
			return nil, false

		default:
			// Can't navigate further - not a map or array
			return nil, false
		}
	}

	return current, true
}

// FormatFieldValue formats a field value for inclusion in a prompt.
// It handles different types appropriately:
// - Arrays: formats as comma-separated list or bullet points
// - Maps: formats as JSON-like structure
// - Primitives: converts to string
func FormatFieldValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		// Format array as bullet points for better readability
		if len(v) == 0 {
			return "[]"
		}
		var builder strings.Builder
		for i, item := range v {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("  - %v", item))
		}
		return builder.String()

	case map[string]interface{}:
		// Format map as key-value pairs
		if len(v) == 0 {
			return "{}"
		}
		var builder strings.Builder
		builder.WriteString("{\n")
		for key, val := range v {
			builder.WriteString(fmt.Sprintf("  %s: %v\n", key, val))
		}
		builder.WriteString("}")
		return builder.String()

	default:
		return fmt.Sprintf("%v", v)
	}
}

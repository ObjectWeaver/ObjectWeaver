package execution

import (
	"strings"
	"testing"
)

func TestResolveFieldPath(t *testing.T) {
	tests := []struct {
		name           string
		fieldPath      string
		generatedVals  map[string]interface{}
		expectedValue  interface{}
		expectedExists bool
	}{
		{
			name:      "Simple field",
			fieldPath: "color",
			generatedVals: map[string]interface{}{
				"color": "red",
			},
			expectedValue:  "red",
			expectedExists: true,
		},
		{
			name:      "Nested object field",
			fieldPath: "car.color",
			generatedVals: map[string]interface{}{
				"car": map[string]interface{}{
					"color": "blue",
					"brand": "Toyota",
				},
			},
			expectedValue:  "blue",
			expectedExists: true,
		},
		{
			name:      "Deeply nested field",
			fieldPath: "user.address.city",
			generatedVals: map[string]interface{}{
				"user": map[string]interface{}{
					"address": map[string]interface{}{
						"city":  "New York",
						"state": "NY",
						"zip":   "10001",
					},
				},
			},
			expectedValue:  "New York",
			expectedExists: true,
		},
		{
			name:      "Array field extraction",
			fieldPath: "cars.color",
			generatedVals: map[string]interface{}{
				"cars": []interface{}{
					map[string]interface{}{"color": "red", "brand": "Toyota"},
					map[string]interface{}{"color": "blue", "brand": "Honda"},
					map[string]interface{}{"color": "green", "brand": "Ford"},
				},
			},
			expectedValue:  []interface{}{"red", "blue", "green"},
			expectedExists: true,
		},
		{
			name:      "Non-existent simple field",
			fieldPath: "missing",
			generatedVals: map[string]interface{}{
				"color": "red",
			},
			expectedValue:  nil,
			expectedExists: false,
		},
		{
			name:      "Non-existent nested field",
			fieldPath: "car.missing",
			generatedVals: map[string]interface{}{
				"car": map[string]interface{}{
					"color": "blue",
				},
			},
			expectedValue:  nil,
			expectedExists: false,
		},
		{
			name:      "Non-existent parent object",
			fieldPath: "missing.color",
			generatedVals: map[string]interface{}{
				"car": map[string]interface{}{
					"color": "blue",
				},
			},
			expectedValue:  nil,
			expectedExists: false,
		},
		{
			name:      "Empty array field extraction",
			fieldPath: "cars.color",
			generatedVals: map[string]interface{}{
				"cars": []interface{}{},
			},
			expectedValue:  nil,
			expectedExists: false,
		},
		{
			name:      "Array with partial matches",
			fieldPath: "items.name",
			generatedVals: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"name": "Item1", "price": 10},
					"not a map",
					map[string]interface{}{"name": "Item2", "price": 20},
				},
			},
			expectedValue:  []interface{}{"Item1", "Item2"},
			expectedExists: true,
		},
		{
			name:      "Invalid path through primitive",
			fieldPath: "color.shade",
			generatedVals: map[string]interface{}{
				"color": "red",
			},
			expectedValue:  nil,
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, exists := ResolveFieldPath(tt.fieldPath, tt.generatedVals)

			if exists != tt.expectedExists {
				t.Errorf("ResolveFieldPath(%q) exists = %v, want %v", tt.fieldPath, exists, tt.expectedExists)
			}

			if !exists {
				return
			}

			// Special handling for array comparison
			if expectedArr, ok := tt.expectedValue.([]interface{}); ok {
				actualArr, ok := value.([]interface{})
				if !ok {
					t.Errorf("Expected array, got %T", value)
					return
				}

				if len(actualArr) != len(expectedArr) {
					t.Errorf("Array length mismatch: got %d, want %d", len(actualArr), len(expectedArr))
					return
				}

				for i := range expectedArr {
					if actualArr[i] != expectedArr[i] {
						t.Errorf("Array element %d: got %v, want %v", i, actualArr[i], expectedArr[i])
					}
				}
			} else {
				if value != tt.expectedValue {
					t.Errorf("ResolveFieldPath(%q) = %v, want %v", tt.fieldPath, value, tt.expectedValue)
				}
			}
		})
	}
}

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name          string
		value         interface{}
		expectedParts []string // Parts that should be in the output
	}{
		{
			name:          "Simple string",
			value:         "hello",
			expectedParts: []string{"hello"},
		},
		{
			name:          "Simple number",
			value:         42,
			expectedParts: []string{"42"},
		},
		{
			name:          "Empty array",
			value:         []interface{}{},
			expectedParts: []string{"[]"},
		},
		{
			name:          "Array of strings",
			value:         []interface{}{"apple", "banana", "orange"},
			expectedParts: []string{"- apple", "- banana", "- orange"},
		},
		{
			name:          "Array of numbers",
			value:         []interface{}{1, 2, 3},
			expectedParts: []string{"- 1", "- 2", "- 3"},
		},
		{
			name:          "Empty map",
			value:         map[string]interface{}{},
			expectedParts: []string{"{}"},
		},
		{
			name: "Simple map",
			value: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			expectedParts: []string{"name:", "John", "age:", "30"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFieldValue(tt.value)

			for _, part := range tt.expectedParts {
				if !strings.Contains(result, part) {
					t.Errorf("FormatFieldValue() result missing expected part %q.\nGot: %s", part, result)
				}
			}
		})
	}
}

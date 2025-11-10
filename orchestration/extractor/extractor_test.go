package extractor

import (
	"strings"
	"testing"
)

func TestDefaultExtractor_ExtractAndJoin(t *testing.T) {
	extractor := NewDefaultExtractor()

	tests := []struct {
		name       string
		currentGen map[string]any
		keys       []string
		expected   string
	}{
		{
			name: "Extract string values",
			currentGen: map[string]any{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			keys:     []string{"key1", "key2", "key3"},
			expected: "value1\nvalue2\nvalue3",
		},
		{
			name: "Extract subset of keys",
			currentGen: map[string]any{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			keys:     []string{"key1", "key3"},
			expected: "value1\nvalue3",
		},
		{
			name: "Extract with non-existent keys",
			currentGen: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			keys:     []string{"key1", "key3", "key2"},
			expected: "value1\nvalue2",
		},
		{
			name: "Extract integer values",
			currentGen: map[string]any{
				"count":  42,
				"age":    25,
				"status": 200,
			},
			keys:     []string{"count", "age", "status"},
			expected: "42\n25\n200",
		},
		{
			name: "Extract boolean values",
			currentGen: map[string]any{
				"active":  true,
				"enabled": false,
			},
			keys:     []string{"active", "enabled"},
			expected: "true\nfalse",
		},
		{
			name: "Extract float values",
			currentGen: map[string]any{
				"price":       19.99,
				"temperature": -5.5,
			},
			keys:     []string{"price", "temperature"},
			expected: "19.99\n-5.5",
		},
		{
			name: "Extract mixed types",
			currentGen: map[string]any{
				"name":   "Alice",
				"age":    30,
				"active": true,
				"score":  95.5,
			},
			keys:     []string{"name", "age", "active", "score"},
			expected: "Alice\n30\ntrue\n95.5",
		},
		{
			name:       "Empty map",
			currentGen: map[string]any{},
			keys:       []string{"key1", "key2"},
			expected:   "",
		},
		{
			name: "Empty keys slice",
			currentGen: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			keys:     []string{},
			expected: "",
		},
		{
			name:       "Empty map and empty keys",
			currentGen: map[string]any{},
			keys:       []string{},
			expected:   "",
		},
		{
			name: "Skip empty string values",
			currentGen: map[string]any{
				"key1": "value1",
				"key2": "",
				"key3": "value3",
			},
			keys:     []string{"key1", "key2", "key3"},
			expected: "value1\n\nvalue3",
		},
		{
			name: "Skip empty map values",
			currentGen: map[string]any{
				"key1":     "value1",
				"emptyMap": map[string]any{},
				"key3":     "value3",
			},
			keys:     []string{"key1", "emptyMap", "key3"},
			expected: "value1\nvalue3",
		},
		{
			name: "Extract nil values (formatted as <nil>)",
			currentGen: map[string]any{
				"key1": "value1",
				"key2": nil,
				"key3": "value3",
			},
			keys:     []string{"key1", "key2", "key3"},
			expected: "value1\n<nil>\nvalue3",
		},
		{
			name: "Extract slice values",
			currentGen: map[string]any{
				"items": []string{"item1", "item2", "item3"},
				"nums":  []int{1, 2, 3},
			},
			keys:     []string{"items", "nums"},
			expected: "[item1 item2 item3]\n[1 2 3]",
		},
		{
			name: "Extract nested map values",
			currentGen: map[string]any{
				"user": map[string]string{
					"name": "Alice",
					"city": "NYC",
				},
				"age": 30,
			},
			keys:     []string{"user", "age"},
			expected: "map[city:NYC name:Alice]\n30",
		},
		{
			name: "Extract struct values",
			currentGen: map[string]any{
				"point": struct {
					X int
					Y int
				}{X: 10, Y: 20},
			},
			keys:     []string{"point"},
			expected: "{10 20}",
		},
		{
			name: "Keys in different order",
			currentGen: map[string]any{
				"a": "alpha",
				"b": "beta",
				"c": "gamma",
			},
			keys:     []string{"c", "a", "b"},
			expected: "gamma\nalpha\nbeta",
		},
		{
			name: "Duplicate keys",
			currentGen: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			keys:     []string{"key1", "key2", "key1"},
			expected: "value1\nvalue2\nvalue1",
		},
		{
			name: "Extract with zero values",
			currentGen: map[string]any{
				"zero_int":    0,
				"zero_float":  0.0,
				"zero_string": "",
				"false_bool":  false,
			},
			keys:     []string{"zero_int", "zero_float", "zero_string", "false_bool"},
			expected: "0\n0\n\nfalse",
		},
		{
			name: "Extract with special characters in values",
			currentGen: map[string]any{
				"text1": "Hello\nWorld",
				"text2": "Tab\there",
				"text3": "Quote: \"test\"",
			},
			keys:     []string{"text1", "text2", "text3"},
			expected: "Hello\nWorld\nTab\there\nQuote: \"test\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ExtractAndJoin(tt.currentGen, tt.keys)

			if result != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestNewDefaultExtractor(t *testing.T) {
	extractor := NewDefaultExtractor()

	if extractor == nil {
		t.Error("NewDefaultExtractor returned nil")
	}

	// Verify it implements the Extractor interface
	var _ Extractor = extractor

	// Test that it works
	result := extractor.ExtractAndJoin(
		map[string]any{"key": "value"},
		[]string{"key"},
	)
	if result != "value" {
		t.Errorf("Expected 'value' but got %q", result)
	}
}

func TestDefaultExtractor_PreservesOrder(t *testing.T) {
	extractor := NewDefaultExtractor()

	currentGen := map[string]any{
		"z": "last",
		"a": "first",
		"m": "middle",
	}

	// The order should be determined by the keys slice, not the map
	keys := []string{"a", "m", "z"}
	result := extractor.ExtractAndJoin(currentGen, keys)
	expected := "first\nmiddle\nlast"

	if result != expected {
		t.Errorf("Order not preserved. Expected %q, got %q", expected, result)
	}
}

func TestDefaultExtractor_LargeDataset(t *testing.T) {
	extractor := NewDefaultExtractor()

	// Create a large map
	currentGen := make(map[string]any)
	keys := make([]string, 1000)

	for i := 0; i < 1000; i++ {
		key := string(rune('a'+(i%26))) + string(rune('0'+(i/26)))
		currentGen[key] = i
		keys[i] = key
	}

	result := extractor.ExtractAndJoin(currentGen, keys)

	// Verify the result contains all values
	lines := strings.Split(result, "\n")
	if len(lines) != 1000 {
		t.Errorf("Expected 1000 lines, got %d", len(lines))
	}
}

func TestDefaultExtractor_ComplexTypes(t *testing.T) {
	extractor := NewDefaultExtractor()

	type CustomStruct struct {
		Name  string
		Value int
	}

	tests := []struct {
		name       string
		currentGen map[string]any
		keys       []string
		expected   string
	}{
		{
			name: "Pointer to struct",
			currentGen: map[string]any{
				"ptr": &CustomStruct{Name: "test", Value: 42},
			},
			keys:     []string{"ptr"},
			expected: "&{test 42}",
		},
		{
			name: "Map of maps",
			currentGen: map[string]any{
				"nested": map[string]map[string]int{
					"outer": {"inner": 10},
				},
			},
			keys:     []string{"nested"},
			expected: "map[outer:map[inner:10]]",
		},
		{
			name: "Channel (should be formatted)",
			currentGen: map[string]any{
				"text":  "before",
				"chan":  make(chan int),
				"after": "after",
			},
			keys:     []string{"text", "chan", "after"},
			expected: "", // Will be validated differently
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ExtractAndJoin(tt.currentGen, tt.keys)

			if tt.name == "Channel (should be formatted)" {
				// Just verify it doesn't panic and returns something
				if !strings.Contains(result, "before") || !strings.Contains(result, "after") {
					t.Errorf("Expected result to contain 'before' and 'after', got: %q", result)
				}
			} else if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDefaultExtractor_Stateless(t *testing.T) {
	extractor := NewDefaultExtractor()

	// First extraction
	result1 := extractor.ExtractAndJoin(
		map[string]any{"key": "value1"},
		[]string{"key"},
	)

	// Second extraction with different data
	result2 := extractor.ExtractAndJoin(
		map[string]any{"key": "value2"},
		[]string{"key"},
	)

	if result1 != "value1" {
		t.Errorf("First extraction failed: expected 'value1', got %q", result1)
	}

	if result2 != "value2" {
		t.Errorf("Second extraction failed: expected 'value2', got %q", result2)
	}

	// Verify they're different (stateless behavior)
	if result1 == result2 {
		t.Error("Extractor appears to maintain state between calls")
	}
}

func TestDefaultExtractor_EmptyMapValue(t *testing.T) {
	extractor := NewDefaultExtractor()

	currentGen := map[string]any{
		"before":   "value1",
		"emptyMap": map[string]any{},
		"after":    "value2",
	}

	result := extractor.ExtractAndJoin(currentGen, []string{"before", "emptyMap", "after"})
	expected := "value1\nvalue2"

	if result != expected {
		t.Errorf("Empty map not skipped correctly. Expected %q, got %q", expected, result)
	}
}

func TestDefaultExtractor_WhitespaceInStrings(t *testing.T) {
	extractor := NewDefaultExtractor()

	currentGen := map[string]any{
		"spaces":   "  value with spaces  ",
		"tabs":     "\tvalue with tabs\t",
		"newlines": "\nvalue with newlines\n",
	}

	result := extractor.ExtractAndJoin(currentGen, []string{"spaces", "tabs", "newlines"})

	// Whitespace should be preserved
	if !strings.Contains(result, "  value with spaces  ") {
		t.Error("Spaces not preserved")
	}
	if !strings.Contains(result, "\tvalue with tabs\t") {
		t.Error("Tabs not preserved")
	}
	if !strings.Contains(result, "\nvalue with newlines\n") {
		t.Error("Newlines not preserved")
	}
}

// Benchmark tests
func BenchmarkDefaultExtractor_SmallMap(b *testing.B) {
	extractor := NewDefaultExtractor()
	currentGen := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	keys := []string{"key1", "key2", "key3"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.ExtractAndJoin(currentGen, keys)
	}
}

func BenchmarkDefaultExtractor_LargeMap(b *testing.B) {
	extractor := NewDefaultExtractor()
	currentGen := make(map[string]any)
	keys := make([]string, 100)

	for i := 0; i < 100; i++ {
		key := string(rune('a'+(i%26))) + string(rune('0'+(i/26)))
		currentGen[key] = i
		keys[i] = key
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.ExtractAndJoin(currentGen, keys)
	}
}

func BenchmarkDefaultExtractor_MixedTypes(b *testing.B) {
	extractor := NewDefaultExtractor()
	currentGen := map[string]any{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"slice":  []int{1, 2, 3},
	}
	keys := []string{"string", "int", "float", "bool", "slice"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.ExtractAndJoin(currentGen, keys)
	}
}

func BenchmarkDefaultExtractor_NonExistentKeys(b *testing.B) {
	extractor := NewDefaultExtractor()
	currentGen := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}
	keys := []string{"key1", "nonexistent", "key2", "missing"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.ExtractAndJoin(currentGen, keys)
	}
}

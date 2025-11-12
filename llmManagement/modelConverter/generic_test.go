package modelConverter

import (
	"testing"
)

func TestNewModelConverter(t *testing.T) {
	converter := NewModelConverter()
	if converter == nil {
		t.Fatal("Expected non-nil ModelConverter")
	}

	// Verify it implements the interface
	var _ ModelConverter = converter
}

func TestProviderModelConverter_Convert(t *testing.T) {
	converter := NewModelConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple model name",
			input:    "gpt-4",
			expected: "gpt-4",
		},
		{
			name:     "model with version",
			input:    "gpt-4-turbo",
			expected: "gpt-4-turbo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "custom model",
			input:    "my-custom-model",
			expected: "my-custom-model",
		},
		{
			name:     "model with special characters",
			input:    "model_v1.0-beta",
			expected: "model_v1.0-beta",
		},
		{
			name:     "long model name",
			input:    "very-long-model-name-with-many-parts",
			expected: "very-long-model-name-with-many-parts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if result != tt.expected {
				t.Errorf("Convert(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProviderModelConverterDirectUsage(t *testing.T) {
	// Test direct struct usage
	converter := &providerModelConverter{}

	result := converter.Convert("test-model")
	if result != "test-model" {
		t.Errorf("Expected 'test-model', got '%s'", result)
	}
}

func TestProviderModelConverterInterface(t *testing.T) {
	// Verify the struct implements the interface correctly
	var converter ModelConverter = &providerModelConverter{}

	models := []string{"gpt-3.5-turbo", "gpt-4", "claude-3", ""}
	for _, model := range models {
		result := converter.Convert(model)
		if result != model {
			t.Errorf("Interface method Convert(%q) = %q, expected %q", model, result, model)
		}
	}
}

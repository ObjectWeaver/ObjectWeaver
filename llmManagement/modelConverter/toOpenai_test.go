package modelConverter

import (
	"testing"
)

func TestNewOpenAiModelConverter(t *testing.T) {
	converter := NewOpenAiModelConverter()
	if converter == nil {
		t.Fatal("Expected non-nil ModelConverter")
	}

	// Verify it implements the interface
	var _ ModelConverter = converter
}

func TestOpenAiModelConverter_Convert(t *testing.T) {
	converter := NewOpenAiModelConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "gpt-4 model",
			input:    "gpt-4",
			expected: "gpt-4-turbo",
		},
		{
			name:     "gpt-3.5 model",
			input:    "gpt-3.5-turbo",
			expected: "gpt-4-turbo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "gpt-4-turbo",
		},
		{
			name:     "custom model",
			input:    "my-custom-model",
			expected: "gpt-4-turbo",
		},
		{
			name:     "any input returns gpt-4-turbo",
			input:    "whatever-model",
			expected: "gpt-4-turbo",
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

func TestOpenAiModelConverterDirectUsage(t *testing.T) {
	// Test direct struct usage
	converter := &OpenAiModelConverter{}
	
	result := converter.Convert("test-model")
	if result != "gpt-4-turbo" {
		t.Errorf("Expected 'gpt-4-turbo', got '%s'", result)
	}
}

func TestOpenAiModelConverterInterface(t *testing.T) {
	// Verify the struct implements the interface correctly
	var converter ModelConverter = &OpenAiModelConverter{}
	
	models := []string{"gpt-3.5-turbo", "gpt-4", "claude-3", ""}
	for _, model := range models {
		result := converter.Convert(model)
		if result != "gpt-4-turbo" {
			t.Errorf("Interface method Convert(%q) = %q, expected 'gpt-4-turbo'", model, result)
		}
	}
}

func TestOpenAiModelConverterConsistency(t *testing.T) {
	converter := NewOpenAiModelConverter()
	
	// Call multiple times with different inputs to verify consistency
	inputs := []string{"model1", "model2", "model3", ""}
	for _, input := range inputs {
		result := converter.Convert(input)
		if result != "gpt-4-turbo" {
			t.Errorf("Convert should always return 'gpt-4-turbo', got %q for input %q", result, input)
		}
	}
}

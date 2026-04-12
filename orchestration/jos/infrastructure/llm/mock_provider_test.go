package llm

import (
	"objectweaver/orchestration/jos/domain"
	"strings"
	"testing"

	"objectweaver/jsonSchema"
)

func TestNewMockProvider(t *testing.T) {
	provider := NewMockProvider()

	if provider == nil {
		t.Fatal("Expected non-nil MockProvider")
	}

	if provider.rng == nil {
		t.Error("Expected rng to be initialized")
	}
}

func TestMockProvider_Generate(t *testing.T) {
	provider := NewMockProvider()

	tests := []struct {
		name   string
		prompt string
		config *domain.GenerationConfig
	}{
		{
			name:   "nil config",
			prompt: "test prompt",
			config: nil,
		},
		{
			name:   "config with no definition",
			prompt: "test",
			config: &domain.GenerationConfig{},
		},
		{
			name:   "config with string type",
			prompt: "generate string",
			config: &domain.GenerationConfig{
				Definition: &jsonSchema.Definition{Type: jsonSchema.String},
			},
		},
		{
			name:   "config with integer type",
			prompt: "generate number",
			config: &domain.GenerationConfig{
				Definition: &jsonSchema.Definition{Type: jsonSchema.Integer},
			},
		},
		{
			name:   "config with number type",
			prompt: "generate float",
			config: &domain.GenerationConfig{
				Definition: &jsonSchema.Definition{Type: jsonSchema.Number},
			},
		},
		{
			name:   "config with boolean type",
			prompt: "generate bool",
			config: &domain.GenerationConfig{
				Definition: &jsonSchema.Definition{Type: jsonSchema.Boolean},
			},
		},
		{
			name:   "config with array type",
			prompt: "generate array",
			config: &domain.GenerationConfig{
				Definition: &jsonSchema.Definition{Type: jsonSchema.Array},
			},
		},
		{
			name:   "config with object type",
			prompt: "generate object",
			config: &domain.GenerationConfig{
				Definition: &jsonSchema.Definition{Type: jsonSchema.Object},
			},
		},
		{
			name:   "config with map type",
			prompt: "generate map",
			config: &domain.GenerationConfig{
				Definition: &jsonSchema.Definition{Type: jsonSchema.Map},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata, err := provider.Generate(tt.prompt, tt.config)

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if result == nil {
				t.Error("Expected non-nil result")
			}

			if metadata == nil {
				t.Error("Expected non-nil metadata")
			}
		})
	}
}

func TestMockProvider_GenerateForType(t *testing.T) {
	provider := NewMockProvider()

	tests := []struct {
		name         string
		def          *jsonSchema.Definition
		validateFunc func(string) bool
	}{
		{
			name: "string type",
			def:  &jsonSchema.Definition{Type: jsonSchema.String},
			validateFunc: func(s string) bool {
				return len(s) > 0
			},
		},
		{
			name: "integer type",
			def:  &jsonSchema.Definition{Type: jsonSchema.Integer},
			validateFunc: func(s string) bool {
				// Should be a number string
				return len(s) > 0 && s[0] >= '0' && s[0] <= '9'
			},
		},
		{
			name: "number type",
			def:  &jsonSchema.Definition{Type: jsonSchema.Number},
			validateFunc: func(s string) bool {
				// Should contain a decimal point for float
				return strings.Contains(s, ".") || (len(s) > 0 && s[0] >= '0')
			},
		},
		{
			name: "boolean type",
			def:  &jsonSchema.Definition{Type: jsonSchema.Boolean},
			validateFunc: func(s string) bool {
				return s == "true" || s == "false"
			},
		},
		{
			name: "array type",
			def:  &jsonSchema.Definition{Type: jsonSchema.Array},
			validateFunc: func(s string) bool {
				return s == "[]"
			},
		},
		{
			name: "object type",
			def:  &jsonSchema.Definition{Type: jsonSchema.Object},
			validateFunc: func(s string) bool {
				return s == "{}"
			},
		},
		{
			name: "map type",
			def:  &jsonSchema.Definition{Type: jsonSchema.Map},
			validateFunc: func(s string) bool {
				return s == "{}"
			},
		},
		{
			name: "unknown type",
			def:  &jsonSchema.Definition{Type: "unknown"},
			validateFunc: func(s string) bool {
				return len(s) > 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.generateForType(tt.def)

			if !tt.validateFunc(result) {
				t.Errorf("Generated value '%s' failed validation for type %s", result, tt.def.Type)
			}
		})
	}
}

func TestMockProvider_RandomString(t *testing.T) {
	provider := NewMockProvider()

	lengths := []int{5, 10, 20, 50, 100}

	for _, length := range lengths {
		t.Run(string(rune(length)), func(t *testing.T) {
			result := provider.randomString(length)

			if len(result) != length {
				t.Errorf("Expected string of length %d, got %d", length, len(result))
			}

			// Verify all characters are alphanumeric
			for _, ch := range result {
				if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
					t.Errorf("String contains non-alphanumeric character: %c", ch)
				}
			}
		})
	}
}

func TestMockProvider_RandomStringUniqueness(t *testing.T) {
	provider := NewMockProvider()

	// Generate multiple random strings and check they're different
	strings := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s := provider.randomString(10)
		strings[s] = true
	}

	// We should have generated many unique strings
	if len(strings) < 50 {
		t.Errorf("Expected at least 50 unique strings, got %d", len(strings))
	}
}

func TestMockProvider_Metadata(t *testing.T) {
	provider := NewMockProvider()

	metadata := provider.metadata()

	if metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if metadata.TokensUsed < 10 {
		t.Errorf("Expected tokens used >= 10, got %d", metadata.TokensUsed)
	}

	if metadata.Cost != 0.001 {
		t.Errorf("Expected cost to be 0.001, got %f", metadata.Cost)
	}

	if metadata.Model != "mock-model" {
		t.Errorf("Expected model to be 'mock-model', got %s", metadata.Model)
	}

	if metadata.FinishReason != "stop" {
		t.Errorf("Expected finish reason to be 'stop', got %s", metadata.FinishReason)
	}
}

func TestMockProvider_SupportsStreaming(t *testing.T) {
	provider := NewMockProvider()

	if provider.SupportsStreaming() {
		t.Error("MockProvider should not support streaming")
	}
}

func TestMockProvider_ModelType(t *testing.T) {
	provider := NewMockProvider()

	modelType := provider.ModelType()
	if modelType != "mock" {
		t.Errorf("Expected model type 'mock', got %s", modelType)
	}
}

func TestMockProvider_GenerateMultipleTimes(t *testing.T) {
	provider := NewMockProvider()

	config := &domain.GenerationConfig{
		Definition: &jsonSchema.Definition{Type: jsonSchema.String},
	}

	// Generate multiple times to ensure consistency
	for i := 0; i < 10; i++ {
		result, metadata, err := provider.Generate("test", config)

		if err != nil {
			t.Errorf("Iteration %d: unexpected error: %v", i, err)
		}

		if result == nil {
			t.Errorf("Iteration %d: got nil result", i)
		}

		if metadata == nil {
			t.Errorf("Iteration %d: got nil metadata", i)
		}
	}
}

func TestMockProvider_AllTypes(t *testing.T) {
	provider := NewMockProvider()

	types := []jsonSchema.DataType{
		jsonSchema.String,
		jsonSchema.Integer,
		jsonSchema.Number,
		jsonSchema.Boolean,
		jsonSchema.Array,
		jsonSchema.Object,
		jsonSchema.Map,
	}

	for _, schemaType := range types {
		t.Run(string(schemaType), func(t *testing.T) {
			config := &domain.GenerationConfig{
				Definition: &jsonSchema.Definition{Type: schemaType},
			}

			result, _, err := provider.Generate("test", config)

			if err != nil {
				t.Errorf("Error generating for type %s: %v", schemaType, err)
			}

			if result == nil {
				t.Errorf("Got nil result for type %s", schemaType)
			}
		})
	}
}

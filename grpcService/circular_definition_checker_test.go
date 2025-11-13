package grpcService

import (
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

func TestDefaultCircularDefinitionChecker_Check(t *testing.T) {
	checker := NewDefaultCircularDefinitionChecker()

	tests := []struct {
		name       string
		definition *jsonSchema.Definition
		hasCircular bool
	}{
		{
			name: "simple definition without circular reference",
			definition: &jsonSchema.Definition{
				Type: "object",
				Properties: map[string]jsonSchema.Definition{
					"name": {Type: "string"},
					"age":  {Type: "number"},
				},
			},
			hasCircular: false,
		},
		{
			name: "nil definition",
			definition: nil,
			hasCircular: false,
		},
		{
			name: "nested definition without circular reference",
			definition: &jsonSchema.Definition{
				Type: "object",
				Properties: map[string]jsonSchema.Definition{
					"person": {
						Type: "object",
						Properties: map[string]jsonSchema.Definition{
							"name": {Type: "string"},
						},
					},
				},
			},
			hasCircular: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.Check(tt.definition)
			if result != tt.hasCircular {
				t.Errorf("Expected hasCircular=%v, got %v", tt.hasCircular, result)
			}
		})
	}
}

func TestDefaultCircularDefinitionChecker_ImplementsInterface(t *testing.T) {
	var _ CircularDefinitionChecker = NewDefaultCircularDefinitionChecker()
}

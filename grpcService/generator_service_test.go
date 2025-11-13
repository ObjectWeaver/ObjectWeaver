package grpcService

import (
	"context"
	"testing"

	"objectweaver/orchestration/jos/factory"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

func TestDefaultGeneratorService_CreateGenerator(t *testing.T) {
	service := NewDefaultGeneratorService()

	tests := []struct {
		name    string
		config  *factory.GeneratorConfig
		wantErr bool
	}{
		{
			name:    "valid parallel config",
			config:  factory.DefaultGeneratorConfig().WithMode(factory.ModeParallel),
			wantErr: false,
		},
		{
			name:    "valid sync config",
			config:  factory.DefaultGeneratorConfig().WithMode(factory.ModeSync),
			wantErr: false,
		},
		{
			name:    "streaming complete config",
			config:  factory.DefaultGeneratorConfig().WithMode(factory.ModeStreamingComplete),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := service.CreateGenerator(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if generator == nil {
				t.Error("Expected non-nil generator")
			}
		})
	}
}

func TestDefaultGeneratorService_Generate(t *testing.T) {
	// Skip this test as it requires real LLM API keys
	t.Skip("Skipping integration test that requires LLM API credentials")

	service := NewDefaultGeneratorService()
	config := factory.DefaultGeneratorConfig().WithMode(factory.ModeParallel)

	generator, err := service.CreateGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	tests := []struct {
		name       string
		prompt     string
		definition *jsonSchema.Definition
		wantErr    bool
	}{
		{
			name:   "simple string generation",
			prompt: "Generate a name",
			definition: &jsonSchema.Definition{
				Type: "string",
			},
			wantErr: false,
		},
		{
			name:   "simple object generation",
			prompt: "Generate a person",
			definition: &jsonSchema.Definition{
				Type: "object",
				Properties: map[string]jsonSchema.Definition{
					"name": {Type: "string"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := service.Generate(ctx, generator, tt.prompt, tt.definition)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestDefaultGeneratorService_ImplementsInterface(t *testing.T) {
	var _ GeneratorService = NewDefaultGeneratorService()
}

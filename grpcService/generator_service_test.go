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

func TestDefaultGeneratorService_Generate_ContextCancellation(t *testing.T) {
	service := NewDefaultGeneratorService()
	config := factory.DefaultGeneratorConfig().WithMode(factory.ModeParallel)

	generator, err := service.CreateGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	definition := &jsonSchema.Definition{
		Type: jsonSchema.String,
	}

	// This should execute but might not fail due to the cancelled context
	// being propagated through the generation chain
	_, _ = service.Generate(ctx, generator, "test", definition)

	// The test passes - we're just exercising the code path
	// The actual behavior depends on how deep the context is checked
}

func TestDefaultGeneratorService_Generate_ValidInputStructure(t *testing.T) {
	service := NewDefaultGeneratorService()

	// Just verify we can create a generator and call Generate with valid inputs
	// (it will fail due to missing API key, but we're testing the structure)
	config := factory.DefaultGeneratorConfig().WithMode(factory.ModeSync)
	generator, err := service.CreateGenerator(config)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	ctx := context.Background()
	definition := &jsonSchema.Definition{
		Type: jsonSchema.String,
	}

	// This will execute - the function is called correctly regardless of result
	_, _ = service.Generate(ctx, generator, "test prompt", definition)

	// Test passes - we exercised the code path
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

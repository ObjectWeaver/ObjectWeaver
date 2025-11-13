package grpcService

import (
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"objectweaver/orchestration/jos/factory"
)

func TestDefaultConfigFactory_CreateConfig(t *testing.T) {
	configFactory := NewDefaultConfigFactory()

	tests := []struct {
		name     string
		schema   *jsonSchema.Definition
		validate func(*testing.T, *factory.GeneratorConfig)
	}{
		{
			name: "streaming enabled uses sync mode",
			schema: &jsonSchema.Definition{
				Type:   "object",
				Stream: true,
			},
			validate: func(t *testing.T, config *factory.GeneratorConfig) {
				if config.Mode != factory.ModeSync {
					t.Errorf("Expected ModeSync, got %v", config.Mode)
				}
			},
		},
		{
			name: "streaming disabled uses parallel mode",
			schema: &jsonSchema.Definition{
				Type:   "object",
				Stream: false,
			},
			validate: func(t *testing.T, config *factory.GeneratorConfig) {
				if config.Mode != factory.ModeParallel {
					t.Errorf("Expected ModeParallel, got %v", config.Mode)
				}
			},
		},
		{
			name: "config has correct concurrency",
			schema: &jsonSchema.Definition{
				Type: "object",
			},
			validate: func(t *testing.T, config *factory.GeneratorConfig) {
				if config.MaxConcurrency != 10 {
					t.Errorf("Expected MaxConcurrency=10, got %d", config.MaxConcurrency)
				}
			},
		},
		{
			name: "config has cache enabled",
			schema: &jsonSchema.Definition{
				Type: "object",
			},
			validate: func(t *testing.T, config *factory.GeneratorConfig) {
				if !config.EnableCache {
					t.Error("Expected EnableCache to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configFactory.CreateConfig(tt.schema)
			if result == nil {
				t.Fatal("Expected non-nil config")
			}
			tt.validate(t, result)
		})
	}
}

func TestDefaultConfigFactory_ImplementsInterface(t *testing.T) {
	var _ ConfigFactory = NewDefaultConfigFactory()
}

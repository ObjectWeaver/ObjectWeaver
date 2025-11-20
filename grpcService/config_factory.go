package grpcService

import (
	"objectweaver/orchestration/jos/factory"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// DefaultConfigFactory is the default implementation of ConfigFactory
type DefaultConfigFactory struct{}

// NewDefaultConfigFactory creates a new DefaultConfigFactory
func NewDefaultConfigFactory() ConfigFactory {
	return &DefaultConfigFactory{}
}

// CreateConfig creates a generator config based on schema settings
func (f *DefaultConfigFactory) CreateConfig(schema *jsonSchema.Definition) *factory.GeneratorConfig {
	config := factory.DefaultGeneratorConfig()

	// Check if streaming is enabled
	if schema.Stream {
		// For non-streaming gRPC calls, we still use sync mode
		// Streaming is handled by StreamGeneratedObjectsV2
		config.Mode = factory.ModeSync
	} else {
		// Use parallel mode (now uses recursive architecture internally)
		config.Mode = factory.ModeParallel
	}

	// Configure based on schema properties
	config.MaxConcurrency = 10
	config.EnableCache = true

	return config
}

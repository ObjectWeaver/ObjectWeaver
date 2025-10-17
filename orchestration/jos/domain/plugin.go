package domain

import "github.com/objectweaver/go-sdk/jsonSchema"

// ValidationPlugin - Validates results
type ValidationPlugin interface {
	Plugin
	Validate(result *GenerationResult, schema *jsonSchema.Definition) ([]ValidationError, error)
}

type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// Plugin - Base plugin interface
type Plugin interface {
	Name() string
	Version() string
	Initialize(config map[string]interface{}) error
}

// PreProcessorPlugin - Runs before generation
type PreProcessorPlugin interface {
	Plugin
	PreProcess(request *GenerationRequest) (*GenerationRequest, error)
}

// PostProcessorPlugin - Runs after generation
type PostProcessorPlugin interface {
	Plugin
	PostProcess(result *GenerationResult) (*GenerationResult, error)
}

// CachePlugin - Handles caching
type CachePlugin interface {
	Plugin
	Get(key string) (*GenerationResult, bool)
	Set(key string, result *GenerationResult) error
}

// ObservabilityPlugin - Handles metrics and tracing
type ObservabilityPlugin interface {
	Plugin
	RecordMetric(name string, value float64, tags map[string]string)
	StartSpan(name string) Span
}

// Span represents a tracing span
type Span interface {
	End()
	SetTag(key string, value interface{})
}

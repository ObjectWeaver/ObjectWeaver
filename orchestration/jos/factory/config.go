package factory

import "firechimp/orchestration/jos/domain"

// ProcessingMode defines the operational mode
type ProcessingMode int

const (
	ModeSync ProcessingMode = iota
	ModeAsync
	ModeParallel
	ModeDependencyAware
	ModeStreaming
	ModeStreamingComplete
	ModeStreamingProgressive
)

// GeneratorConfig configures generator creation
type GeneratorConfig struct {
	Mode                ProcessingMode
	MaxConcurrency      int
	EnableCache         bool
	EnableValidation    bool
	EnableObservability bool
	LLMProvider         string
	Plugins             []domain.Plugin
	Granularity         domain.StreamGranularity
	StreamChannel       chan<- *domain.StreamChunk
}

func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		Mode:                ModeParallel,
		MaxConcurrency:      10,
		EnableCache:         false,
		EnableValidation:    true,
		EnableObservability: false,
		LLMProvider:         "openai",
		Plugins:             make([]domain.Plugin, 0),
		Granularity:         domain.GranularityField,
	}
}

func (c *GeneratorConfig) WithMode(mode ProcessingMode) *GeneratorConfig {
	newConfig := *c
	newConfig.Mode = mode
	return &newConfig
}

func (c *GeneratorConfig) WithConcurrency(max int) *GeneratorConfig {
	newConfig := *c
	newConfig.MaxConcurrency = max
	return &newConfig
}

func (c *GeneratorConfig) WithCache(enabled bool) *GeneratorConfig {
	newConfig := *c
	newConfig.EnableCache = enabled
	return &newConfig
}

func (c *GeneratorConfig) WithPlugin(plugin domain.Plugin) *GeneratorConfig {
	newConfig := *c
	newConfig.Plugins = append(newConfig.Plugins, plugin)
	return &newConfig
}

func (c *GeneratorConfig) WithGranularity(granularity domain.StreamGranularity) *GeneratorConfig {
	newConfig := *c
	newConfig.Granularity = granularity
	return &newConfig
}
